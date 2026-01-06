package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/targc/kontrol/pkg/models"
)

type SoftDeleteResourceByKeyRequest struct {
	Namespace string `json:"namespace"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
}

type SoftDeleteResourceByKeyResponse struct {
	Success bool `json:"success"`
}

func (s *Server) SoftDeleteResourceByKey(c fiber.Ctx) error {
	clusterID := c.Locals("cluster_id").(string)
	ctx := c.Context()

	var req SoftDeleteResourceByKeyRequest

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid request body"})
	}

	err := s.db.
		WithContext(ctx).
		Where("cluster_id = ? AND namespace = ? AND kind = ? AND name = ?",
			clusterID, req.Namespace, req.Kind, req.Name).
		Delete(&models.Resource{}).
		Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to delete resource"})
	}

	return c.JSON(SoftDeleteResourceByKeyResponse{Success: true})
}
