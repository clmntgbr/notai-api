package port

import (
	"context"
	"io"
)

type Thumbnailer interface {
	GenerateJPEG(ctx context.Context, src io.Reader, maxWidth int) ([]byte, error)
}
