package api

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/targc/kontrol/pkg/models"
)

type ListOutOfSyncResourcesResponse struct {
	Data []models.Resource `json:"data"`
}

func (s *Server) ListOutOfSyncResources(c fiber.Ctx) error {
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

	// Resources where generation != applied_state.generation OR applied_state doesn't exist
	err := s.db.
		WithContext(ctx).
		Raw(`
			SELECT r.* FROM k_resources r
			LEFT JOIN k_resource_applied_states a ON r.id = a.resource_id AND a.deleted_at IS NULL
			WHERE r.cluster_id = ?
			AND r.deleted_at IS NULL
			AND (a.id IS NULL OR a.generation != r.generation)
			ORDER BY r.created_at ASC
			LIMIT ?
		`, clusterID, limit).
		Scan(&resources).
		Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to list resources"})
	}

	return c.JSON(ListOutOfSyncResourcesResponse{Data: resources})
}
