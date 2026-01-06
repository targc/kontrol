package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/targc/kontrol/pkg/models"
	"golang.org/x/crypto/bcrypt"
)

func (s *Server) AuthMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		apiKey := c.Get("X-API-Key")
		clusterID := c.Get("X-Cluster-ID")

		if apiKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{Error: "missing api key"})
		}

		if clusterID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{Error: "missing cluster id"})
		}

		var keys []models.ClusterAPIKey

		err := s.db.
			WithContext(c.Context()).
			Where("cluster_id = ? AND deleted_at IS NULL", clusterID).
			Find(&keys).
			Error

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal error"})
		}

		for _, key := range keys {
			if bcrypt.CompareHashAndPassword([]byte(key.KeyHash), []byte(apiKey)) == nil {
				c.Locals("cluster_id", clusterID)
				return c.Next()
			}
		}

		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{Error: "invalid api key"})
	}
}
