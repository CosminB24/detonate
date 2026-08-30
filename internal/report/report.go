package report

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/CosminB24/detonate/internal/collect"
	"github.com/CosminB24/detonate/internal/signature"
)

type Report struct {
	SchemaVersion string   `json:"schema_version"`
	Analysis      Analysis `json:"analysis"`
}

type Analysis struct {
	Ecosystem  string              `json:"ecosystem"`
	Target     string              `json:"target"`
	StartedAt  time.Time           `json:"started_at"`
	Verdict    string              `json:"verdict"`
	Events     int                 `json:"events"`
	Skipped    int                 `json:"skipped"`
	Kinds      map[string]int      `json:"kinds"`
	Findings   []signature.Finding `json:"findings"`
	Behaviours []Behaviour         `json:"behaviours"`
}

type Behaviour struct {
	Kind   string `json:"kind"`
	Target string `json:"target"`
	Phase  string `json:"phase"`
}

// volatilePrefixes are paths that differ between identical runs and would
// otherwise show up as spurious differences when diffing two versions.
// npm writes a debug log whose filename contains a timestamp.
var volatilePrefixes = []string{
	"/root/.npm/_logs", // no trailing slash: the directory itself varies too
	"/sys/",            // kernel interfaces probed conditionally by the runtime
	"/proc/",

	// npm checks for its own updates at most once a day, so this write
	// appears or not depending on when the previous run happened.
	"/root/.npm/_update-notifier-last-checked",
	"/dev/null",
}

// harnessReadPrefixes are paths whose reads belong to the tooling rather than
// to the package: npm loading its own source. Measured on express they were
// 708 of 727 behaviours — 97% of the set — and every module npm happens to
// load lazily is a potential spurious difference between two versions.
//
// Reads only. A package WRITING into npm's own installation is persistence,
// a real behaviour, and must still be reported.
var harnessReadPrefixes = []string{
	"/usr/local/lib/node_modules/npm/",
	"/lib/",
	"/usr/lib/",
	"/etc/ssl/",
	"/usr/local/bin/node",
}

// Failed syscalls are excluded: an attempt is not an action, and counting
// attempts is a large source of false positives. file.stat is excluded as
// well — it is roughly 80% of any trace and carries no signal.
func Behaviours(events []collect.Event) []Behaviour {
	seen := map[Behaviour]bool{}

	for _, e := range events {
		if e.Failed {
			continue
		}
		if e.Partial {
			continue
		}
		if e.Kind == "file.stat" {
			continue
		}
		if e.Target == "" {
			continue
		}
		if hasAnyPrefix(e.Target, volatilePrefixes) {
			continue
		}
		if e.Kind == "file.read" && hasAnyPrefix(e.Target, harnessReadPrefixes) {
			continue
		}
		if strings.HasSuffix(e.Target, "/node_modules") {
			continue
		}
		if e.Target == "/work/package.json" || strings.HasSuffix(e.Target, "package-lock.json") {
			continue
		}

		seen[Behaviour{Kind: e.Kind, Target: e.Target, Phase: e.Phase}] = true
	}

	out := make([]Behaviour, 0, len(seen))
	for b := range seen {
		out = append(out, b)
	}

	sortBehaviours(out)
	return out
}

// Map iteration order is random in Go, so any behaviour set must be sorted
// before it leaves this package — otherwise two identical runs produce
// different reports and the diff is meaningless.
func sortBehaviours(bs []Behaviour) {
	slices.SortFunc(bs, func(a, b Behaviour) int {
		if a.Phase != b.Phase {
			return strings.Compare(a.Phase, b.Phase)
		}
		if a.Kind != b.Kind {
			return strings.Compare(a.Kind, b.Kind)
		}
		return strings.Compare(a.Target, b.Target)
	})
}

func hasAnyPrefix(target string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(target, p) {
			return true
		}
	}
	return false
}

// Kinds counts the events per behaviour kind. Unlike Behaviours it counts
// everything, failures included: it describes the trace, not the package.
func Kinds(events []collect.Event) map[string]int {
	counts := map[string]int{}
	for _, e := range events {
		counts[e.Kind]++
	}
	return counts
}

// Verdict returns the overall assessment.
// There is deliberately no "benign" outcome: with the current coverage
// (single run, install scripts only, exported functions never invoked),
// the absence of findings is not evidence of safety
func Verdict(findings []signature.Finding) string {
	if len(findings) == 0 {
		return "inconclusive"
	}
	for _, f := range findings {
		if f.Severity == "high" {
			return "malicious"
		}
	}
	return "suspicious"
}

func Write(path string, r Report) error {
	return writeJSON(path, r)
}

func WriteDiff(path string, d DiffReport) error {
	return writeJSON(path, d)
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
