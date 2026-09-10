package di

import (
	"log"

	authcmd "go-api/internal/application/command/auth"
	campaigncmd "go-api/internal/application/command/campaign"
	clientcmd "go-api/internal/application/command/client"
	identitycmd "go-api/internal/application/command/identity"
	usercmd "go-api/internal/application/command/user"
	querycampaign "go-api/internal/application/query/campaign"
	queryclient "go-api/internal/application/query/client"
	queryuser "go-api/internal/application/query/user"
	"go-api/internal/infrastructure/centrifugo"
	infraClerk "go-api/internal/infrastructure/clerk"
	"go-api/internal/infrastructure/config"
	"go-api/internal/infrastructure/persistence/outbox"
	"go-api/internal/infrastructure/persistence/read"
	"go-api/internal/infrastructure/persistence/write"
	httphandler "go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/middleware"

	"gorm.io/gorm"
)

type Container struct {
	AuthenticateMiddleware *middleware.AuthenticateMiddleware
	UserWebhookMiddleware  *middleware.UserWebhookMiddleware
	UserWebhookHandler     *httphandler.UserWebhookHandler
	UserHandler            *httphandler.UserHandler
	ClientHandler          *httphandler.ClientHandler
	CampaignHandler        *httphandler.CampaignHandler
	RealtimeHandler        *httphandler.RealtimeHandler
}

func NewContainer(db *gorm.DB, env *config.Config) *Container {
	jwksProvider, err := infraClerk.NewJWKSProvider(env)
	if err != nil {
		log.Fatalf("failed to create JWKS provider: %v", err)
	}

	userWriteRepo := write.NewUserWriteRepository(db)
	userReadRepo := read.NewUserReadRepository(db)
	clientWriteRepo := write.NewClientWriteRepository(db)
	clientReadRepo := read.NewClientReadRepository(db)
	campaignWriteRepo := write.NewCampaignWriteRepository(db)
	campaignReadRepo := read.NewCampaignReadRepository(db)
	outboxRepo := outbox.NewRepository(db)

	createUserHandler := usercmd.NewCreateUserHandler(userWriteRepo, clientWriteRepo, outboxRepo)
	updateUserHandler := usercmd.NewUpdateUserHandler(userWriteRepo, outboxRepo)
	getUserByExternalIDHandler := usercmd.NewGetUserByExternalIDHandler(userWriteRepo)
	deleteUserByExternalIDHandler := usercmd.NewDeleteUserByExternalIDHandler(userWriteRepo, outboxRepo)
	setCurrentClientHandler := usercmd.NewSetCurrentClientHandler(userWriteRepo, clientWriteRepo, outboxRepo)
	validateTokenHandler := authcmd.NewValidateTokenHandler(jwksProvider, userWriteRepo)
	fetchUserHandler := identitycmd.NewFetchUserHandler(infraClerk.NewUserGateway(env.ClerkSecretKey))
	getUserByIDHandler := queryuser.NewGetUserByIDHandler(userReadRepo)

	createClientHandler := clientcmd.NewCreateClientHandler(clientWriteRepo, userWriteRepo, outboxRepo)
	updateClientHandler := clientcmd.NewUpdateClientHandler(clientWriteRepo, outboxRepo)
	deleteClientHandler := clientcmd.NewDeleteClientHandler(clientWriteRepo, outboxRepo)
	removeClientMemberHandler := clientcmd.NewRemoveClientMemberHandler(clientWriteRepo, userWriteRepo, outboxRepo)
	getClientByIDHandler := queryclient.NewGetClientByIDHandler(clientReadRepo)
	listClientsByUserHandler := queryclient.NewListClientsByUserHandler(clientReadRepo)

	createCampaignHandler := campaigncmd.NewCreateCampaignHandler(campaignWriteRepo, outboxRepo)
	updateCampaignHandler := campaigncmd.NewUpdateCampaignHandler(campaignWriteRepo, outboxRepo)
	deleteCampaignHandler := campaigncmd.NewDeleteCampaignHandler(campaignWriteRepo, outboxRepo)
	getCampaignByIDHandler := querycampaign.NewGetCampaignByIDHandler(campaignReadRepo)
	listCampaignsByClientHandler := querycampaign.NewListCampaignsByClientHandler(campaignReadRepo)

	return &Container{
		AuthenticateMiddleware: middleware.NewAuthenticateMiddleware(
			validateTokenHandler,
			fetchUserHandler,
			createUserHandler,
		),
		UserWebhookMiddleware: middleware.NewUserWebhookMiddleware(env.ClerkWebhookSecret),
		UserWebhookHandler: httphandler.NewUserWebhookHandler(
			getUserByExternalIDHandler,
			createUserHandler,
			updateUserHandler,
			deleteUserByExternalIDHandler,
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
		),
		RealtimeHandler: httphandler.NewRealtimeHandler(centrifugo.NewConnectionInfoCreator(env)),
	}
}
