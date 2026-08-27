package web

import (
	"fmt"
	"stage-rigging-clearance/internal/application"
	"stage-rigging-clearance/internal/domain"
)

type Dashboard struct {
	CaseID, Performance, Owner, Status, StatusLabel string
	Revision                                        int
	OpenFindings                                    int
	PermitCode                                      string
	Timeline                                        []application.TimelineItem
}

func dashboard(c *domain.RiggingCase, t []application.TimelineItem) Dashboard {
	rev := 0
	if r := c.CurrentRevision(); r != nil {
		rev = r.RevisionNumber
	}
	code := ""
	if c.Permit != nil {
		code = c.Permit.PermitCode
	}
	n := 0
	for _, f := range c.Findings {
		if f.Status == domain.FindingOpen {
			n++
		}
	}
	return Dashboard{CaseID: c.ID, Performance: c.PerformanceName, Owner: c.OwnerName, Status: string(c.Status), StatusLabel: domain.StateLabel(c.Status), Revision: rev, OpenFindings: n, PermitCode: code, Timeline: t}
}
func formatError(e error) string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("操作未完成：%s", e.Error())
}
func findingView(f domain.SafetyFinding) map[string]any {
	return map[string]any{"id": f.ID, "rule": f.RuleCode, "severity": domain.SeverityLabel(f.Severity), "message": f.Message, "status": f.Status, "remediationRevisionId": f.RemediationRevisionID}
}
