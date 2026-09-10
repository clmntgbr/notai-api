package testutil

import "github.com/google/uuid"

// Stable UUIDs shared across handler tests.
var (
	TestUserID     = uuid.MustParse("01960000-0000-7000-8000-000000000001")
	TestClientID   = uuid.MustParse("01960000-0000-7000-8000-000000000002")
	TestCampaignID = uuid.MustParse("01960000-0000-7000-8000-000000000003")
	TestProjectID  = TestClientID // alias for older test helpers
)
