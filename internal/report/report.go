package report

import (
	"time"
	"encoding/json"
	"os"

	"github.com/CosminB24/detonate/internal/signature"
)

type Report struct {
	SchemaVersion string    `json:"schema_version"`
	Analysis      Analysis  `json:"analysis"`
}

type Analysis struct {
	Ecosystem string              `json:"ecosystem"`
	Target    string              `json:"target"`
	StartedAt time.Time           `json:"started_at"`
	Verdict   string              `json:"verdict"`
	Events    int                 `json:"events"`
	Skipped   int                 `json:"skipped"`
	Findings  []signature.Finding `json:"findings"`
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