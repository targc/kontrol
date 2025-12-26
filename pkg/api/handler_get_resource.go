package api

import (
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v3"
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
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid resource ID",
		})
	}

	result, err := s.Manager.Get(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Resource not found",
		})
	}

	status := "pending"
	if result.AppliedState != nil {
		if result.AppliedState.Generation == result.Resource.Generation {
			status = "synced"
		} else {
			status = "out-of-sync"
		}
	}

	return c.JSON(GetResourceResponse{
		ID:          result.Resource.ID,
		ClusterID:   result.Resource.ClusterID,
		Namespace:   result.Resource.Namespace,
		Kind:        result.Resource.Kind,
		Name:        result.Resource.Name,
		APIVersion:  result.Resource.APIVersion,
		DesiredSpec: result.Resource.DesiredSpec,
		Generation:  result.Resource.Generation,
		Revision:    result.Resource.Revision,
		Status:      status,
	})
}
