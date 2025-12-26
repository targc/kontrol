package api

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
)

type DeleteResourceResponse struct {
	ID      uint   `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (s *Server) HandleDeleteResource(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid resource ID",
		})
	}

	if err := s.Manager.Delete(uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete resource",
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(DeleteResourceResponse{
		ID:      uint(id),
		Status:  "deleting",
		Message: "Resource marked for deletion",
	})
}
