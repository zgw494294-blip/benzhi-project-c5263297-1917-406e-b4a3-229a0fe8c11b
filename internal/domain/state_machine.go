package domain

import "fmt"

func AllowedTransitions() map[CaseStatus][]CaseStatus {
	return map[CaseStatus][]CaseStatus{StatusDraft: {StatusPlanning}, StatusPlanning: {StatusValidationFailed, StatusUnderReview, StatusRemediating}, StatusValidationFailed: {StatusRemediating, StatusPlanning}, StatusRemediating: {StatusPlanning, StatusValidationFailed, StatusUnderReview}, StatusUnderReview: {StatusChangesRequested, StatusApproved, StatusPlanning}, StatusChangesRequested: {StatusPlanning}, StatusApproved: {StatusFrozen, StatusPlanning}, StatusFrozen: {}}
}
func CanTransition(from, to CaseStatus) bool {
	for _, x := range AllowedTransitions()[from] {
		if x == to {
			return true
		}
	}
	return false
}
func StateLabel(s CaseStatus) string {
	switch s {
	case StatusDraft:
		return "草稿"
	case StatusPlanning:
		return "方案编制"
	case StatusValidationFailed:
		return "核验未通过"
	case StatusRemediating:
		return "整改中"
	case StatusUnderReview:
		return "待复核"
	case StatusChangesRequested:
		return "要求修改"
	case StatusApproved:
		return "已批准"
	case StatusFrozen:
		return "已冻结"
	}
	return fmt.Sprintf("未知状态(%s)", s)
}
func (c *RiggingCase) Transition(to CaseStatus, actor string) error {
	if !CanTransition(c.Status, to) {
		return ErrInvalidState
	}
	c.Status = to
	c.touch("Transition:"+string(to), actor)
	return nil
}
func SeverityLabel(s Severity) string {
	switch s {
	case SeverityBlocking:
		return "阻断"
	case SeverityWarning:
		return "警告"
	case SeverityInfo:
		return "提示"
	}
	return string(s)
}
func OutcomeLabel(o ReviewOutcome) string {
	if o == OutcomeApprove {
		return "通过"
	}
	return "退回"
}
