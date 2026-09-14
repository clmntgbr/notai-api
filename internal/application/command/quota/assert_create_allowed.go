package quota

import (
	"context"
	"errors"

	querysubscription "go-api/internal/application/query/subscription"
	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

var (
	ErrMemberQuotaExceeded       = errors.New("member quota exceeded for your current plan")
	ErrCampaignQuotaExceeded     = errors.New("campaign quota exceeded for your current plan")
	ErrVerificationQuotaExceeded = errors.New("verification quota exceeded for your current plan")
	ErrConcurrentQuotaExceeded   = errors.New("concurrent analysis quota exceeded for your current plan")
	ErrFileSizeQuotaExceeded     = errors.New("file size exceeds the maximum allowed for your current plan")
	ErrVideoAnalysisNotAllowed   = errors.New("video analysis is not available on your current plan")
)

type AssertCreateAllowedHandler struct {
	getQuotaUsage *querysubscription.GetQuotaUsageHandler
}

func NewAssertCreateAllowedHandler(
	getQuotaUsage *querysubscription.GetQuotaUsageHandler,
) *AssertCreateAllowedHandler {
	return &AssertCreateAllowedHandler{getQuotaUsage: getQuotaUsage}
}

func (h *AssertCreateAllowedHandler) AssertMemberCreate(ctx context.Context, clientID uuid.UUID) error {
	usage, err := h.getQuotaUsage.Handle(ctx, querysubscription.GetQuotaUsageQuery{ClientID: clientID})
	if err != nil {
		return err
	}
	if usage.Members.Left <= 0 {
		return ErrMemberQuotaExceeded
	}
	return nil
}

func (h *AssertCreateAllowedHandler) AssertCampaignCreate(ctx context.Context, clientID uuid.UUID) error {
	usage, err := h.getQuotaUsage.Handle(ctx, querysubscription.GetQuotaUsageQuery{ClientID: clientID})
	if err != nil {
		return err
	}
	if usage.Campaigns.Left <= 0 {
		return ErrCampaignQuotaExceeded
	}
	return nil
}

func (h *AssertCreateAllowedHandler) AssertVerificationCreate(ctx context.Context, clientID uuid.UUID) error {
	return h.AssertVerificationReserve(ctx, clientID, 1)
}

func (h *AssertCreateAllowedHandler) AssertVerificationReserve(
	ctx context.Context,
	clientID uuid.UUID,
	count int,
) error {
	if count <= 0 {
		return nil
	}
	usage, err := h.getQuotaUsage.Handle(ctx, querysubscription.GetQuotaUsageQuery{ClientID: clientID})
	if err != nil {
		return err
	}
	if usage.Limits.OveragePriceCents > 0 {
		return nil
	}
	if usage.Verifications.Left < int64(count) {
		return ErrVerificationQuotaExceeded
	}
	return nil
}

func (h *AssertCreateAllowedHandler) AssertConcurrentAnalysis(ctx context.Context, clientID uuid.UUID) error {
	usage, err := h.getQuotaUsage.Handle(ctx, querysubscription.GetQuotaUsageQuery{ClientID: clientID})
	if err != nil {
		return err
	}
	if usage.ConcurrentAnalyses.Left <= 0 {
		return ErrConcurrentQuotaExceeded
	}
	return nil
}

func (h *AssertCreateAllowedHandler) AssertFileSize(
	ctx context.Context,
	clientID uuid.UUID,
	sizeBytes int64,
) error {
	usage, err := h.getQuotaUsage.Handle(ctx, querysubscription.GetQuotaUsageQuery{ClientID: clientID})
	if err != nil {
		return err
	}
	maxBytes := int64(usage.Limits.MaxFileSizeMB) * 1024 * 1024
	if maxBytes > 0 && sizeBytes > maxBytes {
		return ErrFileSizeQuotaExceeded
	}
	return nil
}

func (h *AssertCreateAllowedHandler) AssertMediaUpload(
	ctx context.Context,
	clientID uuid.UUID,
	mediaType domainmedia.MediaType,
	sizeBytes *int64,
) error {
	usage, err := h.getQuotaUsage.Handle(ctx, querysubscription.GetQuotaUsageQuery{ClientID: clientID})
	if err != nil {
		return err
	}
	if mediaType == domainmedia.MediaTypeVideo && !usage.Limits.AllowsVideoAnalysis {
		return ErrVideoAnalysisNotAllowed
	}
	if sizeBytes != nil {
		maxBytes := int64(usage.Limits.MaxFileSizeMB) * 1024 * 1024
		if maxBytes > 0 && *sizeBytes > maxBytes {
			return ErrFileSizeQuotaExceeded
		}
	}
	return nil
}

func (h *AssertCreateAllowedHandler) AssertAnalyze(ctx context.Context, clientID uuid.UUID) error {
	if err := h.AssertVerificationCreate(ctx, clientID); err != nil {
		return err
	}
	return h.AssertConcurrentAnalysis(ctx, clientID)
}
