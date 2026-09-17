package handler

import (
	"errors"
	"fmt"
	"strings"
	"time"

	domaincontent "go-api/internal/domain/content"
	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

func parseMediaStatusFilters(values []string) ([]string, error) {
	values = trimQueryValues(values)
	if len(values) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		switch domainmedia.Status(value) {
		case domainmedia.StatusPendingUpload,
			domainmedia.StatusUploaded,
			domainmedia.StatusProcessing,
			domainmedia.StatusAnalyzed,
			domainmedia.StatusFailed:
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			out = append(out, value)
		default:
			return nil, fmt.Errorf("unsupported status %q", value)
		}
	}
	return out, nil
}

func parseMediaVerdictFilters(values []string) ([]string, error) {
	values = trimQueryValues(values)
	if len(values) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		label := domaincontent.Label(value)
		if err := domaincontent.ValidateLabel(label); err != nil {
			return nil, fmt.Errorf("unsupported verdict %q", value)
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out, nil
}

func parseUUIDFilters(values []string) ([]uuid.UUID, error) {
	values = trimQueryValues(values)
	if len(values) == 0 {
		return nil, nil
	}
	out := make([]uuid.UUID, 0, len(values))
	seen := make(map[uuid.UUID]struct{}, len(values))
	for _, value := range values {
		id, err := uuid.Parse(value)
		if err != nil {
			return nil, fmt.Errorf("invalid uuid %q", value)
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

func trimQueryValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

// parseOptionalMediaPeriod resolves optional from/to query params for list endpoints.
// Neither set → no date filter (nil, nil).
// from only → [from, now]; to only → (-∞, to]; from+to → [from, to].
func parseOptionalMediaPeriod(rawFrom, rawTo string) (*time.Time, *time.Time, error) {
	rawFrom = strings.TrimSpace(rawFrom)
	rawTo = strings.TrimSpace(rawTo)

	if rawFrom == "" && rawTo == "" {
		return nil, nil, nil
	}

	var from *time.Time
	if rawFrom != "" {
		parsed, err := parseStatsQueryTime(rawFrom, false)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid from: %w", err)
		}
		from = &parsed
	}

	var to *time.Time
	if rawTo != "" {
		parsed, err := parseStatsQueryTime(rawTo, true)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid to: %w", err)
		}
		to = &parsed
	} else {
		now := time.Now().UTC()
		to = &now
	}

	if from != nil && from.After(*to) {
		return nil, nil, errors.New("from must be before to")
	}
	return from, to, nil
}

// parseMediaStatsPeriod resolves GET /medias/stats from/to query params.
// Both bounds are inclusive. Default (neither set): current UTC calendar month
// [monthStart, lastInstantOfMonth]. from only: [from, now]. to only: error.
// Date-only to includes the whole calendar day.
func parseMediaStatsPeriod(rawFrom, rawTo string) (time.Time, time.Time, error) {
	rawFrom = strings.TrimSpace(rawFrom)
	rawTo = strings.TrimSpace(rawTo)

	if rawFrom == "" && rawTo != "" {
		return time.Time{}, time.Time{}, errors.New("from is required when to is provided")
	}

	now := time.Now().UTC()
	if rawFrom == "" && rawTo == "" {
		from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		to := from.AddDate(0, 1, 0).Add(-time.Nanosecond)
		return from, to, nil
	}

	from, err := parseStatsQueryTime(rawFrom, false)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid from: %w", err)
	}

	var to time.Time
	if rawTo == "" {
		to = now
	} else {
		to, err = parseStatsQueryTime(rawTo, true)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid to: %w", err)
		}
	}

	if from.After(to) {
		return time.Time{}, time.Time{}, errors.New("from must be before to")
	}
	return from, to, nil
}

func parseStatsQueryTime(raw string, isEnd bool) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		if isEnd {
			// Date-only `to` includes the whole calendar day.
			return day.AddDate(0, 0, 1).Add(-time.Nanosecond), nil
		}
		return day, nil
	}
	return time.Time{}, errors.New("use YYYY-MM-DD or RFC3339")
}
