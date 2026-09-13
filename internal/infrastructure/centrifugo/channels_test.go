package centrifugo_test

import (
	"testing"

	"go-api/internal/infrastructure/centrifugo"

	"github.com/google/uuid"
)

func TestUserInterestChannel(t *testing.T) {
	id := uuid.MustParse("01960000-0000-7000-8000-000000000001")
	cases := []struct {
		interest string
		want     string
	}{
		{"", "users:01960000-0000-7000-8000-000000000001#01960000-0000-7000-8000-000000000001"},
		{centrifugo.InterestAccount, "users:01960000-0000-7000-8000-000000000001#01960000-0000-7000-8000-000000000001"},
		{centrifugo.InterestMedia, "users:01960000-0000-7000-8000-000000000001:media#01960000-0000-7000-8000-000000000001"},
		{centrifugo.InterestContent, "users:01960000-0000-7000-8000-000000000001:content#01960000-0000-7000-8000-000000000001"},
		{centrifugo.InterestActivity, "users:01960000-0000-7000-8000-000000000001:activity#01960000-0000-7000-8000-000000000001"},
	}
	for _, tc := range cases {
		got, err := centrifugo.UserInterestChannel(id, tc.interest)
		if err != nil {
			t.Fatalf("interest %q: %v", tc.interest, err)
		}
		if got != tc.want {
			t.Fatalf("interest %q: got %q want %q", tc.interest, got, tc.want)
		}
	}
	if _, err := centrifugo.UserInterestChannel(id, "nope"); err == nil {
		t.Fatal("expected unknown interest error")
	}
}
