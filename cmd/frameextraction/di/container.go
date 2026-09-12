package di

import (
	"log"

	mediacmd "go-api/internal/application/command/media"
	eventmedia "go-api/internal/application/event/media"
	"go-api/internal/application/event/dedup"
	"go-api/internal/application/registry"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/infrastructure/config"
	"go-api/internal/infrastructure/ffmpeg"
	"go-api/internal/infrastructure/messaging/rabbitmq"
	"go-api/internal/infrastructure/persistence/outbox"
	"go-api/internal/infrastructure/persistence/processed"
	"go-api/internal/infrastructure/persistence/write"
	"go-api/internal/infrastructure/storage"

	"gorm.io/gorm"
)

type Container struct {
	Consumer *rabbitmq.Consumer
	Conn     *rabbitmq.Connection
}

func NewContainer(db *gorm.DB, env *config.Config) *Container {
	topology := rabbitmq.DefaultTopology(
		env.RabbitMQExchange,
		env.FrameExtractionQueue,
		env.FrameExtractionRoutingKey,
		env.RabbitMQRetryTTLMS,
	)

	conn, err := rabbitmq.Connect(env.RabbitMQURL, topology)
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}

	minioStorage, err := storage.NewMinIOStorage(env)
	if err != nil {
		log.Fatalf("failed to init storage: %v", err)
	}

	mediaRepo := write.NewMediaWriteRepository(db)
	contentRepo := write.NewContentWriteRepository(db)
	outboxRepo := outbox.NewRepository(db)
	dedupRepo := processed.NewRepository(db)

	extractHandler := mediacmd.NewExtractFramesHandler(
		mediaRepo,
		contentRepo,
		outboxRepo,
		minioStorage,
		ffmpeg.NewExtractor(),
	)

	reg := registry.NewHandlerRegistry()
	reg.Register(domainmedia.EventTypeMediaUploaded, dedup.With(
		dedupRepo,
		"frame_extraction_on_uploaded",
		eventmedia.NewExtractOnUploadedHandler(extractHandler).Handle,
	))

	consumer := rabbitmq.NewConsumer(
		conn,
		reg,
		env.FrameExtractionConcurrency,
		env.WorkerMaxRetries,
	)

	return &Container{
		Consumer: consumer,
		Conn:     conn,
	}
}
