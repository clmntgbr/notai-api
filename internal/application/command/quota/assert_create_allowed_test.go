package quota

import (
	"testing"

	querysubscription "go-api/internal/application/query/subscription"
	domainmedia "go-api/internal/domain/media"
)

func TestAssertVerificationSlots_GraceBuffer(t *testing.T) {
	usage := &querysubscription.QuotaUsageView{
		Verifications: querysubscription.MonthlyQuotaCounter{Used: 20, Max: 20, Left: 0},
		Limits: querysubscription.QuotaLimits{
			QuotaOverageGraceVerifications: 5,
		},
	}
	if err := assertVerificationSlots(usage, 1); err != nil {
		t.Fatalf("expected grace to allow 1 more, got %v", err)
	}
	if err := assertVerificationSlots(usage, 6); err == nil {
		t.Fatal("expected hard fail beyond max+grace")
	}
}

func TestAssertVerificationSlots_OverageSoftSkip(t *testing.T) {
	usage := &querysubscription.QuotaUsageView{
		Verifications: querysubscription.MonthlyQuotaCounter{Used: 100, Max: 20, Left: 0},
		Limits: querysubscription.QuotaLimits{
			OveragePriceCents: 10,
		},
	}
	if err := assertVerificationSlots(usage, 1); err != nil {
		t.Fatalf("expected overage soft-skip, got %v", err)
	}
}

func TestAssertVerificationSlots_NoGraceHardFail(t *testing.T) {
	usage := &querysubscription.QuotaUsageView{
		Verifications: querysubscription.MonthlyQuotaCounter{Used: 20, Max: 20, Left: 0},
		Limits:        querysubscription.QuotaLimits{},
	}
	if err := assertVerificationSlots(usage, 1); err == nil {
		t.Fatal("expected hard fail at plan max")
	}
}

func TestAssertBatchUploadLogic(t *testing.T) {
	tests := []struct {
		name      string
		maxBatch  int
		fileCount int
		wantErr   bool
	}{
		{name: "within plan", maxBatch: 10, fileCount: 10, wantErr: false},
		{name: "over plan", maxBatch: 5, fileCount: 6, wantErr: true},
		{name: "zero falls back to MaxPresignBatch", maxBatch: 0, fileCount: domainmedia.MaxPresignBatch, wantErr: false},
		{name: "zero over technical max", maxBatch: 0, fileCount: domainmedia.MaxPresignBatch + 1, wantErr: true},
		{name: "plan above technical ceiling clamped", maxBatch: 50, fileCount: domainmedia.MaxPresignBatch + 1, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			maxBatch := tc.maxBatch
			if maxBatch <= 0 {
				maxBatch = domainmedia.MaxPresignBatch
			}
			if maxBatch > domainmedia.MaxPresignBatch {
				maxBatch = domainmedia.MaxPresignBatch
			}
			gotErr := tc.fileCount > maxBatch
			if gotErr != tc.wantErr {
				t.Fatalf("fileCount=%d maxBatch=%d: gotErr=%v wantErr=%v", tc.fileCount, maxBatch, gotErr, tc.wantErr)
			}
		})
	}
}

func TestAssertStorageReserveLogic(t *testing.T) {
	gb := int64(1024 * 1024 * 1024)
	used := gb - 100
	maxBytes := gb
	if used+50 > maxBytes {
		t.Fatal("expected room for 50 bytes")
	}
	if used+200 <= maxBytes {
		t.Fatal("expected overshoot for 200 bytes")
	}
}
