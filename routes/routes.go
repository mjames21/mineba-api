// path: routes/routes.go
package routes

import (
	"meniba/controllers"

	"github.com/gofiber/fiber/v2"
)

// Register attaches all API endpoints to the app.
func Register(app *fiber.App) {
	api := app.Group("/api")

	api.Post("/locate", controllers.HandleLocate)

	api.Post("/reports", controllers.HandlePostReport)
	api.Get("/reports", controllers.HandleListReports)

	// Optional: quick echo to verify requests reach the API
	api.Get("/debug/echo", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"method": c.Method(),
			"ct":     c.Get("Content-Type"),
			"len":    len(c.Body()),
		})
	})
}
