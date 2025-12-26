package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/targc/kontrol/pkg/models"
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

	var resources []models.Resource
	query := s.DB.Model(&models.Resource{})

	if clusterID != "" {
		query = query.Where("cluster_id = ?", clusterID)
	}

	var total int64
	query.Count(&total)

	if err := query.Find(&resources).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch resources",
		})
	}

	summaries := make([]ResourceSummary, len(resources))
	for i, r := range resources {
		summaries[i] = ResourceSummary{
			ID:         r.ID,
			ClusterID:  r.ClusterID,
			Namespace:  r.Namespace,
			Kind:       r.Kind,
			Name:       r.Name,
			Generation: r.Generation,
			Revision:   r.Revision,
		}
	}

	return c.JSON(ListResourcesResponse{
		Resources: summaries,
		Total:     total,
	})
}
