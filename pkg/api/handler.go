package api

import (
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type Server struct {
	db *gorm.DB
}

func NewServer(db *gorm.DB) *Server {
	return &Server{db: db}
}

func (s *Server) SetupRoutes(app *fiber.App) {
	// Internal API for workers
	int := app.Group("/int/api/v1", s.AuthMiddleware())

	// Cluster registration
	int.Post("/cluster/register", s.RegisterCluster)

	// Resources (for reconciler)
	int.Get("/resources/out-of-sync", s.ListOutOfSyncResources)
	int.Get("/resources/deleted", s.ListDeletedResources)
	int.Post("/resources/:id/applied-state", s.UpsertAppliedState)
	int.Delete("/resources/:id", s.HardDeleteResource)

	// Resources (for watcher)
	int.Post("/resources/:id/current-state", s.UpsertCurrentState)
	int.Delete("/resources/:id/current-state", s.DeleteCurrentState)

	// Resources (for global syncer)
	int.Post("/resources", s.CreateResource)
	int.Delete("/resources/by-key", s.SoftDeleteResourceByKey)

	// Global resources (for global syncer)
	int.Get("/global-resources/out-of-sync", s.ListOutOfSyncGlobalResources)
	int.Get("/global-resources/deleted", s.ListDeletedGlobalResources)
	int.Post("/global-resources/:id/synced-state", s.UpsertSyncedState)
	int.Delete("/global-resources/:id/synced-state", s.DeleteSyncedState)
}
