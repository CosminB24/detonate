package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/CosminB24/detonate/internal/collect"
	"github.com/CosminB24/detonate/internal/report"
	"github.com/CosminB24/detonate/internal/runner"
	"github.com/CosminB24/detonate/internal/signature"
)

const usage = `Usage:
  detonate npm <package>                                analyse one package
  detonate diff [-runs N] npm <package>@<v1> <v2>       compare two versions

The package argument is a name, a name@version, or a local path.`

var errUsage = errors.New(usage)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "diff":
		err = runDiff(os.Args[2:])
	case "npm":
		err = runAnalyse(os.Args[1], os.Args[2:])
	default:
		err = errUsage
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// runAnalyse detonates a single package and writes out/report.json.
func runAnalyse(ecosystem string, args []string) error {
	if len(args) < 1 {
		return errUsage
	}

	target, err := resolve(args[0])
	if err != nil {
		return err
	}
	outDir, err := outRoot()
	if err != nil {
		return err
	}

	fmt.Printf("%-12s %s\n", "ecosystem:", ecosystem)
	fmt.Printf("%-12s %s\n", "target:", target)

	analysis, err := analyse(ecosystem, target, outDir)
	if err != nil {
		return err
	}

	fmt.Println()
	for kind, n := range analysis.Kinds {
		fmt.Printf("%-16s %d\n", kind, n)
	}
	fmt.Println()
	fmt.Printf("%-12s %s\n", "verdict:", analysis.Verdict)
	fmt.Println()
	if len(analysis.Findings) == 0 {
		fmt.Println("no findings")
	}
	for _, f := range analysis.Findings {
		fmt.Printf("[%s] %s (%d events)\n", f.Severity, f.Title, len(f.Evidence))
	}

	reportPath := filepath.Join(outDir, "report.json")
	if err := report.Write(reportPath, report.Report{
		SchemaVersion: "1.0",
		Analysis:      analysis,
	}); err != nil {
		return err
	}
	fmt.Printf("\n%-12s %s\n", "report:", reportPath)
	return nil
}

// runDiff detonates two versions of a package and reports what the second one
// does that the first did not.
func runDiff(args []string) error {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	runs := fs.Int("runs", 2, "detonations per version")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 3 {
		return errUsage
	}
	if *runs < 1 {
		return errors.New("-runs must be at least 1")
	}

	ecosystem := fs.Arg(0)
	if ecosystem != "npm" {
		return errors.New("only npm is supported")
	}
	fromTarget, err := resolve(fs.Arg(1))
	if err != nil {
		return err
	}
	toTarget, err := resolve(fs.Arg(2))
	if err != nil {
		return err
	}

	outDir, err := outRoot()
	if err != nil {
		return err
	}
	diffDir := filepath.Join(outDir, "diff")

	startedAt := time.Now()
	fmt.Printf("%-12s %s\n", "ecosystem:", ecosystem)
	fmt.Printf("%-12s %s\n", "from:", fromTarget)
	fmt.Printf("%-12s %s\n", "to:", toTarget)
	fmt.Printf("%-12s %d per version\n", "runs:", *runs)

	from, err := detonateSide(ecosystem, fromTarget, filepath.Join(diffDir, "from"), *runs)
	if err != nil {
		return err
	}
	to, err := detonateSide(ecosystem, toTarget, filepath.Join(diffDir, "to"), *runs)
	if err != nil {
		return err
	}

	changes := report.Compare(from.Behaviours, to.Behaviours)
	newFindings := report.NewFindings(from.Findings, to.Findings)

	fmt.Printf("\n=== %s → %s\n\n", fromTarget, toTarget)
	printSide("from:", from)
	printSide("to:", to)

	fmt.Println()
	printBehaviours("NEW", changes.New)
	printBehaviours("REMOVED", changes.Removed)
	fmt.Printf("%-12s %d\n", "unchanged:", len(changes.Unchanged))

	fmt.Println()
	if len(newFindings) == 0 {
		fmt.Println("no new findings")
	}
	for _, f := range newFindings {
		fmt.Printf("NEW FINDING [%s] %s (%d events)\n", f.Severity, f.Title, len(f.Evidence))
	}

	diffPath := filepath.Join(outDir, "diff.json")
	if err := report.WriteDiff(diffPath, report.DiffReport{
		SchemaVersion: "1.0",
		Diff: report.DiffAnalysis{
			Ecosystem:      ecosystem,
			StartedAt:      startedAt,
			RunsPerVersion: *runs,
			From:           from,
			To:             to,
			Changes:        changes,
			NewFindings:    newFindings,
		},
	}); err != nil {
		return err
	}
	fmt.Printf("\n%-12s %s\n", "diff:", diffPath)
	return nil
}

// detonateSide analyses one version several times and keeps only the
// behaviours every run agreed on. Repetition is what makes the comparison
// trustworthy: a behaviour that comes and goes between runs of the SAME
// version would otherwise be reported as a change between versions.
func detonateSide(ecosystem, target, sideDir string, runs int) (report.Side, error) {
	var sets [][]report.Behaviour
	var first report.Analysis

	for i := 1; i <= runs; i++ {
		fmt.Printf("\n--- %s  run %d/%d\n", target, i, runs)

		runDir := filepath.Join(sideDir, fmt.Sprintf("run-%d", i))
		analysis, err := analyse(ecosystem, target, runDir)
		if err != nil {
			return report.Side{}, err
		}
		if i == 1 {
			first = analysis
		}
		if analysis.Verdict != first.Verdict {
			fmt.Printf("warning: verdict differs between runs of %s: %s then %s\n",
				target, first.Verdict, analysis.Verdict)
		}
		sets = append(sets, analysis.Behaviours)

		// A diff detonates several times; without this, each run leaves a
		// root-owned node_modules tree behind that the host user cannot
		// delete. The trace stays — only the work directory goes.
		if err := runner.Clean(runDir); err != nil {
			return report.Side{}, err
		}
	}

	return report.Side{
		Target:     target,
		Verdict:    first.Verdict,
		Findings:   first.Findings,
		Behaviours: report.Stable(sets),
		Unstable:   report.Unstable(sets),
	}, nil
}

// analyse runs the whole pipeline for one target: detonate, parse, match.
// outDir must be unique per detonation — it holds both the work directory the
// two container phases share and the strace log.
func analyse(ecosystem, target, outDir string) (report.Analysis, error) {
	startedAt := time.Now()

	// The work directory is shared between the two container phases. Files in
	// it are created by root inside the container, so it must be removed from
	// a container too — the host user cannot delete them.
	workDir := filepath.Join(outDir, "work")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return report.Analysis{}, err
	}
	if err := runner.Clean(outDir); err != nil {
		return report.Analysis{}, err
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return report.Analysis{}, err
	}

	// What npm will install, as seen from inside the container.
	//
	// A named package is passed through untouched — phase 1 downloads it.
	// A local path is packed into a tarball first: installing a directory
	// creates a symlink ("file:" dependency), and npm never runs lifecycle
	// scripts for those. The tarball is written into the work directory
	// because that is what the container mounts.
	installArg := target
	if isLocalPath(target) {
		pack := exec.Command("npm", "pack", target, "--pack-destination", workDir)
		out, err := pack.Output()
		if err != nil {
			// npm's own message is the only useful part here: the common
			// failure is the host npm being the Windows one under WSL, which
			// mangles paths and would otherwise surface as a bare exit code.
			var exit *exec.ExitError
			if errors.As(err, &exit) {
				return report.Analysis{}, fmt.Errorf("npm pack %s: %w\n%s", target, err, exit.Stderr)
			}
			return report.Analysis{}, fmt.Errorf("npm pack %s: %w", target, err)
		}
		fileName := strings.TrimSpace(string(out))
		installArg = "/work/" + fileName
		fmt.Printf("%-12s %s\n", "packed:", fileName)
	}

	// Two phases: fetch with network but no execution, then execute offline
	// under strace. See internal/runner.
	const logName = "trace.log"
	if err := runner.Detonate(outDir, workDir, installArg, logName); err != nil {
		return report.Analysis{}, err
	}

	tracePath := filepath.Join(outDir, logName)
	fmt.Printf("%-12s %s\n", "trace:", tracePath)

	events, skipped, err := collect.Parse(tracePath)
	if err != nil {
		return report.Analysis{}, err
	}
	fmt.Printf("%-12s %d\n", "events:", len(events))
	fmt.Printf("%-12s %d\n", "skipped:", skipped)

	for _, e := range events {
		if strings.Contains(e.Raw, "resumed") {
			fmt.Printf("RESUMED syscall=%q kind=%s result=%q pid=%s\n",
				e.Syscall, e.Kind, e.Result, e.PID)
		}
	}

	tree := collect.ProcessTree(events)
	fmt.Println()
	for child, parent := range tree {
		fmt.Printf("  %s → %s\n", parent, child)
	}

	roots := collect.ScriptRoots(events)
	fmt.Printf("\nSCRIPT ROOTS: %v\n", roots)

	for _, e := range events {
		if e.Kind == "process.exec" && !e.Failed &&
			(strings.HasSuffix(e.Target, "/sh") || strings.HasSuffix(e.Target, "/bash")) {
			fmt.Printf("SHELL pid=%s phase=%s target=%s parent_phase=%s\n", e.PID, e.Phase, e.Target, e.ParentPhase)
		}
	}

	findings := signature.Match(events)

	return report.Analysis{
		Ecosystem:  ecosystem,
		Target:     target,
		StartedAt:  startedAt,
		Verdict:    report.Verdict(findings),
		Events:     len(events),
		Skipped:    skipped,
		Kinds:      report.Kinds(events),
		Findings:   findings,
		Behaviours: report.Behaviours(events),
	}, nil
}

func printSide(label string, s report.Side) {
	fmt.Printf("%-12s %-14s %d behaviours", label, s.Verdict, len(s.Behaviours))
	if len(s.Unstable) > 0 {
		fmt.Printf(", %d unstable (dropped)", len(s.Unstable))
	}
	fmt.Println()
	for _, b := range s.Unstable {
		fmt.Printf("  unstable  %-16s %s\n", b.Kind, b.Target)
	}
}

func printBehaviours(label string, bs []report.Behaviour) {
	fmt.Printf("%s (%d)\n", label, len(bs))
	for _, b := range bs {
		fmt.Printf("  %-16s %s\n", b.Kind, b.Target)
	}
}

// resolve turns a local path into an absolute one so it stays valid no matter
// which working directory a subprocess runs in. Named packages pass through.
func resolve(target string) (string, error) {
	if !isLocalPath(target) {
		return target, nil
	}
	return filepath.Abs(target)
}

func isLocalPath(target string) bool {
	return strings.HasPrefix(target, ".") || strings.HasPrefix(target, "/")
}

func outRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	outDir := filepath.Join(wd, "out")
	return outDir, os.MkdirAll(outDir, 0o755)
}
