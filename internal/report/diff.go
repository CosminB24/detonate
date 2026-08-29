package report

import (
	"time"

	"github.com/CosminB24/detonate/internal/signature"
)

// DiffReport is the artefact of `detonate diff`: what a package started doing
// between two versions.
type DiffReport struct {
	SchemaVersion string       `json:"schema_version"`
	Diff          DiffAnalysis `json:"diff"`
}

type DiffAnalysis struct {
	Ecosystem      string              `json:"ecosystem"`
	StartedAt      time.Time           `json:"started_at"`
	RunsPerVersion int                 `json:"runs_per_version"`
	From           Side                `json:"from"`
	To             Side                `json:"to"`
	Changes        Diff                `json:"changes"`
	NewFindings    []signature.Finding `json:"new_findings"`
}

// Side is one of the two versions under comparison. Behaviours holds only
// what every run of that version agreed on; Unstable holds the rest.
type Side struct {
	Target     string              `json:"target"`
	Verdict    string              `json:"verdict"`
	Behaviours []Behaviour         `json:"behaviours"`
	Unstable   []Behaviour         `json:"unstable"`
	Findings   []signature.Finding `json:"findings"`
}

// Diff is the result of comparing two behaviour sets. The three groups are
// disjoint and together cover the union of both sets.
type Diff struct {
	New       []Behaviour `json:"new"`
	Removed   []Behaviour `json:"removed"`
	Unchanged []Behaviour `json:"unchanged"`
}

// Compare reports how behaviour changed from one version to another. New is
// what the second version does and the first did not — the question the diff
// exists to answer.
func Compare(from, to []Behaviour) Diff {
	inFrom := index(from)
	inTo := index(to)

	d := Diff{
		New:       make([]Behaviour, 0),
		Removed:   make([]Behaviour, 0),
		Unchanged: make([]Behaviour, 0),
	}
	for b := range inTo {
		if inFrom[b] {
			d.Unchanged = append(d.Unchanged, b)
		} else {
			d.New = append(d.New, b)
		}
	}
	for b := range inFrom {
		if !inTo[b] {
			d.Removed = append(d.Removed, b)
		}
	}

	sortBehaviours(d.New)
	sortBehaviours(d.Removed)
	sortBehaviours(d.Unchanged)
	return d
}

// Stable returns the behaviours every run of the same version produced.
// Two runs of one version are expected to agree; anything that varies is
// nondeterminism the normalisation in Behaviours did not catch, and comparing
// it across versions would report noise as change.
func Stable(runs [][]Behaviour) []Behaviour {
	return agreed(runs, true)
}

// Unstable returns the behaviours only some runs produced. It is reported but
// never compared: a non-empty result is a gap in the normalisation, and the
// size of that gap is the measurement worth keeping.
func Unstable(runs [][]Behaviour) []Behaviour {
	return agreed(runs, false)
}

func agreed(runs [][]Behaviour, inEveryRun bool) []Behaviour {
	if len(runs) == 0 {
		return make([]Behaviour, 0)
	}

	// Counted per run rather than in total, so a duplicate inside one run
	// cannot stand in for a run that never produced the behaviour at all.
	count := map[Behaviour]int{}
	for _, run := range runs {
		for b := range index(run) {
			count[b]++
		}
	}

	out := make([]Behaviour, 0, len(count))
	for b, n := range count {
		if (n == len(runs)) == inEveryRun {
			out = append(out, b)
		}
	}
	sortBehaviours(out)
	return out
}

// NewFindings returns the findings of the second version whose rule did not
// fire for the first. Evidence points into different traces, so the rule is
// the only granularity at which two runs can be compared.
func NewFindings(from, to []signature.Finding) []signature.Finding {
	fired := map[string]bool{}
	for _, f := range from {
		fired[f.RuleID] = true
	}

	out := make([]signature.Finding, 0)
	for _, f := range to {
		if !fired[f.RuleID] {
			out = append(out, f)
		}
	}
	return out
}

func index(bs []Behaviour) map[Behaviour]bool {
	m := make(map[Behaviour]bool, len(bs))
	for _, b := range bs {
		m[b] = true
	}
	return m
}
