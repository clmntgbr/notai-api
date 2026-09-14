package quota

import (
	"context"
	"errors"

	querysubscription "go-api/internal/application/query/subscription"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

var (
	ErrMemberQuotaExceeded       = errors.New("Member quota exceeded for your current plan")
	ErrCampaignQuotaExceeded     = errors.New("Campaign quota exceeded for your current plan")
	ErrVerificationQuotaExceeded = errors.New("Verification quota exceeded for your current plan")
	ErrConcurrentQuotaExceeded   = errors.New("Concurrent analysis quota exceeded for your current plan")
	ErrFileSizeQuotaExceeded     = errors.New("File size exceeds the maximum allowed for your current plan")
	ErrVideoAnalysisNotAllowed   = errors.New("Video analysis is not available on your current plan")
)

type AssertCreateAllowedHandler struct {
	getQuotaUsage *querysubscription.GetQuotaUsageHandler
	locker        port.AnalysisQuotaLocker
}

func NewAssertCreateAllowedHandler(
	getQuotaUsage *querysubscription.GetQuotaUsageHandler,
	locker port.AnalysisQuotaLocker,
) *AssertCreateAllowedHandler {
	return &AssertCreateAllowedHandler{
		getQuotaUsage: getQuotaUsage,
		locker:        locker,
	}
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
	return assertVerificationSlots(usage, count)
}

func (h *AssertCreateAllowedHandler) AssertConcurrentAnalysis(ctx context.Context, clientID uuid.UUID) error {
	usage, err := h.getQuotaUsage.Handle(ctx, querysubscription.GetQuotaUsageQuery{ClientID: clientID})
	if err != nil {
		return err
	}
	return assertConcurrentSlot(usage)
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

// AssertAnalyze must run inside the same DB transaction that transitions the content
// to analyzing, so the workspace advisory lock is held until the slot is reserved.
func (h *AssertCreateAllowedHandler) AssertAnalyze(ctx context.Context, clientID uuid.UUID) error {
	if h.locker == nil {
		return errors.New("analysis quota locker is required")
	}
	usage, err := h.getQuotaUsage.Handle(ctx, querysubscription.GetQuotaUsageQuery{ClientID: clientID})
	if err != nil {
		return err
	}
	if err := h.locker.Lock(ctx, usage.WorkspaceID); err != nil {
		return err
	}
	usage, err = h.getQuotaUsage.Handle(ctx, querysubscription.GetQuotaUsageQuery{ClientID: clientID})
	if err != nil {
		return err
	}
	if err := assertVerificationSlots(usage, 1); err != nil {
		return err
	}
	return assertConcurrentSlot(usage)
}

func assertVerificationSlots(usage *querysubscription.QuotaUsageView, count int) error {
	if usage.Limits.OveragePriceCents > 0 {
		return nil
	}
	if usage.Verifications.Left < int64(count) {
		return ErrVerificationQuotaExceeded
	}
	return nil
}

func assertConcurrentSlot(usage *querysubscription.QuotaUsageView) error {
	if usage.ConcurrentAnalyses.Left <= 0 {
		return ErrConcurrentQuotaExceeded
	}
	return nil
}
