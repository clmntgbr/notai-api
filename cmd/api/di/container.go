package di

import (
	"log"

	authcmd "go-api/internal/application/command/auth"
	campaigncmd "go-api/internal/application/command/campaign"
	clientcmd "go-api/internal/application/command/client"
	identitycmd "go-api/internal/application/command/identity"
	mediacmd "go-api/internal/application/command/media"
	"go-api/internal/application/command/mediaupload"
	cmdquota "go-api/internal/application/command/quota"
	subscriptioncmd "go-api/internal/application/command/subscription"
	usercmd "go-api/internal/application/command/user"
	queryactivity "go-api/internal/application/query/activity"
	querycampaign "go-api/internal/application/query/campaign"
	queryclient "go-api/internal/application/query/client"
	queryinvoice "go-api/internal/application/query/invoice"
	querymedia "go-api/internal/application/query/media"
	queryplan "go-api/internal/application/query/plan"
	querysubscription "go-api/internal/application/query/subscription"
	queryuser "go-api/internal/application/query/user"
	"go-api/internal/infrastructure/centrifugo"
	infraClerk "go-api/internal/infrastructure/clerk"
	"go-api/internal/infrastructure/config"
	"go-api/internal/infrastructure/imaging"
	"go-api/internal/infrastructure/persistence/outbox"
	"go-api/internal/infrastructure/persistence/read"
	"go-api/internal/infrastructure/persistence/write"
	infrastripe "go-api/internal/infrastructure/stripe"
	"go-api/internal/infrastructure/storage"
	httphandler "go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/middleware"

	"gorm.io/gorm"
)

type Container struct {
	AuthenticateMiddleware       *middleware.AuthenticateMiddleware
	UserWebhookMiddleware        *middleware.UserWebhookMiddleware
	MediaUploadWebhookMiddleware *middleware.MediaUploadWebhookMiddleware
	BillingWebhookMiddleware     *middleware.BillingWebhookMiddleware
	UserWebhookHandler           *httphandler.UserWebhookHandler
	MediaUploadWebhookHandler    *httphandler.MediaUploadWebhookHandler
	BillingWebhookHandler        *httphandler.BillingWebhookHandler
	UserHandler                  *httphandler.UserHandler
	ClientHandler                *httphandler.ClientHandler
	CampaignHandler              *httphandler.CampaignHandler
	MediaHandler                 *httphandler.MediaHandler
	ActivityHandler              *httphandler.ActivityHandler
	RealtimeHandler              *httphandler.RealtimeHandler
	PlanHandler                  *httphandler.PlanHandler
	SubscriptionHandler          *httphandler.SubscriptionHandler
	InvoiceHandler               *httphandler.InvoiceHandler
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
	workspaceWriteRepo := write.NewWorkspaceWriteRepository(db)
	workspaceReadRepo := read.NewWorkspaceReadRepository(db)
	clientWriteRepo := write.NewClientWriteRepository(db)
	clientReadRepo := read.NewClientReadRepository(db)
	campaignWriteRepo := write.NewCampaignWriteRepository(db)
	campaignReadRepo := read.NewCampaignReadRepository(db)
	mediaWriteRepo := write.NewMediaWriteRepository(db)
	mediaReadRepo := read.NewMediaReadRepository(db)
	contentWriteRepo := write.NewContentWriteRepository(db)
	contentReadRepo := read.NewContentReadRepository(db)
	analysisResultRepo := write.NewContentAnalysisResultRepository(db)
	activityReadRepo := read.NewActivityEventReadRepository(db)
	outboxRepo := outbox.NewRepository(db)

	quotaReadRepo := read.NewQuotaReadRepository(db)
	planWriteRepo := write.NewPlanWriteRepository(db)
	planReadRepo := read.NewPlanReadRepository(db, quotaReadRepo)
	subscriptionWriteRepo := write.NewSubscriptionWriteRepository(db)
	subscriptionReadRepo := read.NewSubscriptionReadRepository(db, planReadRepo)
	invoiceWriteRepo := write.NewInvoiceWriteRepository(db)
	invoiceReadRepo := read.NewInvoiceReadRepository(db)

	checkoutSessionGateway := infrastripe.NewCheckoutSessionGateway(env)
	billingPortalGateway := infrastripe.NewBillingPortalGateway(env)
	subscriptionGateway := infrastripe.NewSubscriptionGateway(env)

	createUserHandler := usercmd.NewCreateUserHandler(
		userWriteRepo,
		workspaceWriteRepo,
		clientWriteRepo,
		campaignWriteRepo,
		planWriteRepo,
		subscriptionWriteRepo,
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
		workspaceWriteRepo,
		campaignWriteRepo,
		userWriteRepo,
		outboxRepo,
	)
	updateClientHandler := clientcmd.NewUpdateClientHandler(clientWriteRepo, outboxRepo)
	deleteClientHandler := clientcmd.NewDeleteClientHandler(clientWriteRepo, outboxRepo)
	removeClientMemberHandler := clientcmd.NewRemoveClientMemberHandler(clientWriteRepo, userWriteRepo, outboxRepo)
	getClientByIDHandler := queryclient.NewGetClientByIDHandler(clientReadRepo)
	listClientsByUserHandler := queryclient.NewListClientsByUserHandler(clientReadRepo)

	getQuotaUsageHandler := querysubscription.NewGetQuotaUsageHandler(
		clientReadRepo,
		workspaceReadRepo,
		subscriptionReadRepo,
		planReadRepo,
		campaignReadRepo,
		contentReadRepo,
	)
	assertCreateAllowedHandler := cmdquota.NewAssertCreateAllowedHandler(
		getQuotaUsageHandler,
		write.NewAnalysisQuotaLocker(db),
	)
	addClientMemberHandler := clientcmd.NewAddClientMemberHandler(
		clientWriteRepo,
		userWriteRepo,
		assertCreateAllowedHandler,
		outboxRepo,
	)

	createCampaignHandler := campaigncmd.NewCreateCampaignHandler(
		campaignWriteRepo,
		outboxRepo,
		assertCreateAllowedHandler,
	)
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
		assertCreateAllowedHandler,
	)
	processMediaUploadHandler := mediacmd.NewProcessUploadHandler(
		mediaWriteRepo,
		contentWriteRepo,
		outboxRepo,
		objectStorage,
		thumbnailer,
		assertCreateAllowedHandler,
	)
	mediaObjectCreatedDispatcher := mediaupload.NewObjectCreatedDispatcher(
		processBackgroundUploadHandler,
		processMediaUploadHandler,
	)
	getCampaignByIDHandler := querycampaign.NewGetCampaignByIDHandler(campaignReadRepo)
	listCampaignsByClientHandler := querycampaign.NewListCampaignsByClientHandler(campaignReadRepo)
	listMediaByClientHandler := querymedia.NewListByClientHandler(mediaReadRepo, campaignReadRepo)
	getMediaByIDHandler := querymedia.NewGetByIDHandler(
		mediaReadRepo,
		campaignReadRepo,
		analysisResultRepo,
	)
	getMediaStatsByClientHandler := querymedia.NewGetStatsByClientHandler(mediaReadRepo)
	listActivityByClientHandler := queryactivity.NewListByClientHandler(activityReadRepo)

	listActivePlansHandler := queryplan.NewListActivePlansHandler(planReadRepo)
	getCurrentSubscriptionHandler := querysubscription.NewGetCurrentSubscriptionHandler(
		workspaceReadRepo,
		subscriptionReadRepo,
	)
	previewPlanChangeHandler := querysubscription.NewPreviewPlanChangeHandler(
		workspaceReadRepo,
		planReadRepo,
		subscriptionReadRepo,
		subscriptionGateway,
	)
	createSubscriptionHandler := subscriptioncmd.NewCreateSubscriptionHandler(
		workspaceReadRepo,
		userReadRepo,
		planReadRepo,
		subscriptionReadRepo,
		subscriptionWriteRepo,
		outboxRepo,
		fetchUserHandler,
		checkoutSessionGateway,
		subscriptionGateway,
	)
	createBillingPortalHandler := subscriptioncmd.NewCreateBillingPortalHandler(
		workspaceReadRepo,
		subscriptionReadRepo,
		billingPortalGateway,
	)
	listInvoicesHandler := queryinvoice.NewListInvoicesHandler(invoiceReadRepo)

	upsertInvoiceHandler := subscriptioncmd.NewUpsertInvoiceHandler(
		invoiceWriteRepo,
		subscriptionWriteRepo,
		workspaceWriteRepo,
		outboxRepo,
	)
	checkoutCompletedHandler := subscriptioncmd.NewCheckoutCompletedHandler(
		workspaceWriteRepo,
		planWriteRepo,
		subscriptionWriteRepo,
		outboxRepo,
		subscriptionGateway,
		upsertInvoiceHandler,
	)
	subscriptionUpdatedHandler := subscriptioncmd.NewSubscriptionUpdatedHandler(
		planWriteRepo,
		subscriptionWriteRepo,
		outboxRepo,
	)
	subscriptionDeletedHandler := subscriptioncmd.NewSubscriptionDeletedHandler(
		planWriteRepo,
		subscriptionWriteRepo,
		outboxRepo,
	)
	invoicePaymentSucceededHandler := subscriptioncmd.NewInvoicePaymentSucceededHandler(
		subscriptionWriteRepo,
		outboxRepo,
	)
	invoicePaymentFailedHandler := subscriptioncmd.NewInvoicePaymentFailedHandler(
		subscriptionWriteRepo,
		outboxRepo,
	)
	renewalUpcomingHandler := subscriptioncmd.NewSubscriptionRenewalUpcomingHandler(
		subscriptionWriteRepo,
		outboxRepo,
	)
	paymentMethodExpiringHandler := subscriptioncmd.NewPaymentMethodExpiringHandler(
		subscriptionWriteRepo,
		outboxRepo,
	)

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
		BillingWebhookMiddleware: middleware.NewBillingWebhookMiddleware(env.StripeWebhookSecret),
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
		BillingWebhookHandler: httphandler.NewBillingWebhookHandler(
			checkoutCompletedHandler,
			subscriptionUpdatedHandler,
			subscriptionDeletedHandler,
			invoicePaymentSucceededHandler,
			invoicePaymentFailedHandler,
			upsertInvoiceHandler,
			renewalUpcomingHandler,
			paymentMethodExpiringHandler,
		),
		UserHandler: httphandler.NewUserHandler(getUserByIDHandler, setCurrentClientHandler),
		ClientHandler: httphandler.NewClientHandler(
			createClientHandler,
			updateClientHandler,
			deleteClientHandler,
			addClientMemberHandler,
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
		PlanHandler: httphandler.NewPlanHandler(listActivePlansHandler),
		SubscriptionHandler: httphandler.NewSubscriptionHandler(
			getClientByIDHandler,
			getCurrentSubscriptionHandler,
			getQuotaUsageHandler,
			previewPlanChangeHandler,
			createSubscriptionHandler,
			createBillingPortalHandler,
		),
		InvoiceHandler: httphandler.NewInvoiceHandler(getClientByIDHandler, listInvoicesHandler),
	}
}
