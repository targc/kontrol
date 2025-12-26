package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/targc/kontrol/pkg/models"
)

type DeleteResourceResponse struct {
	ID      uint   `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (s *Server) HandleDeleteResource(c fiber.Ctx) error {
	id := c.Params("id")

	var resource models.Resource
	if err := s.DB.First(&resource, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Resource not found",
		})
	}

	resource.Generation++

	if err := s.DB.Model(&resource).Update("generation", resource.Generation).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to mark resource for deletion",
		})
	}

	if err := s.DB.Delete(&resource).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete resource",
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(DeleteResourceResponse{
		ID:      resource.ID,
		Status:  "deleting",
		Message: "Resource marked for deletion",
	})
}
