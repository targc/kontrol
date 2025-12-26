package api

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/targc/kontrol/pkg/models"
)

type UpdateResourceRequest struct {
	DesiredSpec json.RawMessage `json:"desired_spec" validate:"required"`
	Revision    *int            `json:"revision"`
}

type UpdateResourceResponse struct {
	ID         uint   `json:"id"`
	Generation int    `json:"generation"`
	Revision   int    `json:"revision"`
	Status     string `json:"status"`
}

func (s *Server) HandleUpdateResource(c fiber.Ctx) error {
	id := c.Params("id")

	var req UpdateResourceRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	var resource models.Resource
	if err := s.DB.First(&resource, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Resource not found",
		})
	}

	updates := map[string]interface{}{
		"desired_spec": req.DesiredSpec,
		"generation":   resource.Generation + 1,
	}

	if req.Revision != nil {
		updates["revision"] = *req.Revision
	} else {
		updates["revision"] = resource.Revision + 1
	}

	if err := s.DB.Model(&resource).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update resource",
		})
	}

	s.DB.First(&resource, id)

	return c.JSON(UpdateResourceResponse{
		ID:         resource.ID,
		Generation: resource.Generation,
		Revision:   resource.Revision,
		Status:     "pending",
	})
}
