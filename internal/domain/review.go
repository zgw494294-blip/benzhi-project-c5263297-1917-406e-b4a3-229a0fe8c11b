package domain

import (
	"strings"
	"time"
)

func (c *RiggingCase) AddReview(d ReviewDecision, actor string) error {
	if c.Status == StatusFrozen {
		return ErrFrozen
	}
	r := c.CurrentRevision()
	if r == nil {
		return ErrInvalidState
	}
	if d.RevisionID != r.ID || d.RevisionDigest != r.ContentDigest {
		return ErrStaleAssessment
	}
	if c.LastAssessmentDigest != r.ContentDigest {
		return ErrStaleAssessment
	}
	if d.Stage != StageSafety && d.Stage != StageTechnical {
		return ErrInvalidInput
	}
	if d.Outcome != OutcomeApprove && d.Outcome != OutcomeReturn {
		return ErrInvalidInput
	}
	d.Reviewer, d.Comment = strings.TrimSpace(d.Reviewer), strings.TrimSpace(d.Comment)
	if d.Reviewer == "" || (d.Outcome == OutcomeReturn && d.Comment == "") {
		return ErrInvalidInput
	}
	for _, x := range c.Reviews {
		if x.RevisionID == r.ID && x.Stage == d.Stage {
			return ErrReviewOrder
		}
	}
	if d.Stage == StageSafety {
		if c.OpenBlocking() {
			return ErrBlockingFindings
		}
	} else {
		if c.OpenBlocking() {
			return ErrBlockingFindings
		}
		found := false
		for _, x := range c.Reviews {
			if x.RevisionID == r.ID && x.Stage == StageSafety && x.Outcome == OutcomeApprove {
				found = true
			}
		}
		if !found {
			return ErrReviewOrder
		}
	}
	d.CaseID = c.ID
	d.ID = fmtID("review", len(c.Reviews)+1)
	d.DecidedAt = time.Now().UTC()
	c.Reviews = append(c.Reviews, d)
	if d.Outcome == OutcomeReturn {
		c.Status = StatusChangesRequested
	} else if d.Stage == StageTechnical {
		c.Status = StatusApproved
	}
	c.touch("ReviewDecision", actor)
	return nil
}
func fmtID(prefix string, n int) string {
	return prefix + "-" + time.Now().UTC().Format("20060102150405.000000000") + "-" + itoa(n)
}
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	b := make([]byte, 0, 8)
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
