package handler

import (
	"fmt"
	"strings"

	domaincontent "go-api/internal/domain/content"
	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

func parseMediaStatusFilters(raw string) ([]string, error) {
	values := splitCSVQuery(raw)
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

func parseMediaVerdictFilters(raw string) ([]string, error) {
	values := splitCSVQuery(raw)
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

func parseUUIDFilters(raw string) ([]uuid.UUID, error) {
	values := splitCSVQuery(raw)
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

func splitCSVQuery(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}
