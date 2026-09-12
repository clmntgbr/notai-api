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

// FrameExtractor samples a fixed number of frames evenly across a video.
type FrameExtractor interface {
	Extract(ctx context.Context, videoPath string) ([]ExtractedFrame, error)
}
