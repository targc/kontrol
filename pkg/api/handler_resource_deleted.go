package api

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/targc/kontrol/pkg/models"
)

type ListDeletedResourcesResponse struct {
	Data []models.Resource `json:"data"`
}

func (s *Server) ListDeletedResources(c fiber.Ctx) error {
	clusterID := c.Locals("cluster_id").(string)
	ctx := c.Context()

	limit := 100
	if l, err := strconv.Atoi(c.Query("limit", "100")); err == nil && l > 0 {
		limit = l
	}
	if limit > 500 {
		limit = 500
	}

	var resources []models.Resource

	err := s.db.
		WithContext(ctx).
		Unscoped().
		Where("cluster_id = ? AND deleted_at IS NOT NULL", clusterID).
		Order("deleted_at ASC").
		Limit(limit).
		Find(&resources).
		Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to list deleted resources"})
	}

	return c.JSON(ListDeletedResourcesResponse{Data: resources})
}
