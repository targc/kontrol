package api

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/targc/kontrol/pkg/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UpsertCurrentStateRequest struct {
	Spec               json.RawMessage `json:"spec"`
	Generation         int             `json:"generation"`
	Revision           int             `json:"revision"`
	K8sResourceVersion string          `json:"k8s_resource_version"`
}

type UpsertCurrentStateResponse struct {
	Success bool `json:"success"`
}

func (s *Server) UpsertCurrentState(c fiber.Ctx) error {
	ctx := c.Context()
	resourceID, err := uuid.Parse(c.Params("id"))

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid resource id"})
	}

	var req UpsertCurrentStateRequest

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid request body"})
	}

	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	var currentState models.ResourceCurrentState

	err = tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("resource_id = ?", resourceID).
		First(&currentState).
		Error

	if err == gorm.ErrRecordNotFound {
		currentState = models.ResourceCurrentState{
			ID:         uuid.Must(uuid.NewV7()),
			ResourceID: resourceID,
		}

		if err := tx.Create(&currentState).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to create current state"})
		}
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to query current state"})
	}

	// Skip update if k8s_resource_version hasn't changed
	if currentState.K8sResourceVersion == req.K8sResourceVersion {
		if err := tx.Commit().Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to commit"})
		}
		return c.JSON(UpsertCurrentStateResponse{Success: true})
	}

	err = tx.
		Model(&currentState).
		Updates(map[string]interface{}{
			"spec":                 []byte(req.Spec),
			"generation":           req.Generation,
			"revision":             req.Revision,
			"k8s_resource_version": req.K8sResourceVersion,
		}).
		Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to update current state"})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to commit"})
	}

	return c.JSON(UpsertCurrentStateResponse{Success: true})
}
