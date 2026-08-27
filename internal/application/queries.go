package application

import "stage-rigging-clearance/internal/domain"

type CaseView struct {
	Case            *domain.RiggingCase
	Current         *domain.PlanRevision
	OpenFindings    int
	AssessmentFresh bool
}

func BuildView(c *domain.RiggingCase) CaseView {
	n := 0
	for _, f := range c.Findings {
		if f.Status != domain.FindingResolved {
			n++
		}
	}
	return CaseView{Case: c, Current: c.CurrentRevision(), OpenFindings: n, AssessmentFresh: c.LastAssessmentDigest != "" && c.CurrentRevision() != nil && c.LastAssessmentDigest == c.CurrentRevision().ContentDigest}
}
