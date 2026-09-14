package di

import (
	"time"

	mediacmd "go-api/internal/application/command/media"
	"go-api/internal/infrastructure/config"
	"go-api/internal/infrastructure/persistence/outbox"
	"go-api/internal/infrastructure/persistence/write"

	"gorm.io/gorm"
)

type Container struct {
	RecoverStaleProcessingMediasHandler *mediacmd.RecoverStaleProcessingMediasHandler
	StaleAfter                          time.Duration
	MaxAge                              time.Duration
}

func NewContainer(db *gorm.DB, env *config.Config) *Container {
	mediaWriteRepo := write.NewMediaWriteRepository(db)
	contentWriteRepo := write.NewContentWriteRepository(db)
	outboxRepo := outbox.NewRepository(db)

	staleAfter := env.SchedulerStaleAfter
	if staleAfter <= 0 {
		staleAfter = mediacmd.DefaultStaleAfter
	}
	maxAge := env.SchedulerStaleMaxAge
	if maxAge <= 0 {
		maxAge = mediacmd.DefaultMaxAge
	}

	return &Container{
		RecoverStaleProcessingMediasHandler: mediacmd.NewRecoverStaleProcessingMediasHandler(
			mediaWriteRepo,
			contentWriteRepo,
			outboxRepo,
			staleAfter,
			maxAge,
		),
		StaleAfter: staleAfter,
		MaxAge:     maxAge,
	}
}
