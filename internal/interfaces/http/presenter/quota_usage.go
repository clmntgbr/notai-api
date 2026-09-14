package presenter

import (
	"time"

	querysubscription "go-api/internal/application/query/subscription"
)

type MonthlyQuotaCounterResponse struct {
	PeriodStart time.Time `json:"periodStart"`
	PeriodEnd   time.Time `json:"periodEnd"`
	Used        int64     `json:"used"`
	Max         int       `json:"max"`
	Left        int64     `json:"left"`
}

type QuotaCounterResponse struct {
	Used int64 `json:"used"`
	Max  int   `json:"max"`
	Left int64 `json:"left"`
}

type QuotaLimitsResponse struct {
	MaxFileSizeMB       int  `json:"maxFileSizeMb"`
	ReportRetentionDays int  `json:"reportRetentionDays"`
	AllowsVideoAnalysis bool `json:"allowsVideoAnalysis"`
	AllowsPDFExport     bool `json:"allowsPdfExport"`
	AllowsCSVExport     bool `json:"allowsCsvExport"`
	AllowsAPIAccess     bool `json:"allowsApiAccess"`
	OveragePriceCents   int  `json:"overagePriceCents"`
	AnalysisPriority    int  `json:"analysisPriority"`
}

type QuotaUsageResponse struct {
	Members            QuotaCounterResponse        `json:"members"`
	Campaigns          QuotaCounterResponse        `json:"campaigns"`
	Verifications      MonthlyQuotaCounterResponse `json:"verifications"`
	ConcurrentAnalyses QuotaCounterResponse        `json:"concurrentAnalyses"`
	Limits             QuotaLimitsResponse         `json:"limits"`
	PeriodStart        time.Time                   `json:"periodStart"`
	PeriodEnd          time.Time                   `json:"periodEnd"`
}

func NewQuotaUsageResponse(usage *querysubscription.QuotaUsageView) QuotaUsageResponse {
	return QuotaUsageResponse{
		Members: QuotaCounterResponse{
			Used: usage.Members.Used,
			Max:  usage.Members.Max,
			Left: usage.Members.Left,
		},
		Campaigns: QuotaCounterResponse{
			Used: usage.Campaigns.Used,
			Max:  usage.Campaigns.Max,
			Left: usage.Campaigns.Left,
		},
		Verifications: MonthlyQuotaCounterResponse{
			PeriodStart: usage.Verifications.PeriodStart,
			PeriodEnd:   usage.Verifications.PeriodEnd,
			Used:        usage.Verifications.Used,
			Max:         usage.Verifications.Max,
			Left:        usage.Verifications.Left,
		},
		ConcurrentAnalyses: QuotaCounterResponse{
			Used: usage.ConcurrentAnalyses.Used,
			Max:  usage.ConcurrentAnalyses.Max,
			Left: usage.ConcurrentAnalyses.Left,
		},
		Limits: QuotaLimitsResponse{
			MaxFileSizeMB:       usage.Limits.MaxFileSizeMB,
			ReportRetentionDays: usage.Limits.ReportRetentionDays,
			AllowsVideoAnalysis: usage.Limits.AllowsVideoAnalysis,
			AllowsPDFExport:     usage.Limits.AllowsPDFExport,
			AllowsCSVExport:     usage.Limits.AllowsCSVExport,
			AllowsAPIAccess:     usage.Limits.AllowsAPIAccess,
			OveragePriceCents:   usage.Limits.OveragePriceCents,
			AnalysisPriority:    usage.Limits.AnalysisPriority,
		},
		PeriodStart: usage.PeriodStart,
		PeriodEnd:   usage.PeriodEnd,
	}
}
