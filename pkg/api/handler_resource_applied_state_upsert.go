package api

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/targc/kontrol/pkg/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UpsertAppliedStateRequest struct {
	Spec         json.RawMessage `json:"spec"`
	Generation   int             `json:"generation"`
	Revision     int             `json:"revision"`
	Status       string          `json:"status"`
	ErrorMessage *string         `json:"error_message"`
}

type UpsertAppliedStateResponse struct {
	Success bool `json:"success"`
}

func (s *Server) UpsertAppliedState(c fiber.Ctx) error {
	ctx := c.Context()
	resourceID, err := uuid.Parse(c.Params("id"))

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid resource id"})
	}

	var req UpsertAppliedStateRequest

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid request body"})
	}

	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	var appliedState models.ResourceAppliedState

	err = tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("resource_id = ?", resourceID).
		First(&appliedState).
		Error

	if err == gorm.ErrRecordNotFound {
		appliedState = models.ResourceAppliedState{
			ID:         uuid.Must(uuid.NewV7()),
			ResourceID: resourceID,
		}

		if err := tx.Create(&appliedState).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to create applied state"})
		}
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to query applied state"})
	}

	err = tx.
		Model(&appliedState).
		Updates(map[string]interface{}{
			"spec":          []byte(req.Spec),
			"generation":    req.Generation,
			"revision":      req.Revision,
			"status":        req.Status,
			"error_message": req.ErrorMessage,
		}).
		Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to update applied state"})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to commit"})
	}

	return c.JSON(UpsertAppliedStateResponse{Success: true})
}
