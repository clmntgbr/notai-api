package heuristic

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"

	_ "golang.org/x/image/webp"
)

func decodeImage(raw []byte) (image.Image, string, error) {
	img, format, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, "", err
	}
	return img, format, nil
}

func toGrayMatrix(img image.Image, size int) [][]float64 {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	out := make([][]float64, size)
	for y := 0; y < size; y++ {
		row := make([]float64, size)
		for x := 0; x < size; x++ {
			sx := bounds.Min.X + x*w/size
			sy := bounds.Min.Y + y*h/size
			r, g, b, _ := img.At(sx, sy).RGBA()
			// 16-bit channels → luminance
			row[x] = (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 65535.0
		}
		out[y] = row
	}
	return out
}

func laplacianVariance(img image.Image, x0, y0, x1, y1 int) float64 {
	bounds := img.Bounds()
	if x0 < bounds.Min.X {
		x0 = bounds.Min.X
	}
	if y0 < bounds.Min.Y {
		y0 = bounds.Min.Y
	}
	if x1 > bounds.Max.X {
		x1 = bounds.Max.X
	}
	if y1 > bounds.Max.Y {
		y1 = bounds.Max.Y
	}
	if x1-x0 < 3 || y1-y0 < 3 {
		return 0
	}

	var values []float64
	for y := y0 + 1; y < y1-1; y++ {
		for x := x0 + 1; x < x1-1; x++ {
			c := grayAt(img, x, y)
			up := grayAt(img, x, y-1)
			down := grayAt(img, x, y+1)
			left := grayAt(img, x-1, y)
			right := grayAt(img, x+1, y)
			lap := math.Abs(4*c - up - down - left - right)
			values = append(values, lap)
		}
	}
	if len(values) == 0 {
		return 0
	}
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))
	var sumSq float64
	for _, v := range values {
		d := v - mean
		sumSq += d * d
	}
	return sumSq / float64(len(values))
}

func grayAt(img image.Image, x, y int) float64 {
	r, g, b, _ := img.At(x, y).RGBA()
	return (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 65535.0
}

func meanStd(values []float64) (mean, std float64) {
	if len(values) == 0 {
		return 0, 0
	}
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))
	var sumSq float64
	for _, v := range values {
		d := v - mean
		sumSq += d * d
	}
	std = math.Sqrt(sumSq / float64(len(values)))
	return mean, std
}

// dftMagnitudePeaks returns a crude periodicity score from a small grayscale DFT.
func dftPeriodicityScore(gray [][]float64, peakMin float64) float64 {
	n := len(gray)
	if n == 0 || len(gray[0]) != n || n < 8 {
		return 0
	}

	type peak struct {
		mag float64
		u   int
		v   int
	}
	var peaks []peak
	maxMag := 0.0

	for u := 0; u < n; u++ {
		for v := 0; v < n; v++ {
			if u == 0 && v == 0 {
				continue
			}
			var re, im float64
			for y := 0; y < n; y++ {
				for x := 0; x < n; x++ {
					angle := -2 * math.Pi * (float64(u*x+v*y) / float64(n))
					re += gray[y][x] * math.Cos(angle)
					im += gray[y][x] * math.Sin(angle)
				}
			}
			mag := math.Hypot(re, im) / float64(n*n)
			if mag > maxMag {
				maxMag = mag
			}
			if mag >= peakMin {
				peaks = append(peaks, peak{mag: mag, u: u, v: v})
			}
		}
	}
	if maxMag == 0 || len(peaks) < 2 {
		return 0
	}

	// Score: share of energy in non-DC peaks relative to max.
	energy := 0.0
	for _, p := range peaks {
		energy += p.mag
	}
	return energy / (maxMag * float64(len(peaks)))
}

func hasJPEGExif(raw []byte) bool {
	// Look for APP1 (0xFFE1) which usually carries EXIF.
	for i := 0; i+4 < len(raw); i++ {
		if raw[i] == 0xFF && raw[i+1] == 0xE1 {
			if i+10 < len(raw) && string(raw[i+4:i+8]) == "Exif" {
				return true
			}
		}
	}
	return false
}

func hasC2PAMarker(raw []byte) (tool string, generative bool, found bool) {
	lower := bytes.ToLower(raw)
	if !bytes.Contains(lower, []byte("c2pa")) && !bytes.Contains(lower, []byte("jumb")) {
		return "", false, false
	}
	found = true
	generativeHints := []string{"trainedalgorithmicmedia", "generative", "synthetic", "ai generated", "created with ai"}
	for _, hint := range generativeHints {
		if bytes.Contains(lower, []byte(hint)) {
			return "c2pa", true, true
		}
	}
	return "c2pa", false, true
}

func ensureMinSize(img image.Image, min int) error {
	b := img.Bounds()
	if b.Dx() < min || b.Dy() < min {
		return fmt.Errorf("image too small (%dx%d)", b.Dx(), b.Dy())
	}
	return nil
}
