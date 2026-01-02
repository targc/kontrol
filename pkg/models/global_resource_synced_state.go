package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GlobalResourceSyncedState struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	GlobalResourceID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_global_cluster"`
	ClusterID        string `gorm:"type:varchar(100);not null;uniqueIndex:idx_global_cluster"`
	SyncedGeneration int    `gorm:"default:1;not null"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (GlobalResourceSyncedState) TableName() string {
	return "k_global_resource_synced_states"
}
