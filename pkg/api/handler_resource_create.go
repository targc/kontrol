package api

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/targc/kontrol/pkg/models"
)

type CreateResourceRequest struct {
	Namespace   string          `json:"namespace"`
	Kind        string          `json:"kind"`
	Name        string          `json:"name"`
	APIVersion  string          `json:"api_version"`
	DesiredSpec json.RawMessage `json:"desired_spec"`
	Revision    int             `json:"revision"`
}

type CreateResourceResponse struct {
	Data *models.Resource `json:"data"`
}

func (s *Server) CreateResource(c fiber.Ctx) error {
	clusterID := c.Locals("cluster_id").(string)
	ctx := c.Context()

	var req CreateResourceRequest

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid request body"})
	}

	resource := &models.Resource{
		ID:          uuid.Must(uuid.NewV7()),
		ClusterID:   clusterID,
		Namespace:   req.Namespace,
		Kind:        req.Kind,
		Name:        req.Name,
		APIVersion:  req.APIVersion,
		DesiredSpec: req.DesiredSpec,
		Revision:    req.Revision,
	}

	err := s.db.
		WithContext(ctx).
		Create(resource).
		Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to create resource"})
	}

	return c.Status(fiber.StatusCreated).JSON(CreateResourceResponse{Data: resource})
}
