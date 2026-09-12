package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"go-api/internal/domain/port"
)

// Extractor runs ffmpeg to sample JPEG frames from a video file.
type Extractor struct {
	Binary string
}

func NewExtractor() *Extractor {
	return &Extractor{Binary: "ffmpeg"}
}

func (e *Extractor) Extract(
	ctx context.Context,
	videoPath string,
	intervalMs int,
	maxFrames int,
) ([]port.ExtractedFrame, error) {
	if intervalMs <= 0 {
		intervalMs = 2000
	}
	if maxFrames <= 0 {
		maxFrames = 12
	}

	outDir, err := os.MkdirTemp("", "frames-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(outDir)

	fps := 1000.0 / float64(intervalMs)
	pattern := filepath.Join(outDir, "frame_%03d.jpg")
	cmd := exec.CommandContext(
		ctx,
		e.Binary,
		"-hide_banner",
		"-loglevel", "error",
		"-i", videoPath,
		"-vf", fmt.Sprintf("fps=%.6f", fps),
		"-frames:v", strconv.Itoa(maxFrames),
		"-q:v", "2",
		pattern,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg extract: %w (%s)", err, strings.TrimSpace(string(out)))
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		return nil, err
	}

	frames := make([]port.ExtractedFrame, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jpg") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(outDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		idx := len(frames)
		frames = append(frames, port.ExtractedFrame{
			Index:       idx,
			TimestampMs: int64(idx * intervalMs),
			JPEGBytes:   raw,
		})
	}
	if len(frames) == 0 {
		return nil, fmt.Errorf("ffmpeg extract: no frames produced")
	}
	return frames, nil
}
