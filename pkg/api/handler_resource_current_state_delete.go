package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/targc/kontrol/pkg/models"
)

type DeleteCurrentStateResponse struct {
	Success bool `json:"success"`
}

func (s *Server) DeleteCurrentState(c fiber.Ctx) error {
	ctx := c.Context()
	resourceID, err := uuid.Parse(c.Params("id"))

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid resource id"})
	}

	err = s.db.
		WithContext(ctx).
		Unscoped().
		Where("resource_id = ?", resourceID).
		Delete(&models.ResourceCurrentState{}).
		Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to delete current state"})
	}

	return c.JSON(DeleteCurrentStateResponse{Success: true})
}
