package handler

import "context"

type mediaObjectCreatedHandler interface {
	Handle(ctx context.Context, objectKey, contentType string, size int64) error
}
