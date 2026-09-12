package analysis

import (
	domaincontent "go-api/internal/domain/content"
	"go-api/internal/domain/port"
	"go-api/internal/infrastructure/analysis/heuristic"
	"go-api/internal/infrastructure/analysis/metadata"
	"go-api/internal/infrastructure/config"
	"go-api/internal/infrastructure/sightengine"

	"gorm.io/gorm"
)

// BuildDetectors returns always-on local detectors plus AI providers that have credentials.
func BuildDetectors(cfg *config.Config, storage port.Storage, db *gorm.DB) []domaincontent.Detector {
	detectors := []domaincontent.Detector{
		metadata.New(),
		heuristic.New(storage, heuristic.NewDBRulesetStore(db), heuristic.NewKnownHashRepository(db)),
	}

	if cfg.SightengineAPIUser != "" && cfg.SightengineAPISecret != "" {
		detectors = append(detectors, sightengine.New(sightengine.Config{
			APIURL:    cfg.SightengineAPIURL,
			APIUser:   cfg.SightengineAPIUser,
			APISecret: cfg.SightengineAPISecret,
		}, storage))
	}

	return detectors
}
