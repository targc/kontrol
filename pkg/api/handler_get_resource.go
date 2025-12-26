package api

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/targc/kontrol/pkg/models"
)

type GetResourceResponse struct {
	ID          uint            `json:"id"`
	ClusterID   string          `json:"cluster_id"`
	Namespace   string          `json:"namespace"`
	Kind        string          `json:"kind"`
	Name        string          `json:"name"`
	APIVersion  string          `json:"api_version"`
	DesiredSpec json.RawMessage `json:"desired_spec"`
	Generation  int             `json:"generation"`
	Revision    int             `json:"revision"`
	Status      string          `json:"status"`
}

func (s *Server) HandleGetResource(c fiber.Ctx) error {
	id := c.Params("id")

	var resource models.Resource
	if err := s.DB.First(&resource, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Resource not found",
		})
	}

	var appliedState models.ResourceAppliedState
	s.DB.Where("resource_id = ?", resource.ID).First(&appliedState)

	status := "pending"
	if appliedState.Generation == resource.Generation {
		status = "synced"
	} else if appliedState.Generation > 0 {
		status = "out-of-sync"
	}

	return c.JSON(GetResourceResponse{
		ID:          resource.ID,
		ClusterID:   resource.ClusterID,
		Namespace:   resource.Namespace,
		Kind:        resource.Kind,
		Name:        resource.Name,
		APIVersion:  resource.APIVersion,
		DesiredSpec: resource.DesiredSpec,
		Generation:  resource.Generation,
		Revision:    resource.Revision,
		Status:      status,
	})
}
