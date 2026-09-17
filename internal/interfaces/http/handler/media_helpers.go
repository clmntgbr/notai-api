package handler

import (
	"fmt"
	"strings"

	domaincontent "go-api/internal/domain/content"
	domainmedia "go-api/internal/domain/media"
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
