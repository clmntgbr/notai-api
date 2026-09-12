package content

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"go-api/internal/domain/analysisresult"
	domaincontent "go-api/internal/domain/content"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type AnalyzeContentCommand struct {
	ContentID uuid.UUID
}

type AnalyzeContentHandler struct {
	contentRepo            domaincontent.ContentWriteRepository
	resultRepo             analysisresult.WriteRepository
	outbox                 port.OutboxRepository
	detectors              []domaincontent.Detector
	maxConcurrentDetectors int
}

func NewAnalyzeContentHandler(
	contentRepo domaincontent.ContentWriteRepository,
	resultRepo analysisresult.WriteRepository,
	outbox port.OutboxRepository,
	detectors []domaincontent.Detector,
	maxConcurrentDetectors int,
) *AnalyzeContentHandler {
	if maxConcurrentDetectors <= 0 {
		maxConcurrentDetectors = 3
	}
	return &AnalyzeContentHandler{
		contentRepo:            contentRepo,
		resultRepo:             resultRepo,
		outbox:                 outbox,
		detectors:              detectors,
		maxConcurrentDetectors: maxConcurrentDetectors,
	}
}

func (h *AnalyzeContentHandler) Handle(ctx context.Context, cmd AnalyzeContentCommand) error {
	if cmd.ContentID == uuid.Nil {
		return errors.New("content id is required")
	}
	if len(h.detectors) == 0 {
		return errors.New("no detectors configured")
	}

	content, err := h.contentRepo.GetByID(ctx, cmd.ContentID)
	if err != nil {
		return err
	}
	if content == nil {
		return nil
	}

	// Already analyzed.
	if content.Status == domaincontent.StatusAnalyzed {
		return nil
	}

	switch content.Status {
	case domaincontent.StatusUploaded:
		if err := content.StartAnalysis(); err != nil {
			return err
		}
		if err := h.contentRepo.WithTransaction(ctx, func(txCtx context.Context) error {
			if err := h.contentRepo.Update(txCtx, content); err != nil {
				return err
			}
			return h.outbox.StoreEvents(txCtx, content.PullEvents())
		}); err != nil {
			return err
		}
	case domaincontent.StatusAnalyzing:
		// Resume after crash / retry.
	default:
		return nil
	}

	weights := make(map[string]float64, len(h.detectors))
	g, gctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, h.maxConcurrentDetectors)

	for _, d := range h.detectors {
		d := d
		weights[d.Name()] = d.ExpectedWeight()
		g.Go(func() error {
			sem <- struct{}{}
			defer func() { <-sem }()

			started := time.Now().UTC()
			signals, analyzeErr := d.Analyze(gctx, content.ID, content.ObjectKey)
			completed := time.Now().UTC()

			result, err := analysisresult.New(content.ID, d.Name(), started, completed, d.ExpectedWeight())
			if err != nil {
				return err
			}
			if analyzeErr != nil {
				log.Printf(
					"detector failed content_id=%s detector=%s err=%v",
					content.ID,
					d.Name(),
					analyzeErr,
				)
				result.MarkFailed(analyzeErr.Error())
			} else {
				result.MarkSuccess(signals)
			}

			// Persist immediately, independently of other detectors.
			return h.resultRepo.Upsert(ctx, result)
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	results, err := h.resultRepo.FindByContentID(ctx, content.ID)
	if err != nil {
		return err
	}
	for i := range results {
		if w, ok := weights[results[i].DetectorName]; ok {
			results[i].Weight = w
		}
	}

	verdict := aggregate(results)

	return h.contentRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		fresh, err := h.contentRepo.GetByID(txCtx, content.ID)
		if err != nil {
			return err
		}
		if fresh == nil {
			return nil
		}
		if fresh.Status != domaincontent.StatusAnalyzing {
			return nil
		}
		if err := fresh.RenderVerdict(verdict); err != nil {
			return err
		}
		if err := h.contentRepo.Update(txCtx, fresh); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, fresh.PullEvents())
	})
}

func aggregate(results []analysisresult.Result) domaincontent.Verdict {
	var signals []domaincontent.Signal
	var respondedWeight, totalWeight float64
	successCount := 0

	for _, r := range results {
		totalWeight += r.ExpectedWeight()
		if r.Status != analysisresult.StatusSuccess {
			continue
		}
		successCount++
		signals = append(signals, r.Signals...)
		respondedWeight += r.ExpectedWeight()
	}

	label, confidence := computeLabel(signals)

	if respondedWeight < totalWeight {
		signals = append(signals, domaincontent.Signal{
			Type: "meta",
			Code: "partial_sources",
			Description: fmt.Sprintf(
				"%d/%d sources responded",
				successCount,
				len(results),
			),
			Weight: 0,
		})
	}

	return domaincontent.Verdict{
		Label:      label,
		Confidence: confidence,
		Signals:    signals,
	}
}

func computeLabel(signals []domaincontent.Signal) (domaincontent.Label, float64) {
	var aiScore float64
	for _, s := range signals {
		if s.Code == "sightengine_genai" || s.Type == "model" {
			if s.Weight > aiScore {
				aiScore = s.Weight
			}
		}
		if s.Code == "suspicious_filename" && s.Weight > aiScore {
			aiScore = s.Weight
		}
	}

	switch {
	case aiScore >= 0.7:
		return domaincontent.LabelAIGenerated, aiScore
	case aiScore <= 0.3:
		conf := 1 - aiScore
		if conf > 1 {
			conf = 1
		}
		return domaincontent.LabelHuman, conf
	default:
		return domaincontent.LabelUncertain, 0.5
	}
}
