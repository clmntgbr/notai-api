package heuristic

import (
	"context"
	"log"
	"sync"
	"time"

	"go-api/internal/infrastructure/persistence/dbtype"

	"gorm.io/gorm"
)

type RulesetStore interface {
	Active(ctx context.Context) (RulesetConfig, error)
}

type DBRulesetStore struct {
	db       *gorm.DB
	mu       sync.RWMutex
	cached   RulesetConfig
	loadedAt time.Time
	ttl      time.Duration
}

func NewDBRulesetStore(db *gorm.DB) *DBRulesetStore {
	return &DBRulesetStore{db: db, ttl: 2 * time.Minute}
}

type heuristicRulesetRow struct {
	Version int
	Params  dbtype.JSONB
}

func (heuristicRulesetRow) TableName() string { return "heuristic_rulesets" }

func (s *DBRulesetStore) Active(ctx context.Context) (RulesetConfig, error) {
	s.mu.RLock()
	if !s.loadedAt.IsZero() && time.Since(s.loadedAt) < s.ttl {
		cfg := s.cached
		s.mu.RUnlock()
		return cfg, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.loadedAt.IsZero() && time.Since(s.loadedAt) < s.ttl {
		return s.cached, nil
	}

	var row heuristicRulesetRow
	err := s.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("version DESC").
		First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			cfg := DefaultRuleset()
			s.cached = cfg
			s.loadedAt = time.Now()
			log.Printf("heuristic ruleset: no active row, using embedded defaults")
			return cfg, nil
		}
		return RulesetConfig{}, err
	}

	cfg, err := ParseRulesetParams(row.Version, row.Params)
	if err != nil {
		return RulesetConfig{}, err
	}
	s.cached = cfg
	s.loadedAt = time.Now()
	return cfg, nil
}
