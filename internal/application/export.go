package application

import (
	"encoding/json"
	"stage-rigging-clearance/internal/domain"
	"stage-rigging-clearance/internal/storage"
)

type ExportDocument struct {
	Case     domain.RiggingCase `json:"case"`
	Timeline []TimelineItem     `json:"timeline"`
	Digest   string             `json:"digest"`
}

func (s *Service) Export(id string) ([]byte, error) {
	c, e := s.store.Get(id)
	if e != nil {
		return nil, e
	}
	t, e := s.Timeline(id)
	if e != nil {
		return nil, e
	}
	doc := ExportDocument{Case: *c, Timeline: t, Digest: storage.SnapshotDigest(c)}
	b, e := json.Marshal(doc)
	if e != nil {
		return nil, e
	}
	return b, nil
}
func (s *Service) Compare(id, a, b string) (RevisionDiff, error) {
	c, e := s.store.Get(id)
	if e != nil {
		return RevisionDiff{}, e
	}
	ra, e := c.RevisionByID(a)
	if e != nil {
		return RevisionDiff{}, e
	}
	rb, e := c.RevisionByID(b)
	if e != nil {
		return RevisionDiff{}, e
	}
	return DiffRevisions(*ra, *rb), nil
}
func (s *Service) Findings(id string) ([]domain.SafetyFinding, error) {
	c, e := s.store.Get(id)
	if e != nil {
		return nil, e
	}
	return append([]domain.SafetyFinding(nil), c.Findings...), nil
}
