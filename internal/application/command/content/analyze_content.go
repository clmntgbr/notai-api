package content

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	cmdquota "go-api/internal/application/command/quota"
	"go-api/internal/domain/analysisresult"
	domaincontent "go-api/internal/domain/content"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type AnalyzeContentCommand struct {
	ContentID uuid.UUID
}

type AnalyzeContentHandler struct {
	contentRepo            domaincontent.ContentWriteRepository
	mediaRepo              domainmedia.MediaWriteRepository
	resultRepo             analysisresult.WriteRepository
	outbox                 port.OutboxRepository
	detectors              []domaincontent.Detector
	maxConcurrentDetectors int
	quota                  *cmdquota.AssertCreateAllowedHandler
}

func NewAnalyzeContentHandler(
	contentRepo domaincontent.ContentWriteRepository,
	mediaRepo domainmedia.MediaWriteRepository,
	resultRepo analysisresult.WriteRepository,
	outbox port.OutboxRepository,
	detectors []domaincontent.Detector,
	maxConcurrentDetectors int,
	quota *cmdquota.AssertCreateAllowedHandler,
) *AnalyzeContentHandler {
	if maxConcurrentDetectors <= 0 {
		maxConcurrentDetectors = 3
	}
	return &AnalyzeContentHandler{
		contentRepo:            contentRepo,
		mediaRepo:              mediaRepo,
		resultRepo:             resultRepo,
		outbox:                 outbox,
		detectors:              detectors,
		maxConcurrentDetectors: maxConcurrentDetectors,
		quota:                  quota,
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

	if content.Status == domaincontent.StatusAnalyzed {
		return h.tryFinalizeMedia(ctx, content.MediaID)
	}

	switch content.Status {
	case domaincontent.StatusUploaded:
		media, err := h.mediaRepo.GetByID(ctx, content.MediaID)
		if err != nil {
			return err
		}
		if media == nil {
			return fmt.Errorf("media %s not found for content %s", content.MediaID, content.ID)
		}
		if err := h.contentRepo.WithTransaction(ctx, func(txCtx context.Context) error {
			fresh, err := h.contentRepo.GetByID(txCtx, content.ID)
			if err != nil {
				return err
			}
			if fresh == nil {
				return nil
			}
			if fresh.Status != domaincontent.StatusUploaded {
				*content = *fresh
				return nil
			}
			if h.quota != nil {
				if err := h.quota.AssertAnalyze(txCtx, media.ClientID); err != nil {
					return err
				}
			}
			if err := fresh.StartAnalysis(); err != nil {
				return err
			}
			if err := h.contentRepo.Update(txCtx, fresh); err != nil {
				return err
			}
			if err := h.outbox.StoreEvents(txCtx, fresh.PullEvents()); err != nil {
				return err
			}
			*content = *fresh
			return nil
		}); err != nil {
			if h.quota != nil && isHardAnalysisQuotaError(err) {
				if failErr := h.failQuotaBlockedAnalysis(ctx, content, media, err); failErr != nil {
					return failErr
				}
			}
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

	detectors := h.selectDetectors(ctx, content.ClientID)
	for _, d := range detectors {
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
			if tracker, ok := d.(domaincontent.RulesetVersionTracker); ok {
				result.RulesetVersion = tracker.TakeRulesetVersion(content.ID)
			}

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

	if err := h.contentRepo.WithTransaction(ctx, func(txCtx context.Context) error {
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
	}); err != nil {
		return err
	}

	return h.tryFinalizeMedia(ctx, content.MediaID)
}

func isHardAnalysisQuotaError(err error) bool {
	return errors.Is(err, cmdquota.ErrVerificationQuotaExceeded) ||
		errors.Is(err, cmdquota.ErrVideoAnalysisNotAllowed) ||
		errors.Is(err, cmdquota.ErrFileSizeQuotaExceeded) ||
		errors.Is(err, cmdquota.ErrStorageQuotaExceeded)
}

// selectDetectors keeps every local detector and at most N external ones
// (stable order: first external in the configured list wins when N >= 1).
// Quota lookup failures fail closed (zero external detectors).
func (h *AnalyzeContentHandler) selectDetectors(
	ctx context.Context,
	clientID uuid.UUID,
) []domaincontent.Detector {
	maxExternal := 0
	if h.quota != nil {
		usage, err := h.quota.Usage(ctx, clientID)
		if err == nil {
			maxExternal = usage.Limits.MaxDetectorsPerAnalysis
			if maxExternal < 0 {
				maxExternal = 0
			}
		}
	} else {
		return h.detectors
	}

	selected := make([]domaincontent.Detector, 0, len(h.detectors))
	externalKept := 0
	for _, d := range h.detectors {
		if domaincontent.IsExternal(d) {
			if externalKept >= maxExternal {
				continue
			}
			externalKept++
		}
		selected = append(selected, d)
	}
	return selected
}

func (h *AnalyzeContentHandler) failQuotaBlockedAnalysis(
	ctx context.Context,
	content *domaincontent.Content,
	media *domainmedia.Media,
	quotaErr error,
) error {
	reason := quotaErr.Error()
	if err := content.MarkFailed(); err != nil {
		return err
	}

	return h.mediaRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.contentRepo.Update(txCtx, content); err != nil {
			return err
		}
		events := content.PullEvents()

		fresh, err := h.mediaRepo.GetByID(txCtx, media.ID)
		if err != nil {
			return err
		}
		if fresh != nil &&
			fresh.Status != domainmedia.StatusAnalyzed &&
			fresh.Status != domainmedia.StatusFailed {
			if err := fresh.MarkFailed(reason); err != nil {
				return err
			}
			if err := h.mediaRepo.Update(txCtx, fresh); err != nil {
				return err
			}
			events = append(events, fresh.PullEvents()...)
		}
		return h.outbox.StoreEvents(txCtx, events)
	})
}

func (h *AnalyzeContentHandler) tryFinalizeMedia(ctx context.Context, mediaID uuid.UUID) error {
	if h.mediaRepo == nil || mediaID == uuid.Nil {
		return nil
	}

	return h.mediaRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		media, err := h.mediaRepo.GetByID(txCtx, mediaID)
		if err != nil {
			return err
		}
		if media == nil {
			return nil
		}
		if media.Status == domainmedia.StatusAnalyzed || media.Status == domainmedia.StatusFailed {
			return nil
		}

		siblings, err := h.contentRepo.ListByMediaID(txCtx, mediaID)
		if err != nil {
			return err
		}
		if len(siblings) == 0 {
			return nil
		}
		for _, c := range siblings {
			if !c.IsTerminal() {
				return nil
			}
		}

		verdict := AggregateMediaVerdict(siblings)
		if err := media.RenderGlobalVerdict(verdict); err != nil {
			return err
		}
		if err := h.mediaRepo.Update(txCtx, media); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, media.PullEvents())
	})
}

// AggregateMediaVerdict builds the parent media verdict from child contents.
func AggregateMediaVerdict(contents []domaincontent.Content) domainmedia.Verdict {
	total := len(contents)
	flagged := 0
	failed := 0
	uncertain := 0
	for _, c := range contents {
		if c.Status == domaincontent.StatusFailed {
			failed++
			continue
		}
		if c.Label == nil {
			uncertain++
			continue
		}
		switch *c.Label {
		case domaincontent.LabelAIGenerated:
			flagged++
		case domaincontent.LabelUncertain:
			uncertain++
		}
	}

	label := domaincontent.LabelHuman
	switch {
	case flagged > 0:
		label = domaincontent.LabelAIGenerated
	case failed > 0 || uncertain > 0:
		label = domaincontent.LabelUncertain
	}

	return domainmedia.Verdict{
		Label:        label,
		FlaggedCount: flagged,
		TotalCount:   total,
		FailedCount:  failed,
	}
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
		if s.Type == "meta" {
			continue
		}
		switch s.Type {
		case "model", "heuristic", "metadata":
			if s.Weight > aiScore {
				aiScore = s.Weight
			}
		default:
			if s.Code == "sightengine_genai" || s.Code == "suspicious_filename" {
				if s.Weight > aiScore {
					aiScore = s.Weight
				}
			}
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
