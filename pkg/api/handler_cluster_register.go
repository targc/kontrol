package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/targc/kontrol/pkg/models"
	"gorm.io/gorm/clause"
)

type RegisterClusterResponse struct {
	Success bool `json:"success"`
}

func (s *Server) RegisterCluster(c fiber.Ctx) error {
	clusterID := c.Locals("cluster_id").(string)
	ctx := c.Context()

	cluster := models.Cluster{ID: clusterID}

	err := s.db.
		WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&cluster).
		Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to register cluster"})
	}

	return c.JSON(RegisterClusterResponse{Success: true})
}
