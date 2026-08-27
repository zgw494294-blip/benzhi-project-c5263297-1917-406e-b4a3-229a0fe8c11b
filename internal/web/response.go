package web

import (
	"net/http"
	"stage-rigging-clearance/internal/domain"
)

type ErrorResponse struct {
	Code      string                   `json:"code"`
	Message   string                   `json:"message"`
	Retryable bool                     `json:"retryable"`
	Issues    []domain.ValidationIssue `json:"issues,omitempty"`
}

func classifyError(e error) ErrorResponse {
	r := ErrorResponse{Code: "invalid_request", Message: e.Error()}
	if v, ok := e.(domain.ValidationError); ok {
		r.Issues = v.Issues
	}
	switch e {
	case domain.ErrVersionConflict:
		r.Code = "version_conflict"
		r.Retryable = true
	case domain.ErrFrozen:
		r.Code = "frozen"
	case domain.ErrBlockingFindings:
		r.Code = "blocking_findings"
	case domain.ErrStaleAssessment:
		r.Code = "stale_assessment"
	case domain.ErrReviewOrder:
		r.Code = "review_order"
	case domain.ErrNotFound:
		r.Code = "not_found"
	}
	return r
}
func writeError(w http.ResponseWriter, e error) {
	status := http.StatusBadRequest
	if e == domain.ErrNotFound {
		status = http.StatusNotFound
	}
	if e == domain.ErrVersionConflict {
		status = http.StatusConflict
	}
	writeJSON(w, status, classifyError(e))
}
func noStore(w http.ResponseWriter) { w.Header().Set("Cache-Control", "no-store") }
