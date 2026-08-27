package domain

import "time"

type CaseStatus string

const (
	StatusDraft            CaseStatus = "Draft"
	StatusPlanning         CaseStatus = "Planning"
	StatusValidationFailed CaseStatus = "ValidationFailed"
	StatusRemediating      CaseStatus = "Remediating"
	StatusUnderReview      CaseStatus = "UnderReview"
	StatusChangesRequested CaseStatus = "ChangesRequested"
	StatusApproved         CaseStatus = "Approved"
	StatusFrozen           CaseStatus = "Frozen"
)

type Severity string

const (
	SeverityInfo     Severity = "Info"
	SeverityWarning  Severity = "Warning"
	SeverityBlocking Severity = "Blocking"
)

type FindingStatus string

const (
	FindingOpen     FindingStatus = "Open"
	FindingResolved FindingStatus = "Resolved"
)

type ReviewStage string

const (
	StageSafety    ReviewStage = "Safety"
	StageTechnical ReviewStage = "Technical"
)

type ReviewOutcome string

const (
	OutcomeApprove ReviewOutcome = "Approve"
	OutcomeReturn  ReviewOutcome = "Return"
)

type LoadPoint struct {
	ID            string  `json:"loadPointId"`
	RevisionID    string  `json:"revisionId"`
	PointCode     string  `json:"pointCode"`
	StageZone     string  `json:"stageZone"`
	ComponentName string  `json:"componentName"`
	RatedLoadKg   float64 `json:"ratedLoadKg"`
	ActualLoadKg  float64 `json:"actualLoadKg"`
	ClearanceMm   float64 `json:"clearanceMm"`
	PositionXmm   float64 `json:"positionXmm"`
	PositionYmm   float64 `json:"positionYmm"`
	CueStart      int     `json:"cueStart"`
	CueEnd        int     `json:"cueEnd"`
}
type PlanRevision struct {
	ID                 string      `json:"revisionId"`
	CaseID             string      `json:"caseId"`
	ReplacesRevisionID string      `json:"replacesRevisionId"`
	ChangeNote         string      `json:"changeNote"`
	SubmittedBy        string      `json:"submittedBy"`
	ContentDigest      string      `json:"contentDigest"`
	RevisionNumber     int         `json:"revisionNumber"`
	SubmittedAt        time.Time   `json:"submittedAt"`
	Points             []LoadPoint `json:"points"`
}
type SafetyFinding struct {
	ID                    string        `json:"findingId"`
	CaseID                string        `json:"caseId"`
	RevisionID            string        `json:"revisionId"`
	RuleCode              string        `json:"ruleCode"`
	Message               string        `json:"message"`
	RemediationRevisionID string        `json:"remediationRevisionId"`
	RemediationNote       string        `json:"remediationNote"`
	VerifiedBy            string        `json:"verifiedBy"`
	Severity              Severity      `json:"severity"`
	RelatedPointIDs       []string      `json:"relatedPointIds"`
	Status                FindingStatus `json:"status"`
	VerifiedAt            *time.Time    `json:"verifiedAt"`
}
type ReviewDecision struct {
	ID             string        `json:"decisionId"`
	CaseID         string        `json:"caseId"`
	RevisionID     string        `json:"revisionId"`
	Reviewer       string        `json:"reviewer"`
	Comment        string        `json:"comment"`
	RevisionDigest string        `json:"revisionDigest"`
	Stage          ReviewStage   `json:"reviewStage"`
	Outcome        ReviewOutcome `json:"outcome"`
	DecidedAt      time.Time     `json:"decidedAt"`
}
type ClearancePermit struct {
	ID                 string    `json:"permitId"`
	PermitCode         string    `json:"permitCode"`
	CaseID             string    `json:"caseId"`
	FrozenRevisionID   string    `json:"frozenRevisionId"`
	SnapshotDigest     string    `json:"snapshotDigest"`
	IssuedBy           string    `json:"issuedBy"`
	VerificationDigest string    `json:"verificationDigest"`
	IssuedAt           time.Time `json:"issuedAt"`
}
type AssessmentSnapshot struct {
	RevisionID   string                  `json:"revisionId"`
	InputDigest  string                  `json:"inputDigest"`
	RuleVersion  string                  `json:"ruleVersion"`
	PointMetrics []AssessmentPointMetric `json:"pointMetrics"`
	ZoneLoads    map[string]float64      `json:"zoneLoads"`
	ZoneMetrics  []AssessmentZoneMetric  `json:"zoneMetrics,omitempty"`
	Eccentricity float64                 `json:"eccentricity"`
	FindingIDs   []string                `json:"findingIds"`
}
type AssessmentPointMetric struct {
	PointCode   string  `json:"pointCode"`
	Utilization float64 `json:"utilization"`
}
type AssessmentZoneMetric struct {
	Zone       string  `json:"zone"`
	LoadKg     float64 `json:"loadKg"`
	PointCount int     `json:"pointCount"`
}
type RiggingCase struct {
	ID                   string              `json:"caseId"`
	PerformanceName      string              `json:"performanceName"`
	StageZones           string              `json:"stageZones"`
	OwnerName            string              `json:"ownerName"`
	PerformanceDate      time.Time           `json:"performanceDate"`
	Status               CaseStatus          `json:"status"`
	CurrentRevisionID    string              `json:"currentRevisionId"`
	ExpectedVersion      int                 `json:"expectedVersion"`
	CreatedAt            time.Time           `json:"createdAt"`
	UpdatedAt            time.Time           `json:"updatedAt"`
	Revisions            []PlanRevision      `json:"revisions"`
	Findings             []SafetyFinding     `json:"findings"`
	Reviews              []ReviewDecision    `json:"reviews"`
	Permit               *ClearancePermit    `json:"permit"`
	Audit                []AuditEvent        `json:"audit"`
	LastAssessmentDigest string              `json:"lastAssessmentDigest"`
	LastAssessment       *AssessmentSnapshot `json:"lastAssessment,omitempty"`
	RevisionIdempotency  map[string]string   `json:"revisionIdempotency,omitempty"`
}
type AuditEvent struct {
	Seq     int       `json:"seq"`
	Type    string    `json:"type"`
	Message string    `json:"message"`
	Actor   string    `json:"actor"`
	At      time.Time `json:"at"`
}
