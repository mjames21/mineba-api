// path: main.go
package main

import (
	"context"
	"log"
	"time"

	"meniba/database"
	"meniba/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	if err := database.Connect(context.Background()); err != nil {
		log.Fatalf("db connect failed: %v", err)
	}

	app := fiber.New()
	app.Use(recover.New())

	// Log concise request lines
	app.Use(logger.New(logger.Config{
		TimeFormat: "15:04:05",
	}))

	// CORS (dev-friendly)
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000, http://localhost:3001, http://localhost:3002",
		AllowMethods:     "GET,POST,OPTIONS",
		AllowHeaders:     "*",
		AllowCredentials: false,
		MaxAge:           int((12 * time.Hour).Seconds()),
	}))

	// Early debug line (also shows OPTIONS)
	app.Use(func(c *fiber.Ctx) error {
		log.Printf(">> %s %s origin=%q ct=%q", c.Method(), c.Path(), c.Get("Origin"), c.Get("Content-Type"))
		return c.Next()
	})

	// Static preview for uploaded files
	app.Static("/uploads", "./uploads")

	// Health
	app.Get("/healthz", func(c *fiber.Ctx) error { return c.SendString("ok") })

	// API
	routes.Register(app)

	log.Println("API listening on :3005")
	log.Fatal(app.Listen(":3005"))
}
