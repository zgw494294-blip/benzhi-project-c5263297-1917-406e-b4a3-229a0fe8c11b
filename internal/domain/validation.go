package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type ValidationIssue struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func ValidateCaseInput(performance, zones, owner string) []ValidationIssue {
	var out []ValidationIssue
	if strings.TrimSpace(performance) == "" {
		out = append(out, ValidationIssue{"performanceName", "演出名称不能为空"})
	}
	if strings.TrimSpace(zones) == "" {
		out = append(out, ValidationIssue{"stageZones", "舞台区域不能为空"})
	}
	if strings.TrimSpace(owner) == "" {
		out = append(out, ValidationIssue{"ownerName", "负责人不能为空"})
	}
	return out
}
func ValidatePerformanceDate(value string, now time.Time) (time.Time, []ValidationIssue) {
	d, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, []ValidationIssue{{"performanceDate", "演出日期必须符合 YYYY-MM-DD 格式"}}
	}
	today := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	if d.Before(today) {
		return time.Time{}, []ValidationIssue{{"performanceDate", "演出日期不得早于当前日期"}}
	}
	return d, nil
}
func ValidatePoint(p LoadPoint) []ValidationIssue {
	var out []ValidationIssue
	if strings.TrimSpace(p.PointCode) == "" {
		out = append(out, ValidationIssue{"pointCode", "吊点编号不能为空"})
	}
	if p.RatedLoadKg <= 0 {
		out = append(out, ValidationIssue{"ratedLoadKg", "额定载荷必须大于零"})
	}
	if p.ActualLoadKg < 0 {
		out = append(out, ValidationIssue{"actualLoadKg", "实际载荷不能为负"})
	}
	if p.ClearanceMm < 0 {
		out = append(out, ValidationIssue{"clearanceMm", "净空不能为负"})
	}
	if p.CueEnd <= p.CueStart {
		out = append(out, ValidationIssue{"cueRange", "场景区间必须递增"})
	}
	return out
}
func ValidateRevision(r PlanRevision) []ValidationIssue {
	var out []ValidationIssue
	if len(r.Points) == 0 {
		return []ValidationIssue{{"points", "至少登记一个吊点"}}
	}
	seen := map[string]bool{}
	for i, p := range r.Points {
		p.PointCode = strings.TrimSpace(p.PointCode)
		for _, x := range ValidatePoint(p) {
			x.Field = fmt.Sprintf("points[%d].%s", i, x.Field)
			out = append(out, x)
		}
		if seen[p.PointCode] {
			out = append(out, ValidationIssue{fmt.Sprintf("points[%d].pointCode", i), "吊点编号重复"})
		}
		seen[p.PointCode] = true
	}
	return out
}
func (c *RiggingCase) RevisionHistory() []PlanRevision {
	out := append([]PlanRevision(nil), c.Revisions...)
	for i := range out {
		out[i].Points = append([]LoadPoint(nil), out[i].Points...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RevisionNumber < out[j].RevisionNumber })
	return out
}
func (c *RiggingCase) FindingSummary() map[Severity]int {
	m := map[Severity]int{}
	for _, f := range c.Findings {
		m[f.Severity]++
	}
	return m
}
func (c *RiggingCase) HasApprovedStage(stage ReviewStage) bool {
	for _, r := range c.Reviews {
		if r.RevisionID == c.CurrentRevisionID && r.Stage == stage && r.Outcome == OutcomeApprove {
			return true
		}
	}
	return false
}
func (c *RiggingCase) CanFreeze() error {
	if c.Status == StatusFrozen {
		return ErrFrozen
	}
	r := c.CurrentRevision()
	if r == nil {
		return ErrInvalidState
	}
	if c.LastAssessment == nil || c.LastAssessment.RevisionID != r.ID || c.LastAssessment.InputDigest != r.ContentDigest || c.LastAssessmentDigest != r.ContentDigest {
		return ErrStaleAssessment
	}
	if c.OpenBlocking() {
		return ErrBlockingFindings
	}
	if !c.HasApprovedStage(StageSafety) || !c.HasApprovedStage(StageTechnical) {
		return ErrReviewOrder
	}
	if c.Status != StatusApproved {
		return ErrInvalidState
	}
	return nil
}
