package di

import (
	"log"

	authcmd "go-api/internal/application/command/auth"
	campaigncmd "go-api/internal/application/command/campaign"
	clientcmd "go-api/internal/application/command/client"
	identitycmd "go-api/internal/application/command/identity"
	mediacmd "go-api/internal/application/command/media"
	"go-api/internal/application/command/mediaupload"
	usercmd "go-api/internal/application/command/user"
	queryactivity "go-api/internal/application/query/activity"
	querycampaign "go-api/internal/application/query/campaign"
	queryclient "go-api/internal/application/query/client"
	querymedia "go-api/internal/application/query/media"
	queryuser "go-api/internal/application/query/user"
	"go-api/internal/infrastructure/centrifugo"
	infraClerk "go-api/internal/infrastructure/clerk"
	"go-api/internal/infrastructure/config"
	"go-api/internal/infrastructure/imaging"
	"go-api/internal/infrastructure/persistence/outbox"
	"go-api/internal/infrastructure/persistence/read"
	"go-api/internal/infrastructure/persistence/write"
	"go-api/internal/infrastructure/storage"
	httphandler "go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/middleware"

	"gorm.io/gorm"
)

type Container struct {
	AuthenticateMiddleware       *middleware.AuthenticateMiddleware
	UserWebhookMiddleware        *middleware.UserWebhookMiddleware
	MediaUploadWebhookMiddleware *middleware.MediaUploadWebhookMiddleware
	UserWebhookHandler           *httphandler.UserWebhookHandler
	MediaUploadWebhookHandler    *httphandler.MediaUploadWebhookHandler
	UserHandler                  *httphandler.UserHandler
	ClientHandler                *httphandler.ClientHandler
	CampaignHandler              *httphandler.CampaignHandler
	MediaHandler                 *httphandler.MediaHandler
	ActivityHandler              *httphandler.ActivityHandler
	RealtimeHandler              *httphandler.RealtimeHandler
}

func NewContainer(db *gorm.DB, env *config.Config) *Container {
	jwksProvider, err := infraClerk.NewJWKSProvider(env)
	if err != nil {
		log.Fatalf("failed to create JWKS provider: %v", err)
	}

	objectStorage, err := storage.NewMinIOStorage(env)
	if err != nil {
		log.Fatalf("failed to create object storage: %v", err)
	}
	thumbnailer := imaging.NewThumbnailer()

	userWriteRepo := write.NewUserWriteRepository(db)
	userReadRepo := read.NewUserReadRepository(db)
	clientWriteRepo := write.NewClientWriteRepository(db)
	clientReadRepo := read.NewClientReadRepository(db)
	campaignWriteRepo := write.NewCampaignWriteRepository(db)
	campaignReadRepo := read.NewCampaignReadRepository(db)
	mediaWriteRepo := write.NewMediaWriteRepository(db)
	mediaReadRepo := read.NewMediaReadRepository(db)
	contentWriteRepo := write.NewContentWriteRepository(db)
	activityReadRepo := read.NewActivityEventReadRepository(db)
	outboxRepo := outbox.NewRepository(db)

	createUserHandler := usercmd.NewCreateUserHandler(
		userWriteRepo,
		clientWriteRepo,
		campaignWriteRepo,
		outboxRepo,
	)
	updateUserHandler := usercmd.NewUpdateUserHandler(userWriteRepo, outboxRepo)
	getUserByExternalIDHandler := usercmd.NewGetUserByExternalIDHandler(userWriteRepo)
	deleteUserByExternalIDHandler := usercmd.NewDeleteUserByExternalIDHandler(userWriteRepo, outboxRepo)
	setCurrentClientHandler := usercmd.NewSetCurrentClientHandler(userWriteRepo, clientWriteRepo, outboxRepo)
	validateTokenHandler := authcmd.NewValidateTokenHandler(jwksProvider, userWriteRepo)
	fetchUserHandler := identitycmd.NewFetchUserHandler(infraClerk.NewUserGateway(env.ClerkSecretKey))
	getUserByIDHandler := queryuser.NewGetUserByIDHandler(userReadRepo)

	createClientHandler := clientcmd.NewCreateClientHandler(
		clientWriteRepo,
		campaignWriteRepo,
		userWriteRepo,
		outboxRepo,
	)
	updateClientHandler := clientcmd.NewUpdateClientHandler(clientWriteRepo, outboxRepo)
	deleteClientHandler := clientcmd.NewDeleteClientHandler(clientWriteRepo, outboxRepo)
	removeClientMemberHandler := clientcmd.NewRemoveClientMemberHandler(clientWriteRepo, userWriteRepo, outboxRepo)
	getClientByIDHandler := queryclient.NewGetClientByIDHandler(clientReadRepo)
	listClientsByUserHandler := queryclient.NewListClientsByUserHandler(clientReadRepo)

	createCampaignHandler := campaigncmd.NewCreateCampaignHandler(campaignWriteRepo, outboxRepo)
	updateCampaignHandler := campaigncmd.NewUpdateCampaignHandler(campaignWriteRepo, outboxRepo)
	deleteCampaignHandler := campaigncmd.NewDeleteCampaignHandler(campaignWriteRepo, outboxRepo)
	presignBackgroundHandler := campaigncmd.NewPresignBackgroundHandler(
		campaignWriteRepo,
		outboxRepo,
		objectStorage,
	)
	clearBackgroundHandler := campaigncmd.NewClearBackgroundHandler(
		campaignWriteRepo,
		outboxRepo,
		objectStorage,
	)
	processBackgroundUploadHandler := campaigncmd.NewProcessBackgroundUploadHandler(
		campaignWriteRepo,
		outboxRepo,
		objectStorage,
		thumbnailer,
	)
	presignMediaHandler := mediacmd.NewPresignMediaHandler(
		campaignWriteRepo,
		mediaWriteRepo,
		outboxRepo,
		objectStorage,
	)
	processMediaUploadHandler := mediacmd.NewProcessUploadHandler(
		mediaWriteRepo,
		contentWriteRepo,
		outboxRepo,
		objectStorage,
		thumbnailer,
	)
	mediaObjectCreatedDispatcher := mediaupload.NewObjectCreatedDispatcher(
		processBackgroundUploadHandler,
		processMediaUploadHandler,
	)
	getCampaignByIDHandler := querycampaign.NewGetCampaignByIDHandler(campaignReadRepo)
	listCampaignsByClientHandler := querycampaign.NewListCampaignsByClientHandler(campaignReadRepo)
	listMediaByClientHandler := querymedia.NewListByClientHandler(mediaReadRepo, campaignReadRepo)
	getMediaByIDHandler := querymedia.NewGetByIDHandler(mediaReadRepo)
	getMediaStatsByClientHandler := querymedia.NewGetStatsByClientHandler(mediaReadRepo)
	listActivityByClientHandler := queryactivity.NewListByClientHandler(activityReadRepo)

	return &Container{
		AuthenticateMiddleware: middleware.NewAuthenticateMiddleware(
			validateTokenHandler,
			fetchUserHandler,
			createUserHandler,
		),
		UserWebhookMiddleware: middleware.NewUserWebhookMiddleware(env.ClerkWebhookSecret),
		MediaUploadWebhookMiddleware: middleware.NewMediaUploadWebhookMiddleware(
			env.MinIOWebhookSecret,
		),
		UserWebhookHandler: httphandler.NewUserWebhookHandler(
			getUserByExternalIDHandler,
			createUserHandler,
			updateUserHandler,
			deleteUserByExternalIDHandler,
		),
		MediaUploadWebhookHandler: httphandler.NewMediaUploadWebhookHandler(
			env.StorageBucket,
			mediaObjectCreatedDispatcher,
		),
		UserHandler: httphandler.NewUserHandler(getUserByIDHandler, setCurrentClientHandler),
		ClientHandler: httphandler.NewClientHandler(
			createClientHandler,
			updateClientHandler,
			deleteClientHandler,
			removeClientMemberHandler,
			getClientByIDHandler,
			listClientsByUserHandler,
		),
		CampaignHandler: httphandler.NewCampaignHandler(
			createCampaignHandler,
			updateCampaignHandler,
			deleteCampaignHandler,
			getCampaignByIDHandler,
			listCampaignsByClientHandler,
			getClientByIDHandler,
			presignBackgroundHandler,
			clearBackgroundHandler,
			objectStorage,
		),
		MediaHandler: httphandler.NewMediaHandler(
			presignMediaHandler,
			listMediaByClientHandler,
			getMediaByIDHandler,
			getMediaStatsByClientHandler,
			objectStorage,
		),
		ActivityHandler: httphandler.NewActivityHandler(listActivityByClientHandler),
		RealtimeHandler: httphandler.NewRealtimeHandler(centrifugo.NewConnectionInfoCreator(env)),
	}
}
