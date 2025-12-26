package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/targc/kontrol/pkg/manager"
	"gorm.io/gorm"
)

type Server struct {
	DB      *gorm.DB
	Manager *manager.ResourceManager
}

func NewServer(db *gorm.DB) *Server {
	return &Server{
		DB:      db,
		Manager: manager.NewResourceManager(db),
	}
}

func (s *Server) SetupRoutes(app *fiber.App) {
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept"},
	}))

	api := app.Group("/api/v1")

	api.Post("/resources", s.HandleCreateResource)
	api.Get("/resources", s.HandleListResources)
	api.Get("/resources/:id", s.HandleGetResource)
	api.Put("/resources/:id", s.HandleUpdateResource)
	api.Delete("/resources/:id", s.HandleDeleteResource)

	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy"})
	})
}
