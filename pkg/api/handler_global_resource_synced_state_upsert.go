package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/targc/kontrol/pkg/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UpsertSyncedStateRequest struct {
	SyncedGeneration int `json:"synced_generation"`
}

type UpsertSyncedStateResponse struct {
	Success bool `json:"success"`
}

func (s *Server) UpsertSyncedState(c fiber.Ctx) error {
	clusterID := c.Locals("cluster_id").(string)
	ctx := c.Context()
	globalResourceID, err := uuid.Parse(c.Params("id"))

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid global resource id"})
	}

	var req UpsertSyncedStateRequest

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid request body"})
	}

	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	var syncedState models.GlobalResourceSyncedState

	err = tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("global_resource_id = ? AND cluster_id = ?", globalResourceID, clusterID).
		First(&syncedState).
		Error

	if err == gorm.ErrRecordNotFound {
		syncedState = models.GlobalResourceSyncedState{
			ID:               uuid.Must(uuid.NewV7()),
			GlobalResourceID: globalResourceID,
			ClusterID:        clusterID,
			SyncedGeneration: req.SyncedGeneration,
		}

		if err := tx.Create(&syncedState).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to create synced state"})
		}
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to query synced state"})
	} else {
		err = tx.
			Model(&syncedState).
			Update("synced_generation", req.SyncedGeneration).
			Error

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to update synced state"})
		}
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to commit"})
	}

	return c.JSON(UpsertSyncedStateResponse{Success: true})
}
