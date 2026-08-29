package main

import (
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

func main() {
	const requiredArgs = 3
	if len(os.Args) < requiredArgs {
		fmt.Fprintln(os.Stderr, "Usage: detonate npm <package>")
		os.Exit(1)
	}
	if os.Args[1] != "npm" {
		fmt.Fprintln(os.Stderr, "Only npm is supported")
		os.Exit(1)
	}

	startedAt := time.Now()
	ecosystem := os.Args[1]
	target := os.Args[2]

	// A local path is resolved to an absolute path so it stays valid no matter
	// which working directory a subprocess runs in.
	isLocalPath := strings.HasPrefix(target, ".") || strings.HasPrefix(target, "/")
	if isLocalPath {
		abs, err := filepath.Abs(target)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		target = abs
	}

	fmt.Printf("%-12s %s\n", "ecosystem:", ecosystem)
	fmt.Printf("%-12s %s\n", "target:", target)

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	outPath := filepath.Join(wd, "out")
	if err := os.MkdirAll(outPath, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// The work directory is shared between the two container phases. Files in
	// it are created by root inside the container, so it must be removed from
	// a container too — the host user cannot delete them.
	workPath := filepath.Join(outPath, "work")
	if err := runner.Clean(outPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(workPath, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// What npm will install, as seen from inside the container.
	//
	// A named package is passed through untouched — phase 1 downloads it.
	// A local path is packed into a tarball first: installing a directory
	// creates a symlink ("file:" dependency), and npm never runs lifecycle
	// scripts for those. The tarball is written into the work directory
	// because that is what the container mounts.
	installArg := target
	if isLocalPath {
		pack := exec.Command("npm", "pack", target, "--pack-destination", workPath)
		out, err := pack.Output()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fileName := strings.TrimSpace(string(out))
		installArg = "/work/" + fileName
		fmt.Printf("%-12s %s\n", "packed:", fileName)
	}

	// Two phases: fetch with network but no execution, then execute offline
	// under strace. See internal/runner.
	logName := "trace.log"
	if err := runner.Detonate(outPath, workPath, installArg, logName); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	tracePath := filepath.Join(outPath, logName)
	fmt.Printf("%-12s %s\n", "trace:", tracePath)

	events, skipped, err := collect.Parse(tracePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%-12s %d\n", "events:", len(events))
	fmt.Printf("%-12s %d\n", "skipped:", skipped)

	counts := map[string]int{}
	for _, e := range events {
		counts[e.Kind]++
	}
	fmt.Println()
	for kind, n := range counts {
		fmt.Printf("%-16s %d\n", kind, n)
	}

	findings := signature.Match(events)
	verdict := report.Verdict(findings)

	fmt.Println()
	fmt.Printf("%-12s %s\n", "verdict:", verdict)
	fmt.Println()
	if len(findings) == 0 {
		fmt.Println("no findings")
	}
	for _, f := range findings {
		fmt.Printf("[%s] %s (%d events)\n", f.Severity, f.Title, len(f.Evidence))
	}

	rep := report.Report{
		SchemaVersion: "1.0",
		Analysis: report.Analysis{
			Ecosystem: ecosystem,
			Target:    target,
			StartedAt: startedAt,
			Verdict:   verdict,
			Events:    len(events),
			Skipped:   skipped,
			Findings:  findings,
			Behaviours: report.Behaviours(events),
		},
	}
	reportPath := filepath.Join(outPath, "report.json")
	if err := report.Write(reportPath, rep); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("\n%-12s %s\n", "report:", reportPath)
}