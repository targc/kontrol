package api

import (
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type GlobalResourceForSync struct {
	ID          uuid.UUID       `json:"id"`
	Namespace   string          `json:"namespace"`
	Kind        string          `json:"kind"`
	Name        string          `json:"name"`
	APIVersion  string          `json:"api_version"`
	DesiredSpec json.RawMessage `json:"desired_spec"`
	Generation  int             `json:"generation"`
	Revision    int             `json:"revision"`
}

type ListOutOfSyncGlobalResourcesResponse struct {
	Data []GlobalResourceForSync `json:"data"`
}

func (s *Server) ListOutOfSyncGlobalResources(c fiber.Ctx) error {
	clusterID := c.Locals("cluster_id").(string)
	ctx := c.Context()

	limit := 100
	if l, err := strconv.Atoi(c.Query("limit", "100")); err == nil && l > 0 {
		limit = l
	}
	if limit > 500 {
		limit = 500
	}

	var resources []GlobalResourceForSync

	// Global resources where synced_generation < generation OR synced_state doesn't exist for this cluster
	err := s.db.
		WithContext(ctx).
		Raw(`
			SELECT gr.id, gr.namespace, gr.kind, gr.name, gr.api_version, gr.desired_spec, gr.generation, gr.revision
			FROM k_global_resources gr
			LEFT JOIN k_global_resource_synced_states ss
				ON gr.id = ss.global_resource_id
				AND ss.cluster_id = ?
				AND ss.deleted_at IS NULL
			WHERE gr.deleted_at IS NULL
			AND (ss.id IS NULL OR ss.synced_generation < gr.generation)
			ORDER BY gr.created_at ASC
			LIMIT ?
		`, clusterID, limit).
		Scan(&resources).
		Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to list global resources"})
	}

	return c.JSON(ListOutOfSyncGlobalResourcesResponse{Data: resources})
}
