package assessment

import (
	"fmt"
	"sort"
	"stage-rigging-clearance/internal/domain"
)

type Report struct {
	Title     string            `json:"title"`
	Passed    bool              `json:"passed"`
	Blocking  int               `json:"blocking"`
	Warnings  int               `json:"warnings"`
	Metrics   Result            `json:"metrics"`
	RuleTexts map[string]string `json:"ruleTexts"`
}

func BuildReport(c *domain.RiggingCase, r Result) Report {
	p, w := 0, 0
	text := map[string]string{}
	for _, f := range r.Findings {
		text[f.RuleCode] = RuleDescription(f.RuleCode)
		if f.Severity == domain.SeverityBlocking {
			p++
		} else if f.Severity == domain.SeverityWarning {
			w++
		}
	}
	return Report{Title: fmt.Sprintf("修订 %d 安全核验", revisionNumber(c, r.RevisionID)), Passed: p == 0, Blocking: p, Warnings: w, Metrics: r, RuleTexts: text}
}
func revisionNumber(c *domain.RiggingCase, id string) int {
	for _, r := range c.Revisions {
		if r.ID == id {
			return r.RevisionNumber
		}
	}
	return 0
}
func StableFindingOrder(fs []domain.SafetyFinding) []domain.SafetyFinding {
	out := append([]domain.SafetyFinding(nil), fs...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].RuleCode == out[j].RuleCode {
			return out[i].ID < out[j].ID
		}
		return out[i].RuleCode < out[j].RuleCode
	})
	return out
}
func CheckThresholds(p domain.LoadPoint) (bool, bool) {
	return Utilization(p) > 1, p.ClearanceMm < MinClearanceMm
}
