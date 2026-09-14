package presenter

import (
	"time"

	domainplan "go-api/internal/domain/plan"
	domainquota "go-api/internal/domain/quota"
)

type QuotaResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	MaxClientMembers         int `json:"maxClientMembers"`
	MaxCampaigns             int `json:"maxCampaigns"`
	MaxVerificationsPerMonth int `json:"maxVerificationsPerMonth"`
	MaxConcurrentAnalyses    int `json:"maxConcurrentAnalyses"`
	MaxFileSizeMB            int `json:"maxFileSizeMb"`
	ReportRetentionDays      int `json:"reportRetentionDays"`

	AllowsVideoAnalysis bool `json:"allowsVideoAnalysis"`
	AllowsPDFExport     bool `json:"allowsPdfExport"`
	AllowsCSVExport     bool `json:"allowsCsvExport"`
	AllowsAPIAccess     bool `json:"allowsApiAccess"`
	OveragePriceCents   int  `json:"overagePriceCents"`
	AnalysisPriority    int  `json:"analysisPriority"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type PlanResponse struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Description     *string       `json:"description"`
	Slug            string        `json:"slug"`
	StripePriceID   *string       `json:"stripePriceId"`
	IsActive        bool          `json:"isActive"`
	BillingInterval string        `json:"billingInterval"`
	Price           float64       `json:"price"`
	Currency        string        `json:"currency"`
	QuotaID         string        `json:"quotaId"`
	Quota           QuotaResponse `json:"quota"`
	CreatedAt       time.Time     `json:"createdAt"`
	UpdatedAt       time.Time     `json:"updatedAt"`
}

func NewQuotaResponseFromView(view domainquota.QuotaView) QuotaResponse {
	return QuotaResponse{
		ID:                       view.ID.String(),
		Name:                     view.Name,
		MaxClientMembers:         view.MaxClientMembers,
		MaxCampaigns:             view.MaxCampaigns,
		MaxVerificationsPerMonth: view.MaxVerificationsPerMonth,
		MaxConcurrentAnalyses:    view.MaxConcurrentAnalyses,
		MaxFileSizeMB:            view.MaxFileSizeMB,
		ReportRetentionDays:      view.ReportRetentionDays,
		AllowsVideoAnalysis:      view.AllowsVideoAnalysis,
		AllowsPDFExport:          view.AllowsPDFExport,
		AllowsCSVExport:          view.AllowsCSVExport,
		AllowsAPIAccess:          view.AllowsAPIAccess,
		OveragePriceCents:        view.OveragePriceCents,
		AnalysisPriority:         view.AnalysisPriority,
		CreatedAt:                view.CreatedAt,
		UpdatedAt:                view.UpdatedAt,
	}
}

func NewPlanResponseFromView(view domainplan.PlanView) PlanResponse {
	resp := PlanResponse{
		ID:              view.ID.String(),
		Name:            view.Name,
		Description:     optionalNonEmptyString(view.Description),
		Slug:            view.Slug,
		StripePriceID:   optionalNonEmptyString(view.StripePriceID),
		IsActive:        view.IsActive,
		BillingInterval: string(view.BillingInterval),
		Price:           view.Price,
		Currency:        string(view.Currency),
		QuotaID:         view.QuotaID.String(),
		CreatedAt:       view.CreatedAt,
		UpdatedAt:       view.UpdatedAt,
	}
	if view.Quota != nil {
		resp.Quota = NewQuotaResponseFromView(*view.Quota)
	}
	return resp
}

func NewPlanListResponseFromViews(views []domainplan.PlanView) []PlanResponse {
	items := make([]PlanResponse, 0, len(views))
	for _, view := range views {
		items = append(items, NewPlanResponseFromView(view))
	}
	return items
}
