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

// FramesPerVideo is the fixed number of frames extracted from each video,
// spaced evenly from start to end.
const FramesPerVideo = 10

// Extractor runs ffmpeg to sample JPEG frames from a video file.
type Extractor struct {
	Binary       string
	ProbeBinary  string
	FramesPerVid int
}

func NewExtractor() *Extractor {
	return &Extractor{
		Binary:       "ffmpeg",
		ProbeBinary:  "ffprobe",
		FramesPerVid: FramesPerVideo,
	}
}

func (e *Extractor) Extract(ctx context.Context, videoPath string) ([]port.ExtractedFrame, error) {
	n := e.FramesPerVid
	if n <= 0 {
		n = FramesPerVideo
	}

	durationMs, err := e.probeDurationMs(ctx, videoPath)
	if err != nil {
		return nil, err
	}
	if durationMs <= 0 {
		return nil, fmt.Errorf("ffmpeg extract: invalid duration %d", durationMs)
	}

	outDir, err := os.MkdirTemp("", "frames-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(outDir)

	// Sample n frames evenly across the whole duration.
	fps := float64(n) / (float64(durationMs) / 1000.0)
	pattern := filepath.Join(outDir, "frame_%03d.jpg")
	cmd := exec.CommandContext(
		ctx,
		e.Binary,
		"-hide_banner",
		"-loglevel", "error",
		"-i", videoPath,
		"-vf", fmt.Sprintf("fps=%.8f", fps),
		"-frames:v", strconv.Itoa(n),
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

	frames := make([]port.ExtractedFrame, 0, n)
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
			TimestampMs: timestampForIndex(idx, len(entries), durationMs),
			JPEGBytes:   raw,
		})
	}
	if len(frames) == 0 {
		return nil, fmt.Errorf("ffmpeg extract: no frames produced")
	}
	// Recompute timestamps with actual frame count (may be < n on very short clips).
	for i := range frames {
		frames[i].TimestampMs = timestampForIndex(i, len(frames), durationMs)
	}
	return frames, nil
}

func timestampForIndex(index, count int, durationMs int64) int64 {
	if count <= 1 {
		return 0
	}
	return int64(float64(index) * float64(durationMs) / float64(count-1))
}

func (e *Extractor) probeDurationMs(ctx context.Context, videoPath string) (int64, error) {
	bin := e.ProbeBinary
	if bin == "" {
		bin = "ffprobe"
	}
	cmd := exec.CommandContext(
		ctx,
		bin,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("ffprobe duration: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	sec, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return 0, fmt.Errorf("ffprobe duration parse: %w", err)
	}
	return int64(sec * 1000), nil
}
