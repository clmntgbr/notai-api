package di

import (
	"log"

	mediacmd "go-api/internal/application/command/media"
	cmdquota "go-api/internal/application/command/quota"
	"go-api/internal/application/event/dedup"
	eventmedia "go-api/internal/application/event/media"
	querysubscription "go-api/internal/application/query/subscription"
	"go-api/internal/application/registry"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/infrastructure/config"
	"go-api/internal/infrastructure/ffmpeg"
	"go-api/internal/infrastructure/messaging/rabbitmq"
	"go-api/internal/infrastructure/persistence/outbox"
	"go-api/internal/infrastructure/persistence/processed"
	"go-api/internal/infrastructure/persistence/read"
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

	getQuotaUsageHandler := querysubscription.NewGetQuotaUsageHandler(
		read.NewClientReadRepository(db),
		read.NewWorkspaceReadRepository(db),
		read.NewSubscriptionReadRepository(db, read.NewPlanReadRepository(db, read.NewQuotaReadRepository(db))),
		read.NewPlanReadRepository(db, read.NewQuotaReadRepository(db)),
		read.NewCampaignReadRepository(db),
		read.NewContentReadRepository(db),
		read.NewMediaReadRepository(db),
	)
	assertCreateAllowedHandler := cmdquota.NewAssertCreateAllowedHandler(
		getQuotaUsageHandler,
		write.NewAnalysisQuotaLocker(db),
	)

	extractHandler := mediacmd.NewExtractFramesHandler(
		mediaRepo,
		contentRepo,
		outboxRepo,
		minioStorage,
		ffmpeg.NewExtractor(),
		assertCreateAllowedHandler,
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
