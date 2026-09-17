package subscription

import (
	"context"
	"errors"
	"time"

	domaincampaign "go-api/internal/domain/campaign"
	domainclient "go-api/internal/domain/client"
	domaincontent "go-api/internal/domain/content"
	domainplan "go-api/internal/domain/plan"
	domainquota "go-api/internal/domain/quota"
	domainsubscription "go-api/internal/domain/subscription"
	domainworkspace "go-api/internal/domain/workspace"

	"github.com/google/uuid"
)

type GetQuotaUsageQuery struct {
	ClientID uuid.UUID
}

type GetQuotaUsageHandler struct {
	clientRepo       domainclient.ClientReadRepository
	workspaceRepo    domainworkspace.WorkspaceReadRepository
	subscriptionRepo domainsubscription.SubscriptionReadRepository
	planRepo         domainplan.PlanReadRepository
	campaignRepo     domaincampaign.CampaignReadRepository
	contentRepo      domaincontent.ContentReadRepository
}

func NewGetQuotaUsageHandler(
	clientRepo domainclient.ClientReadRepository,
	workspaceRepo domainworkspace.WorkspaceReadRepository,
	subscriptionRepo domainsubscription.SubscriptionReadRepository,
	planRepo domainplan.PlanReadRepository,
	campaignRepo domaincampaign.CampaignReadRepository,
	contentRepo domaincontent.ContentReadRepository,
) *GetQuotaUsageHandler {
	return &GetQuotaUsageHandler{
		clientRepo:       clientRepo,
		workspaceRepo:    workspaceRepo,
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		campaignRepo:     campaignRepo,
		contentRepo:      contentRepo,
	}
}

func (h *GetQuotaUsageHandler) Handle(ctx context.Context, q GetQuotaUsageQuery) (*QuotaUsageView, error) {
	client, err := h.clientRepo.FindByID(ctx, q.ClientID)
	if err != nil {
		return nil, errors.New("failed to get client")
	}
	if client == nil {
		return nil, ErrSubscriptionNotFound
	}

	workspace, err := h.workspaceRepo.FindByID(ctx, client.WorkspaceID)
	if err != nil {
		return nil, errors.New("failed to get workspace")
	}
	if workspace == nil || workspace.SubscriptionID == nil {
		return nil, ErrSubscriptionNotFound
	}

	sub, err := h.subscriptionRepo.FindByID(ctx, *workspace.SubscriptionID)
	if err != nil {
		return nil, errors.New("failed to get subscription")
	}
	if sub == nil {
		return nil, ErrSubscriptionNotFound
	}
	if sub.Plan == nil || sub.Plan.Quota == nil {
		return nil, errors.New("subscription plan quota not found")
	}

	quota := sub.Plan.Quota
	if sub.Status != domainsubscription.StatusActive {
		freePlan, err := h.planRepo.FindBySlug(ctx, domainplan.FreePlanSlug)
		if err != nil || freePlan == nil || freePlan.Quota == nil {
			return nil, errors.New("free plan quota not found")
		}
		quota = freePlan.Quota
	}

	return h.buildUsage(ctx, client, workspace.ID, sub, quota)
}

func (h *GetQuotaUsageHandler) buildUsage(
	ctx context.Context,
	client *domainclient.ClientView,
	workspaceID uuid.UUID,
	sub *domainsubscription.SubscriptionView,
	quota *domainquota.QuotaView,
) (*QuotaUsageView, error) {
	anchor := sub.QuotaPeriodStart
	if anchor.IsZero() {
		anchor = sub.StartDate
	}
	if anchor.IsZero() {
		anchor = sub.CreatedAt
	}

	periodStart, periodEnd := domainsubscription.CurrentQuotaPeriod(anchor, time.Now().UTC())
	membersUsed := int64(len(client.MemberIDs))

	campaignsUsed, err := h.campaignRepo.CountNonDefaultByWorkspaceID(ctx, workspaceID)
	if err != nil {
		return nil, errors.New("failed to count campaigns")
	}

	verificationsUsed, err := h.contentRepo.CountQuotaUnitsByWorkspaceIDInPeriod(
		ctx,
		workspaceID,
		periodStart,
		periodEnd,
	)
	if err != nil {
		return nil, errors.New("failed to count verifications")
	}

	concurrentUsed, err := h.contentRepo.CountByWorkspaceIDAndStatus(
		ctx,
		workspaceID,
		string(domaincontent.StatusAnalyzing),
	)
	if err != nil {
		return nil, errors.New("failed to count concurrent analyses")
	}

	return &QuotaUsageView{
		WorkspaceID: workspaceID,
		Members: QuotaCounter{
			Used: membersUsed,
			Max:  quota.MaxClientMembers,
			Left: quotaLeft(quota.MaxClientMembers, membersUsed),
		},
		Campaigns: QuotaCounter{
			Used: campaignsUsed,
			Max:  quota.MaxCampaigns,
			Left: quotaLeft(quota.MaxCampaigns, campaignsUsed),
		},
		Verifications: MonthlyQuotaCounter{
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			Used:        verificationsUsed,
			Max:         quota.MaxVerificationsPerMonth,
			Left:        quotaLeft(quota.MaxVerificationsPerMonth, verificationsUsed),
		},
		ConcurrentAnalyses: QuotaCounter{
			Used: concurrentUsed,
			Max:  quota.MaxConcurrentAnalyses,
			Left: quotaLeft(quota.MaxConcurrentAnalyses, concurrentUsed),
		},
		Limits: QuotaLimits{
			MaxFileSizeMB:                  quota.MaxFileSizeMB,
			ReportRetentionDays:            quota.ReportRetentionDays,
			AllowsVideoAnalysis:            quota.AllowsVideoAnalysis,
			AllowsPDFExport:                quota.AllowsPDFExport,
			AllowsCSVExport:                quota.AllowsCSVExport,
			AllowsAPIAccess:                quota.AllowsAPIAccess,
			OveragePriceCents:              quota.OveragePriceCents,
			AnalysisPriority:               quota.AnalysisPriority,
			MaxDetectorsPerAnalysis:        quota.MaxDetectorsPerAnalysis,
			MaxFramesPerVideo:              quota.MaxFramesPerVideo,
			AllowsReanalysis:               quota.AllowsReanalysis,
			MaxStorageGB:                   quota.MaxStorageGB,
			MaxBatchUploadSize:             quota.MaxBatchUploadSize,
			FrameRetentionDays:             quota.FrameRetentionDays,
			AllowsCustomRuleset:            quota.AllowsCustomRuleset,
			AllowsWhiteLabelReport:         quota.AllowsWhiteLabelReport,
			AllowsWebhooks:                 quota.AllowsWebhooks,
			QuotaOverageGraceVerifications: quota.QuotaOverageGraceVerifications,
		},
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
	}, nil
}

func quotaLeft(max int, used int64) int64 {
	left := int64(max) - used
	if left < 0 {
		return 0
	}
	return left
}
