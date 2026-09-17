package port

import (
	"context"
)

// ExtractedFrame is one sampled video frame ready for upload/analysis.
type ExtractedFrame struct {
	Index       int
	TimestampMs int64
	JPEGBytes   []byte
}

// FrameExtractor samples frames evenly across a video.
type FrameExtractor interface {
	Extract(ctx context.Context, videoPath string, maxFrames int) ([]ExtractedFrame, error)
}

// DefaultFramesPerVideo is used when callers omit a plan-specific cap.
const DefaultFramesPerVideo = 10

// MaxHardFramesPerVideo is the absolute technical ceiling, even if the plan is higher.
const MaxHardFramesPerVideo = 30
