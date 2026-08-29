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
	Findings   []signature.Finding `json:"findings"`
	Behaviours []Behaviour         `json:"behaviours"`
}

type Behaviour struct {
	Kind   string `json:"kind"`
	Target string `json:"target"`
}

// volatilePrefixes are paths that differ between identical runs and would
// otherwise show up as spurious differences when diffing two versions.
// npm writes a debug log whose filename contains a timestamp.
var volatilePrefixes = []string{
	"/root/.npm/_logs/",
	"/sys/", // kernel interfaces probed conditionally by the runtime
	"/proc/",
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
		if isVolatile(e.Target) {
			continue
		}
		if strings.HasSuffix(e.Target, "/node_modules") {
			continue
		}
		if e.Target == "/work/package.json" || e.Target == "/work/package-lock.json" {
			continue
		}
		if strings.HasSuffix(e.Target, "package-lock.json") ||
			e.Target == "/work/package.json" {
			continue
		}

		seen[Behaviour{Kind: e.Kind, Target: e.Target}] = true
	}

	out := make([]Behaviour, 0, len(seen))
	for b := range seen {
		out = append(out, b)
	}

	// Map iteration order is random in Go, so the result must be sorted to be
	// stable — otherwise two identical runs would produce different reports.
	slices.SortFunc(out, func(a, b Behaviour) int {
		if a.Kind != b.Kind {
			return strings.Compare(a.Kind, b.Kind)
		}
		return strings.Compare(a.Target, b.Target)
	})

	return out
}

func isVolatile(target string) bool {
	for _, p := range volatilePrefixes {
		if strings.HasPrefix(target, p) {
			return true
		}
	}
	return false
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
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
