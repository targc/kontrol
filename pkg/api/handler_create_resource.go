package api

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/targc/kontrol/pkg/models"
)

type CreateResourceRequest struct {
	ClusterID   string          `json:"cluster_id" validate:"required"`
	Namespace   string          `json:"namespace" validate:"required"`
	Kind        string          `json:"kind" validate:"required"`
	Name        string          `json:"name" validate:"required"`
	APIVersion  string          `json:"api_version" validate:"required"`
	DesiredSpec json.RawMessage `json:"desired_spec" validate:"required"`
}

type CreateResourceResponse struct {
	ID         uint   `json:"id"`
	ClusterID  string `json:"cluster_id"`
	Namespace  string `json:"namespace"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Generation int    `json:"generation"`
	Revision   int    `json:"revision"`
}

func (s *Server) HandleCreateResource(c fiber.Ctx) error {
	var req CreateResourceRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	resource := models.Resource{
		ClusterID:   req.ClusterID,
		Namespace:   req.Namespace,
		Kind:        req.Kind,
		Name:        req.Name,
		APIVersion:  req.APIVersion,
		DesiredSpec: req.DesiredSpec,
		Generation:  1,
		Revision:    1,
	}

	if err := s.DB.Create(&resource).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create resource",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(CreateResourceResponse{
		ID:         resource.ID,
		ClusterID:  resource.ClusterID,
		Namespace:  resource.Namespace,
		Kind:       resource.Kind,
		Name:       resource.Name,
		Generation: resource.Generation,
		Revision:   resource.Revision,
	})
}
