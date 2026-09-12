package heuristic

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"

	domaincontent "go-api/internal/domain/content"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

const DetectorName = "heuristic"

// Detector orchestrates internal heuristic checks against one shared image decode.
type Detector struct {
	storage  port.Storage
	rulesets RulesetStore
	checks   []Check
	versions sync.Map // contentID.String() -> ruleset version
}

func New(storage port.Storage, rulesets RulesetStore, hashRepo KnownHashRepository) *Detector {
	return &Detector{
		storage:  storage,
		rulesets: rulesets,
		checks: []Check{
			ExifCheck{},
			NewKnownHashCheck(hashRepo),
			JPEGQuantCheck{},
			FrequencyCheck{},
			PatchNoiseCheck{},
			PRNUCheck{},
			GeometryCheck{},
		},
	}
}

func (d *Detector) Name() string { return DetectorName }

func (d *Detector) ExpectedWeight() float64 { return 1 }

// TakeRulesetVersion returns and clears the ruleset version recorded for a content analysis.
func (d *Detector) TakeRulesetVersion(contentID uuid.UUID) *int {
	if v, ok := d.versions.LoadAndDelete(contentID.String()); ok {
		ver := v.(int)
		if ver <= 0 {
			return nil
		}
		return &ver
	}
	return nil
}

func (d *Detector) Analyze(
	ctx context.Context,
	contentID uuid.UUID,
	objectKey string,
) ([]domaincontent.Signal, error) {
	ruleset := DefaultRuleset()
	if d.rulesets != nil {
		active, err := d.rulesets.Active(ctx)
		if err != nil {
			log.Printf("heuristic: load ruleset failed, using defaults: %v", err)
		} else {
			ruleset = active
		}
	}
	d.versions.Store(contentID.String(), ruleset.Version)

	if d.storage == nil {
		return nil, fmt.Errorf("heuristic: storage is required")
	}

	body, err := d.storage.Get(ctx, objectKey)
	if err != nil {
		return nil, fmt.Errorf("heuristic: download: %w", err)
	}
	defer body.Close()

	raw, err := io.ReadAll(io.LimitReader(body, domaincontent.MaxContentBytes+1))
	if err != nil {
		return nil, fmt.Errorf("heuristic: read: %w", err)
	}
	if int64(len(raw)) > domaincontent.MaxContentBytes {
		return nil, fmt.Errorf("heuristic: file too large")
	}

	img, format, err := decodeImage(raw)
	if err != nil {
		return nil, fmt.Errorf("heuristic: decode image: %w", err)
	}

	in := CheckInput{
		ContentID: contentID,
		ObjectKey: objectKey,
		RawBytes:  raw,
		Img:       img,
		Format:    format,
	}

	var signals []domaincontent.Signal
	for _, check := range d.checks {
		params := ruleset.ParamsFor(check.Name())
		if !params.Enabled {
			continue
		}
		sigs, err := check.Run(ctx, in, params)
		if err != nil {
			log.Printf("heuristic check failed check=%s err=%v", check.Name(), err)
			continue
		}
		signals = append(signals, sigs...)
		if ruleset.ShouldShortCircuit(signals) {
			break
		}
	}

	return signals, nil
}
