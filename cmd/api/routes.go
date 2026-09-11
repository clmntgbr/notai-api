package main

import (
	"go-api/cmd/api/di"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
)

func setupRoutes(app *fiber.App, container *di.Container) {
	setupHealthChecks(app)
	setupWebhooks(app, container)
	setupAPIRoutes(app, container)
}

func setupWebhooks(app *fiber.App, container *di.Container) {
	webhooks := app.Group("/webhooks")
	webhooks.Post("/clerk", container.UserWebhookMiddleware.Protected(), container.UserWebhookHandler.Execute)
	webhooks.Post(
		"/minio/object-created",
		container.MediaUploadWebhookMiddleware.Protected(),
		container.MediaUploadWebhookHandler.ObjectCreated,
	)
}

func setupHealthChecks(app *fiber.App) {
	app.Get(healthcheck.LivenessEndpoint, healthcheck.New())
	app.Get(healthcheck.ReadinessEndpoint, healthcheck.New())
	app.Get(healthcheck.StartupEndpoint, healthcheck.New())
}

func setupAPIRoutes(app *fiber.App, container *di.Container) {
	public := app.Group("/api")

	protected := public.Group("", container.AuthenticateMiddleware.Protected())
	setupUserRoutes(protected, container)
	setupClientRoutes(protected, container)
	setupCampaignRoutes(protected, container)
	setupContentRoutes(protected, container)
	setupRealtimeRoutes(protected, container)
}

func setupRealtimeRoutes(api fiber.Router, container *di.Container) {
	api.Get("/realtime/connection", container.RealtimeHandler.GetConnection)
}

func setupUserRoutes(api fiber.Router, container *di.Container) {
	api.Get("/users/me", container.UserHandler.GetUser)
	api.Put("/users/me/current-client", container.UserHandler.SetCurrentClient)
}

func setupClientRoutes(api fiber.Router, container *di.Container) {
	api.Get("/clients", container.ClientHandler.List)
	api.Post("/clients", container.ClientHandler.Create)
	api.Get("/clients/:id", container.ClientHandler.GetByID)
	api.Put("/clients/:id", container.ClientHandler.Update)
	api.Delete("/clients/:id", container.ClientHandler.Delete)
	api.Delete("/clients/:id/members/:userId", container.ClientHandler.RemoveMember)
}

func setupCampaignRoutes(api fiber.Router, container *di.Container) {
	api.Get("/campaigns", container.CampaignHandler.List)
	api.Post("/campaigns", container.CampaignHandler.Create)
	api.Get("/campaigns/:id", container.CampaignHandler.GetByID)
	api.Put("/campaigns/:id", container.CampaignHandler.Update)
	api.Delete("/campaigns/:id", container.CampaignHandler.Delete)
	api.Post("/campaigns/:id/background/presign", container.CampaignHandler.PresignBackground)
	api.Delete("/campaigns/:id/background", container.CampaignHandler.ClearBackground)
	api.Get("/campaigns/:id/thumbnail", container.CampaignHandler.GetThumbnail)
}

func setupContentRoutes(api fiber.Router, container *di.Container) {
	api.Get("/contents", container.ContentHandler.List)
	api.Post("/contents/presign", container.ContentHandler.Presign)
	api.Get("/contents/:id", container.ContentHandler.GetByID)
	api.Get("/contents/:id/thumbnail", container.ContentHandler.GetThumbnail)
}
