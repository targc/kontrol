package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/targc/kontrol/pkg/models"
)

type HardDeleteResourceResponse struct {
	Success bool `json:"success"`
}

func (s *Server) HardDeleteResource(c fiber.Ctx) error {
	clusterID := c.Locals("cluster_id").(string)
	ctx := c.Context()
	resourceID, err := uuid.Parse(c.Params("id"))

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid resource id"})
	}

	// Verify resource belongs to this cluster
	var resource models.Resource

	err = s.db.
		WithContext(ctx).
		Unscoped().
		Where("id = ? AND cluster_id = ?", resourceID, clusterID).
		First(&resource).
		Error

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: "resource not found"})
	}

	// Hard delete the resource
	err = s.db.
		WithContext(ctx).
		Unscoped().
		Delete(&resource).
		Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to delete resource"})
	}

	return c.JSON(HardDeleteResourceResponse{Success: true})
}
