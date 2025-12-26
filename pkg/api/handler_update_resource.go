package api

import (
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v3"
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
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid resource ID",
		})
	}

	var req UpdateResourceRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	result, err := s.Manager.Update(uint(id), req.DesiredSpec, req.Revision)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update resource",
		})
	}

	return c.JSON(UpdateResourceResponse{
		ID:         result.Resource.ID,
		Generation: result.Resource.Generation,
		Revision:   result.Resource.Revision,
		Status:     "pending",
	})
}
