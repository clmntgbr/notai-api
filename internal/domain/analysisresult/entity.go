package analysisresult

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	domaincontent "go-api/internal/domain/content"

	"github.com/google/uuid"
)

type Status string

const (
	StatusSuccess Status = "success"
	StatusFailed  Status = "failed"
	StatusTimeout Status = "timeout"
)

var (
	ErrInvalidDetectorName = errors.New("detector name is required")
	ErrInvalidStatus       = errors.New("invalid analysis result status")
)

// Result is one detector outcome for a content item.
type Result struct {
	ID             uuid.UUID
	ContentID      uuid.UUID
	DetectorName   string
	Status         Status
	Signals        []domaincontent.Signal
	Error          string
	StartedAt      time.Time
	CompletedAt    time.Time
	Weight         float64
	RulesetVersion *int
}

func New(contentID uuid.UUID, detectorName string, startedAt, completedAt time.Time, weight float64) (*Result, error) {
	detectorName = strings.TrimSpace(detectorName)
	if detectorName == "" {
		return nil, ErrInvalidDetectorName
	}
	if contentID == uuid.Nil {
		return nil, errors.New("content id is required")
	}
	if weight <= 0 {
		weight = 1
	}
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	if completedAt.IsZero() {
		completedAt = time.Now().UTC()
	}
	return &Result{
		ID:           uuid.New(),
		ContentID:    contentID,
		DetectorName: detectorName,
		StartedAt:    startedAt.UTC(),
		CompletedAt:  completedAt.UTC(),
		Weight:       weight,
	}, nil
}

func (r *Result) MarkSuccess(signals []domaincontent.Signal) {
	r.Status = StatusSuccess
	r.Signals = signals
	r.Error = ""
	r.CompletedAt = time.Now().UTC()
}

func (r *Result) MarkFailed(message string) {
	r.Status = StatusFailed
	r.Signals = nil
	r.Error = message
	r.CompletedAt = time.Now().UTC()
}

func (r *Result) MarkTimeout(message string) {
	r.Status = StatusTimeout
	r.Signals = nil
	r.Error = message
	r.CompletedAt = time.Now().UTC()
}

func (r *Result) ExpectedWeight() float64 {
	if r.Weight <= 0 {
		return 1
	}
	return r.Weight
}

func (r *Result) SignalsJSON() ([]byte, error) {
	if len(r.Signals) == 0 {
		return nil, nil
	}
	return json.Marshal(r.Signals)
}

func SignalsFromJSON(raw []byte) ([]domaincontent.Signal, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var signals []domaincontent.Signal
	if err := json.Unmarshal(raw, &signals); err != nil {
		return nil, err
	}
	return signals, nil
}
