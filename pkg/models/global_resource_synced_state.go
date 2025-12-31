package models

import (
	"time"

	"gorm.io/gorm"
)

type GlobalResourceSyncedState struct {
	ID               uint   `gorm:"primaryKey"`
	GlobalResourceID uint   `gorm:"not null;uniqueIndex:idx_global_cluster"`
	ClusterID        string `gorm:"type:varchar(100);not null;uniqueIndex:idx_global_cluster"`
	SyncedGeneration int    `gorm:"default:1;not null"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (GlobalResourceSyncedState) TableName() string {
	return "global_resource_synced_states"
}
