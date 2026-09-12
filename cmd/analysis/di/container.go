package di

import (
	"log"

	contentcmd "go-api/internal/application/command/content"
	eventcontent "go-api/internal/application/event/content"
	"go-api/internal/application/event/dedup"
	"go-api/internal/application/registry"
	domaincontent "go-api/internal/domain/content"
	"go-api/internal/infrastructure/analysis"
	"go-api/internal/infrastructure/config"
	"go-api/internal/infrastructure/messaging/rabbitmq"
	"go-api/internal/infrastructure/persistence/outbox"
	"go-api/internal/infrastructure/persistence/processed"
	"go-api/internal/infrastructure/persistence/write"
	"go-api/internal/infrastructure/sightengine"
	"go-api/internal/infrastructure/storage"

	"gorm.io/gorm"
)

type Container struct {
	Consumer *rabbitmq.Consumer
	Conn     *rabbitmq.Connection
}

func NewContainer(db *gorm.DB, env *config.Config) *Container {
	sightengine.CleanupTempResiduals()

	topology := rabbitmq.DefaultTopology(
		env.RabbitMQExchange,
		env.AnalysisQueue,
		env.AnalysisRoutingKey,
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

	contentRepo := write.NewContentWriteRepository(db)
	resultRepo := write.NewContentAnalysisResultRepository(db)
	outboxRepo := outbox.NewRepository(db)
	dedupRepo := processed.NewRepository(db)

	detectors := analysis.BuildDetectors(env, minioStorage)
	analyzeHandler := contentcmd.NewAnalyzeContentHandler(
		contentRepo,
		resultRepo,
		outboxRepo,
		detectors,
		env.AnalysisMaxDetectors,
	)

	reg := registry.NewHandlerRegistry()
	reg.Register(domaincontent.EventTypeContentUploaded, dedup.With(
		dedupRepo,
		"analysis_on_uploaded",
		eventcontent.NewAnalyzeOnUploadedHandler(analyzeHandler).Handle,
	))

	consumer := rabbitmq.NewConsumer(
		conn,
		reg,
		env.AnalysisConcurrency,
		env.WorkerMaxRetries,
	)

	return &Container{
		Consumer: consumer,
		Conn:     conn,
	}
}
