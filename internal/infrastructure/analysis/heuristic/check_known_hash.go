package heuristic

import (
	"context"
	"fmt"
	"image"

	domaincontent "go-api/internal/domain/content"

	"github.com/corona10/goimagehash"
)

type KnownHashCheck struct {
	repo KnownHashRepository
}

func NewKnownHashCheck(repo KnownHashRepository) *KnownHashCheck {
	return &KnownHashCheck{repo: repo}
}

func (c *KnownHashCheck) Name() string { return "known_hash" }

func (c *KnownHashCheck) Run(ctx context.Context, in CheckInput, p CheckParams) ([]domaincontent.Signal, error) {
	if c.repo == nil {
		return []domaincontent.Signal{
			skippedSignal("known_hash", "Known-hash repository not configured"),
		}, nil
	}
	if in.Img == nil {
		return []domaincontent.Signal{
			skippedSignal("known_hash", "No decoded image available"),
		}, nil
	}

	hash, err := goimagehash.PerceptionHash(toRGBA(in.Img))
	if err != nil {
		return nil, err
	}

	maxDistance := p.IntParam("hash_max_distance")
	if maxDistance <= 0 {
		maxDistance = 8
	}
	match, found, err := c.repo.FindClosest(ctx, fmt.Sprintf("%016x", hash.GetHash()), maxDistance)
	if err != nil {
		return nil, err
	}
	if !found {
		return []domaincontent.Signal{
			okSignal("known_hash", "No match in known AI corpus"),
		}, nil
	}

	source := match.Source
	if source == "" {
		source = "known-ai-corpus"
	}
	return []domaincontent.Signal{{
		Type: "heuristic",
		Code: "known_hash_match",
		Description: fmt.Sprintf(
			"Matches previously flagged content (%s, distance %d)",
			source,
			match.Distance,
		),
		Weight: p.WeightFor("known_hash_match"),
	}}, nil
}

func toRGBA(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}
	b := img.Bounds()
	out := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			out.Set(x, y, img.At(x, y))
		}
	}
	return out
}
