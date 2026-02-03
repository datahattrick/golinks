package api

import (
	"github.com/gofiber/fiber/v3"
)

// jsonSuccess returns a 200 response with data wrapped in the standard envelope.
func jsonSuccess(c fiber.Ctx, data any) error {
	return c.JSON(fiber.Map{
		"status": "ok",
		"data":   data,
	})
}

// jsonError returns an error response with the given HTTP status code.
func jsonError(c fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{
		"status": "error",
		"error":  message,
	})
}
