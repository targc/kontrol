package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/targc/kontrol/pkg/models"
)

type DeleteSyncedStateResponse struct {
	Success bool `json:"success"`
}

func (s *Server) DeleteSyncedState(c fiber.Ctx) error {
	clusterID := c.Locals("cluster_id").(string)
	ctx := c.Context()
	globalResourceID, err := uuid.Parse(c.Params("id"))

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid global resource id"})
	}

	err = s.db.
		WithContext(ctx).
		Unscoped().
		Where("global_resource_id = ? AND cluster_id = ?", globalResourceID, clusterID).
		Delete(&models.GlobalResourceSyncedState{}).
		Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to delete synced state"})
	}

	return c.JSON(DeleteSyncedStateResponse{Success: true})
}
