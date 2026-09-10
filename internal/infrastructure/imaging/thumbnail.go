package imaging

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp"
)

type Thumbnailer struct{}

func NewThumbnailer() *Thumbnailer {
	return &Thumbnailer{}
}

func (t *Thumbnailer) GenerateJPEG(_ context.Context, src io.Reader, maxWidth int) ([]byte, error) {
	img, err := imaging.Decode(src, imaging.AutoOrientation(true))
	if err != nil {
		return nil, errors.New("failed to decode image")
	}

	thumb := imaging.Resize(img, maxWidth, 0, imaging.Lanczos)

	var buf bytes.Buffer
	if err := imaging.Encode(&buf, thumb, imaging.JPEG, imaging.JPEGQuality(80)); err != nil {
		return nil, errors.New("failed to encode thumbnail")
	}

	return buf.Bytes(), nil
}
