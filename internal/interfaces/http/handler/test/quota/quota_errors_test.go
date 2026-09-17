package quotatest

import (
	"errors"
	"net/http"
	"testing"

	cmdquota "go-api/internal/application/command/quota"
	querysubscription "go-api/internal/application/query/subscription"
	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/testutil"

	"github.com/gofiber/fiber/v3"
)

func TestRespondQuotaError_Mapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{
			name:       "subscription not found",
			err:        querysubscription.ErrSubscriptionNotFound,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "member quota exceeded",
			err:        cmdquota.ErrMemberQuotaExceeded,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "campaign quota exceeded",
			err:        cmdquota.ErrCampaignQuotaExceeded,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "verification quota exceeded",
			err:        cmdquota.ErrVerificationQuotaExceeded,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "concurrent quota exceeded",
			err:        cmdquota.ErrConcurrentQuotaExceeded,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "file size quota exceeded",
			err:        cmdquota.ErrFileSizeQuotaExceeded,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "video analysis not allowed",
			err:        cmdquota.ErrVideoAnalysisNotAllowed,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "batch upload quota exceeded",
			err:        cmdquota.ErrBatchUploadQuotaExceeded,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "storage quota exceeded",
			err:        cmdquota.ErrStorageQuotaExceeded,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "nil error",
			err:        nil,
			wantStatus: 0,
		},
		{
			name:       "unmapped error",
			err:        errors.New("something else"),
			wantStatus: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := testutil.NewTestApp()
			app.Get("/quota", func(c fiber.Ctx) error {
				handled, err := handler.RespondQuotaErrorForTest(c, tc.err)
				if err != nil {
					return err
				}
				if handled {
					return nil
				}
				return c.SendStatus(http.StatusTeapot)
			})

			req, err := testutil.JSONRequest(http.MethodGet, "/quota", nil)
			if err != nil {
				t.Fatalf("build request: %v", err)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("perform request: %v", err)
			}

			if tc.wantStatus == 0 {
				if resp.StatusCode != http.StatusTeapot {
					t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusTeapot)
				}
				return
			}
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("status: got %d want %d", resp.StatusCode, tc.wantStatus)
			}
		})
	}
}
