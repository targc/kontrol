package api

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/targc/kontrol/pkg/models"
)

type ListDeletedGlobalResourcesResponse struct {
	Data []models.GlobalResource `json:"data"`
}

func (s *Server) ListDeletedGlobalResources(c fiber.Ctx) error {
	ctx := c.Context()

	limit := 100
	if l, err := strconv.Atoi(c.Query("limit", "100")); err == nil && l > 0 {
		limit = l
	}
	if limit > 500 {
		limit = 500
	}

	var resources []models.GlobalResource

	err := s.db.
		WithContext(ctx).
		Unscoped().
		Where("deleted_at IS NOT NULL").
		Order("deleted_at ASC").
		Limit(limit).
		Find(&resources).
		Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to list deleted global resources"})
	}

	return c.JSON(ListDeletedGlobalResourcesResponse{Data: resources})
}
