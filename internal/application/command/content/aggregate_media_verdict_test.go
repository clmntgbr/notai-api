package content

import (
	"testing"

	domaincontent "go-api/internal/domain/content"
)

func TestAggregateMediaVerdict_ImageHuman(t *testing.T) {
	label := domaincontent.LabelHuman
	v := AggregateMediaVerdict([]domaincontent.Content{{
		Status: domaincontent.StatusAnalyzed,
		Label:  &label,
	}})
	if v.Label != domaincontent.LabelHuman || v.TotalCount != 1 || v.FlaggedCount != 0 {
		t.Fatalf("unexpected verdict: %+v", v)
	}
}

func TestAggregateMediaVerdict_AnyAIFlags(t *testing.T) {
	human := domaincontent.LabelHuman
	ai := domaincontent.LabelAIGenerated
	v := AggregateMediaVerdict([]domaincontent.Content{
		{Status: domaincontent.StatusAnalyzed, Label: &human},
		{Status: domaincontent.StatusAnalyzed, Label: &ai},
	})
	if v.Label != domaincontent.LabelAIGenerated || v.FlaggedCount != 1 || v.TotalCount != 2 {
		t.Fatalf("unexpected verdict: %+v", v)
	}
}

func TestAggregateMediaVerdict_FailedCounts(t *testing.T) {
	human := domaincontent.LabelHuman
	v := AggregateMediaVerdict([]domaincontent.Content{
		{Status: domaincontent.StatusAnalyzed, Label: &human},
		{Status: domaincontent.StatusFailed},
	})
	if v.Label != domaincontent.LabelUncertain || v.FailedCount != 1 {
		t.Fatalf("unexpected verdict: %+v", v)
	}
}
