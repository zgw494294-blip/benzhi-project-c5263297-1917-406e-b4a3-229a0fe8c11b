package application

import "stage-rigging-clearance/internal/domain"

type CreateCommand struct{ PerformanceName, StageZones, OwnerName, PerformanceDate string }
type RevisionCommand struct {
	CaseID                   string
	ExpectedVersion          int
	Note, By, IdempotencyKey string
	Points                   []domain.LoadPoint
}
type ReviewCommand struct {
	CaseID                            string
	ExpectedVersion                   int
	Stage, Outcome, Reviewer, Comment string
}
