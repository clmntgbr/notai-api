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

// FrameExtractor samples frames from a local video file path.
type FrameExtractor interface {
	Extract(ctx context.Context, videoPath string, intervalMs int, maxFrames int) ([]ExtractedFrame, error)
}
