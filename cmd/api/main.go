package main

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/targc/kontrol/pkg/api"
	"github.com/targc/kontrol/pkg/config"
	"github.com/targc/kontrol/pkg/database"
)

func main() {
	ctx := context.Background()

	cfg := config.LoadAPIConfig(ctx)

	db, err := database.Connect(cfg.DBURL)

	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if cfg.AutoMigrate {
		err = database.AutoMigrate(db)

		if err != nil {
			log.Fatalf("failed to run migrations: %v", err)
		}
	}

	app := fiber.New()

	server := api.NewServer(db)
	server.SetupRoutes(app)

	log.Printf("Starting API server on port %s", cfg.ServerPort)

	err = app.Listen(":" + cfg.ServerPort)

	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
