// path: main.go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Auto-load .env from current working directory
	_ "github.com/joho/godotenv/autoload"

	"meniba/database"
	"meniba/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func main() {
	addr := getenv("ADDR", ":3005")
	uploadDir := getenv("UPLOAD_DIR", "uploads")
	allowOrigins := getenv("CORS_ALLOW_ORIGINS", "http://localhost:3000")
	mustMkdir(uploadDir)

	// DB connect
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := database.Connect(ctx); err != nil {
		log.Fatalf("mongo connect failed: %v", err)
	}
	defer database.Disconnect(context.Background())

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		AppName:               "meniba-api",
	})
	app.Use(recover.New(), logger.New())

	// CORS (handles preflight automatically)
	app.Use(cors.New(cors.Config{
		AllowOrigins:     allowOrigins,
		AllowMethods:     "GET,POST,OPTIONS",
		AllowHeaders:     "Content-Type, Authorization",
		AllowCredentials: false,
		MaxAge:           600,
	}))

	// Explicit OPTIONS fallback
	app.Options("/*", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })

	// static + health
	app.Static("/uploads", uploadDir, fiber.Static{Browse: true})
	app.Get("/healthz", func(c *fiber.Ctx) error { return c.SendString("ok") })
	app.Get("/health/db", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := database.Client().Ping(ctx, readpref.Primary()); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).SendString("mongo down: " + err.Error())
		}
		return c.SendString("mongo ok")
	})

	// routes
	routes.Register(app)

	// run + graceful shutdown
	go func() {
		log.Printf("listening on %s", addr)
		if err := app.Listen(addr); err != nil {
			log.Printf("server stopped: %v", err)
		}
	}()
	waitForStop()
	log.Println("shutting down...")
	_ = app.Shutdown()
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func mustMkdir(p string) {
	if err := os.MkdirAll(p, 0o755); err != nil {
		log.Fatalf("mkdir %s: %v", p, err)
	}
}

func waitForStop() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
}
