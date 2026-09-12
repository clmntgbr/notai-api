package di

import (
	"log"

	eventactivity "go-api/internal/application/event/activity"
	eventcampaign "go-api/internal/application/event/campaign"
	eventclient "go-api/internal/application/event/client"
	eventcontent "go-api/internal/application/event/content"
	"go-api/internal/application/event/dedup"
	eventuser "go-api/internal/application/event/user"
	"go-api/internal/application/registry"
	domaincampaign "go-api/internal/domain/campaign"
	domainclient "go-api/internal/domain/client"
	domaincontent "go-api/internal/domain/content"
	domainuser "go-api/internal/domain/user"
	"go-api/internal/infrastructure/centrifugo"
	"go-api/internal/infrastructure/config"
	"go-api/internal/infrastructure/messaging/rabbitmq"
	"go-api/internal/infrastructure/notification"
	"go-api/internal/infrastructure/persistence/outbox"
	"go-api/internal/infrastructure/persistence/processed"
	"go-api/internal/infrastructure/persistence/read"
	"go-api/internal/infrastructure/persistence/write"

	"gorm.io/gorm"
)

type Container struct {
	Relay    *outbox.Relay
	Consumer *rabbitmq.Consumer
	Conn     *rabbitmq.Connection
}

func NewContainer(db *gorm.DB, env *config.Config) *Container {
	topology := rabbitmq.DefaultTopology(
		env.RabbitMQExchange,
		env.RabbitMQQueue,
		env.RabbitMQRoutingKey,
		env.RabbitMQRetryTTLMS,
	)

	conn, err := rabbitmq.Connect(env.RabbitMQURL, topology)
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}

	publisher := rabbitmq.NewPublisher(conn, env.RabbitMQExchange)
	outboxRepo := outbox.NewRepository(db)
	relay := outbox.NewRelay(outboxRepo, publisher, env.OutboxPollInterval, 50)

	dedupRepo := processed.NewRepository(db)
	notifier := notification.NewLogNotifier()
	realtimePublisher := centrifugo.NewPublisher(env)
	clientReadRepo := read.NewClientReadRepository(db)
	userReadRepo := read.NewUserReadRepository(db)
	contentReadRepo := read.NewContentReadRepository(db)
	activityWriteRepo := write.NewActivityEventWriteRepository(db)
	activityProjector := eventactivity.NewProjector(
		activityWriteRepo,
		contentReadRepo,
		userReadRepo,
		clientReadRepo,
		realtimePublisher,
	)
	publishUserRealtime := eventuser.NewPublishRealtimeHandler(realtimePublisher)
	publishClientRealtime := eventclient.NewPublishRealtimeHandler(realtimePublisher)
	publishCampaignRealtime := eventcampaign.NewPublishRealtimeHandler(realtimePublisher, clientReadRepo)
	publishContentRealtime := eventcontent.NewPublishRealtimeHandler(realtimePublisher, clientReadRepo)
	reg := registry.NewHandlerRegistry()

	reg.Register(domainuser.EventTypeUserCreated, dedup.With(
		dedupRepo,
		"user_created",
		eventuser.NewUserCreatedHandler().Handle,
	))
	reg.Register(domainuser.EventTypeUserCreated, dedup.With(
		dedupRepo,
		"notify_user_on_created",
		eventuser.NewNotifyUserOnCreatedHandler(notifier).Handle,
	))
	reg.Register(domainuser.EventTypeUserCreated, dedup.With(
		dedupRepo,
		"publish_user_created_realtime",
		publishUserRealtime.OnCreated,
	))
	reg.Register(domainuser.EventTypeUserUpdated, dedup.With(
		dedupRepo,
		"user_updated",
		eventuser.NewUserUpdatedHandler().Handle,
	))
	reg.Register(domainuser.EventTypeUserUpdated, dedup.With(
		dedupRepo,
		"publish_user_updated_realtime",
		publishUserRealtime.OnUpdated,
	))
	reg.Register(domainuser.EventTypeUserDeleted, dedup.With(
		dedupRepo,
		"user_deleted",
		eventuser.NewUserDeletedHandler().Handle,
	))
	reg.Register(domainuser.EventTypeUserDeleted, dedup.With(
		dedupRepo,
		"publish_user_deleted_realtime",
		publishUserRealtime.OnDeleted,
	))
	reg.Register(domainuser.EventTypeUserCurrentClientChanged, dedup.With(
		dedupRepo,
		"user_current_client_changed",
		eventuser.NewUserCurrentClientChangedHandler().Handle,
	))
	reg.Register(domainuser.EventTypeUserCurrentClientChanged, dedup.With(
		dedupRepo,
		"publish_user_current_client_changed_realtime",
		publishUserRealtime.OnCurrentClientChanged,
	))

	reg.Register(domainclient.EventTypeClientCreated, dedup.With(
		dedupRepo,
		"client_created",
		eventclient.NewClientCreatedHandler().Handle,
	))
	reg.Register(domainclient.EventTypeClientCreated, dedup.With(
		dedupRepo,
		"publish_client_created_realtime",
		publishClientRealtime.OnCreated,
	))
	reg.Register(domainclient.EventTypeClientUpdated, dedup.With(
		dedupRepo,
		"client_updated",
		eventclient.NewClientUpdatedHandler().Handle,
	))
	reg.Register(domainclient.EventTypeClientUpdated, dedup.With(
		dedupRepo,
		"publish_client_updated_realtime",
		publishClientRealtime.OnUpdated,
	))
	reg.Register(domainclient.EventTypeClientDeleted, dedup.With(
		dedupRepo,
		"client_deleted",
		eventclient.NewClientDeletedHandler().Handle,
	))
	reg.Register(domainclient.EventTypeClientDeleted, dedup.With(
		dedupRepo,
		"publish_client_deleted_realtime",
		publishClientRealtime.OnDeleted,
	))
	reg.Register(domainclient.EventTypeClientMemberAdded, dedup.With(
		dedupRepo,
		"client_member_added",
		eventclient.NewClientMemberAddedHandler().Handle,
	))
	reg.Register(domainclient.EventTypeClientMemberAdded, dedup.With(
		dedupRepo,
		"publish_client_member_added_realtime",
		publishClientRealtime.OnMemberAdded,
	))
	reg.Register(domainclient.EventTypeClientMemberAdded, dedup.With(
		dedupRepo,
		"project_activity_on_client_member_added",
		activityProjector.OnClientMemberAdded,
	))
	reg.Register(domainclient.EventTypeClientMemberRemoved, dedup.With(
		dedupRepo,
		"client_member_removed",
		eventclient.NewClientMemberRemovedHandler().Handle,
	))
	reg.Register(domainclient.EventTypeClientMemberRemoved, dedup.With(
		dedupRepo,
		"publish_client_member_removed_realtime",
		publishClientRealtime.OnMemberRemoved,
	))

	reg.Register(domaincampaign.EventTypeCampaignCreated, dedup.With(
		dedupRepo,
		"campaign_created",
		eventcampaign.NewCampaignCreatedHandler().Handle,
	))
	reg.Register(domaincampaign.EventTypeCampaignCreated, dedup.With(
		dedupRepo,
		"publish_campaign_created_realtime",
		publishCampaignRealtime.OnCreated,
	))
	reg.Register(domaincampaign.EventTypeCampaignCreated, dedup.With(
		dedupRepo,
		"project_activity_on_campaign_created",
		activityProjector.OnCampaignCreated,
	))
	reg.Register(domaincampaign.EventTypeCampaignUpdated, dedup.With(
		dedupRepo,
		"campaign_updated",
		eventcampaign.NewCampaignUpdatedHandler().Handle,
	))
	reg.Register(domaincampaign.EventTypeCampaignUpdated, dedup.With(
		dedupRepo,
		"publish_campaign_updated_realtime",
		publishCampaignRealtime.OnUpdated,
	))
	reg.Register(domaincampaign.EventTypeCampaignDeleted, dedup.With(
		dedupRepo,
		"campaign_deleted",
		eventcampaign.NewCampaignDeletedHandler().Handle,
	))
	reg.Register(domaincampaign.EventTypeCampaignDeleted, dedup.With(
		dedupRepo,
		"publish_campaign_deleted_realtime",
		publishCampaignRealtime.OnDeleted,
	))
	reg.Register(domaincampaign.EventTypeCampaignBackgroundUpdated, dedup.With(
		dedupRepo,
		"campaign_background_updated",
		eventcampaign.NewCampaignBackgroundUpdatedHandler().Handle,
	))
	reg.Register(domaincampaign.EventTypeCampaignBackgroundUpdated, dedup.With(
		dedupRepo,
		"publish_campaign_background_updated_realtime",
		publishCampaignRealtime.OnBackgroundUpdated,
	))

	reg.Register(domaincontent.EventTypeContentCreated, dedup.With(
		dedupRepo,
		"content_created",
		eventcontent.NewContentCreatedHandler().Handle,
	))
	reg.Register(domaincontent.EventTypeContentCreated, dedup.With(
		dedupRepo,
		"publish_content_created_realtime",
		publishContentRealtime.OnCreated,
	))
	reg.Register(domaincontent.EventTypeContentUploaded, dedup.With(
		dedupRepo,
		"content_uploaded",
		eventcontent.NewContentUploadedHandler().Handle,
	))
	reg.Register(domaincontent.EventTypeContentUploaded, dedup.With(
		dedupRepo,
		"publish_content_uploaded_realtime",
		publishContentRealtime.OnUploaded,
	))
	reg.Register(domaincontent.EventTypeContentStatusChanged, dedup.With(
		dedupRepo,
		"content_status_changed",
		eventcontent.NewContentStatusChangedHandler().Handle,
	))
	reg.Register(domaincontent.EventTypeContentStatusChanged, dedup.With(
		dedupRepo,
		"publish_content_status_changed_realtime",
		publishContentRealtime.OnStatusChanged,
	))
	reg.Register(domaincontent.EventTypeContentStatusChanged, dedup.With(
		dedupRepo,
		"project_activity_on_content_status_changed",
		activityProjector.OnContentStatusChanged,
	))
	reg.Register(domaincontent.EventTypeContentVerdictRendered, dedup.With(
		dedupRepo,
		"content_verdict_rendered",
		eventcontent.NewContentVerdictRenderedHandler().Handle,
	))
	reg.Register(domaincontent.EventTypeContentVerdictRendered, dedup.With(
		dedupRepo,
		"publish_content_verdict_rendered_realtime",
		publishContentRealtime.OnVerdictRendered,
	))
	reg.Register(domaincontent.EventTypeContentVerdictRendered, dedup.With(
		dedupRepo,
		"project_activity_on_content_verdict_rendered",
		activityProjector.OnContentVerdictRendered,
	))

	consumer := rabbitmq.NewConsumer(conn, reg, env.WorkerConcurrency, env.WorkerMaxRetries)

	return &Container{
		Relay:    relay,
		Consumer: consumer,
		Conn:     conn,
	}
}
