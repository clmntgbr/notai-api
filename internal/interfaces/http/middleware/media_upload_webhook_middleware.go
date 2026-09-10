package middleware

import (
	"github.com/gofiber/fiber/v3"
)

type MediaUploadWebhookMiddleware struct {
	secret string
}

func NewMediaUploadWebhookMiddleware(secret string) *MediaUploadWebhookMiddleware {
	return &MediaUploadWebhookMiddleware{secret: secret}
}

func (m *MediaUploadWebhookMiddleware) Protected() fiber.Handler {
	return func(c fiber.Ctx) error {
		if m.secret == "" {
			return c.Next()
		}

		authHeader := c.Get("Authorization")
		expected := "Bearer " + m.secret
		if authHeader != expected && authHeader != m.secret {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Unauthorized",
			})
		}

		return c.Next()
	}
}
