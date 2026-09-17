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
	ErrVerificationQuotaExceeded = errors.New("Quota exceeded for your current plan")
	ErrConcurrentQuotaExceeded   = errors.New("Concurrent analysis quota exceeded for your current plan")
	ErrFileSizeQuotaExceeded     = errors.New("File size exceeds the maximum allowed for your current plan")
	ErrVideoAnalysisNotAllowed   = errors.New("Video analysis is not available on your current plan")
	ErrBatchUploadQuotaExceeded  = errors.New("Batch upload size exceeds the maximum allowed for your current plan")
	ErrStorageQuotaExceeded      = errors.New("Storage quota exceeded for your current plan")
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

func (h *AssertCreateAllowedHandler) AssertBatchUpload(
	ctx context.Context,
	clientID uuid.UUID,
	fileCount int,
) error {
	if fileCount <= 0 {
		return nil
	}
	usage, err := h.getQuotaUsage.Handle(ctx, querysubscription.GetQuotaUsageQuery{ClientID: clientID})
	if err != nil {
		return err
	}
	maxBatch := usage.Limits.MaxBatchUploadSize
	if maxBatch <= 0 {
		maxBatch = domainmedia.MaxPresignBatch
	}
	if maxBatch > domainmedia.MaxPresignBatch {
		maxBatch = domainmedia.MaxPresignBatch
	}
	if fileCount > maxBatch {
		return ErrBatchUploadQuotaExceeded
	}
	return nil
}

func (h *AssertCreateAllowedHandler) AssertStorageReserve(
	ctx context.Context,
	clientID uuid.UUID,
	additionalBytes int64,
) error {
	if additionalBytes <= 0 {
		return nil
	}
	usage, err := h.getQuotaUsage.Handle(ctx, querysubscription.GetQuotaUsageQuery{ClientID: clientID})
	if err != nil {
		return err
	}
	maxBytes := int64(usage.Limits.MaxStorageGB) * 1024 * 1024 * 1024
	if maxBytes <= 0 {
		return nil
	}
	if usage.Storage.Used+additionalBytes > maxBytes {
		return ErrStorageQuotaExceeded
	}
	return nil
}

// Usage returns the current quota usage snapshot for the client's billing workspace.
func (h *AssertCreateAllowedHandler) Usage(
	ctx context.Context,
	clientID uuid.UUID,
) (*querysubscription.QuotaUsageView, error) {
	return h.getQuotaUsage.Handle(ctx, querysubscription.GetQuotaUsageQuery{ClientID: clientID})
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
	grace := usage.Limits.QuotaOverageGraceVerifications
	if grace < 0 {
		grace = 0
	}
	effectiveLeft := int64(usage.Verifications.Max) + int64(grace) - usage.Verifications.Used
	if effectiveLeft < int64(count) {
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
