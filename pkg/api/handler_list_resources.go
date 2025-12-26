package api

import (
	"github.com/gofiber/fiber/v3"
)

type ListResourcesResponse struct {
	Resources []ResourceSummary `json:"resources"`
	Total     int64             `json:"total"`
}

type ResourceSummary struct {
	ID         uint   `json:"id"`
	ClusterID  string `json:"cluster_id"`
	Namespace  string `json:"namespace"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Generation int    `json:"generation"`
	Revision   int    `json:"revision"`
}

func (s *Server) HandleListResources(c fiber.Ctx) error {
	clusterID := c.Query("cluster_id")

	results, err := s.Manager.List(clusterID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch resources",
		})
	}

	summaries := make([]ResourceSummary, len(results))
	for i, r := range results {
		summaries[i] = ResourceSummary{
			ID:         r.Resource.ID,
			ClusterID:  r.Resource.ClusterID,
			Namespace:  r.Resource.Namespace,
			Kind:       r.Resource.Kind,
			Name:       r.Resource.Name,
			Generation: r.Resource.Generation,
			Revision:   r.Resource.Revision,
		}
	}

	return c.JSON(ListResourcesResponse{
		Resources: summaries,
		Total:     int64(len(results)),
	})
}
