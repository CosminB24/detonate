package report

import "github.com/CosminB24/detonate/internal/signature"

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