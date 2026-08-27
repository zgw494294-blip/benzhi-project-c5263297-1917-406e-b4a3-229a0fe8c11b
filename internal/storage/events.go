package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"stage-rigging-clearance/internal/domain"
	"time"
)

type Event struct {
	Seq        int       `json:"seq"`
	CaseID     string    `json:"caseId"`
	Type       string    `json:"type"`
	At         time.Time `json:"at"`
	Digest     string    `json:"digest"`
	PrevDigest string    `json:"prevDigest,omitempty"`
}

func (s *Store) AppendEvent(e Event) error {
	if s.dir == "" {
		return nil
	}
	f, err := os.OpenFile(filepath.Join(s.dir, "events.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	b, _ := json.Marshal(e)
	_, err = f.Write(append(b, '\n'))
	return err
}
func ValidateEvent(e Event) error {
	if e.Seq < 1 || e.CaseID == "" || e.Type == "" {
		return domain.ErrInvalidInput
	}
	return nil
}
