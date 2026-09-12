package heuristic

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type KnownHashMatch struct {
	Source   string
	Label    string
	Distance int
}

type KnownHashRepository interface {
	FindClosest(ctx context.Context, phash string, maxDistance int) (KnownHashMatch, bool, error)
}

type knownHashRow struct {
	PHash  string `gorm:"column:phash"`
	Source string `gorm:"column:source"`
	Label  string `gorm:"column:label"`
}

func (knownHashRow) TableName() string { return "known_ai_content_hashes" }

type dbKnownHashRepository struct {
	db *gorm.DB
}

func NewKnownHashRepository(db *gorm.DB) KnownHashRepository {
	return &dbKnownHashRepository{db: db}
}

func (r *dbKnownHashRepository) FindClosest(
	ctx context.Context,
	phash string,
	maxDistance int,
) (KnownHashMatch, bool, error) {
	if phash == "" {
		return KnownHashMatch{}, false, nil
	}
	if maxDistance < 0 {
		maxDistance = 0
	}

	var rows []knownHashRow
	if err := r.db.WithContext(ctx).Find(&rows).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return KnownHashMatch{}, false, nil
		}
		return KnownHashMatch{}, false, err
	}

	best := KnownHashMatch{Distance: maxDistance + 1}
	found := false
	for _, row := range rows {
		d := hammingHex(phash, row.PHash)
		if d < 0 || d > maxDistance {
			continue
		}
		if !found || d < best.Distance {
			best = KnownHashMatch{Source: row.Source, Label: row.Label, Distance: d}
			found = true
		}
	}
	return best, found, nil
}

func hammingHex(a, b string) int {
	if len(a) != len(b) {
		return -1
	}
	dist := 0
	for i := 0; i < len(a); i++ {
		x := hexNibble(a[i]) ^ hexNibble(b[i])
		for x > 0 {
			dist += x & 1
			x >>= 1
		}
	}
	return dist
}

func hexNibble(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	default:
		return 0
	}
}
