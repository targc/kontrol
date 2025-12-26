package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/targc/kontrol/pkg/api"
	"github.com/targc/kontrol/pkg/config"
	"github.com/targc/kontrol/pkg/database"
)

func main() {
	log.Println("Starting Kontrol API Server...")

	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	app := fiber.New(fiber.Config{
		AppName: "Kontrol API Server",
	})

	server := api.NewServer(db)
	server.SetupRoutes(app)

	log.Printf("API Server listening on port %s", cfg.ServerPort)

	if err := app.Listen(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
