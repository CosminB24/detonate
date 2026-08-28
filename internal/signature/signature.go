package signature

import (
	"strings"

	"github.com/CosminB24/detonate/internal/collect"
)

type Rule struct {
	ID       string   // stable identifier, e.g. CRED_FILE_READ
	Title    string   // human-readable summary
	Severity string   // high / medium / low
	Kind     string   // event kind the rule looks for
	Contains []string // rule fires if Target contains ANY of these
}

type Finding struct {
	RuleID   string `json:"rule_id"`
	Title    string `json:"title"`
	Severity string `json:"severity"`
	Evidence []int  `json:"evidence"`
}

// rules is the built-in rule set. Later phase these move to YAML files
var rules = []Rule{
	{
		ID:       "CRED_FILE_READ",
		Title:    "Read credential file",
		Severity: "high",
		Kind:     "file.read",
		Contains: []string{".ssh/", ".aws/", ".npmrc"},
	},
	{
		ID:       "SHELL_RC_WRITE",
		Title:    "Modified shell startup file",
		Severity: "high",
		Kind:     "file.write",
		Contains: []string{".bashrc", ".zshrc", ".profile"},
	},
	{
		ID:       "SHELL_SPAWN",
		Title:    "Install script spawned a shell",
		Severity: "medium",
		Kind:     "process.exec",
		Contains: []string{"/sh", "/bash"},
	},
}

// Match runs every rule against the events and returns the findings.
// A rule with no matching events produces no finding: a finding without
// evidence is never emitted.
func Match(events []collect.Event) []Finding {
	var findings []Finding

	for _, rule := range rules {
		var evidence []int

		for _, e := range events {
			if e.Failed {
				continue
			}

			if e.Kind != rule.Kind {
				continue
			}

			for _, frag := range rule.Contains {
				if strings.Contains(e.Target, frag) {
					evidence = append(evidence, e.Seq)
					break // one match per event is enough
				}
			}
		}

		if len(evidence) > 0 {
			findings = append(findings, Finding{
				RuleID:   rule.ID,
				Title:    rule.Title,
				Severity: rule.Severity,
				Evidence: evidence,
			})
		}
	}

	return findings
}