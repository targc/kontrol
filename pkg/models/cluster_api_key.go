package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClusterAPIKey struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	ClusterID string         `gorm:"type:varchar(100);not null;index" json:"cluster_id"`
	KeyHash   string         `gorm:"type:varchar(255);not null" json:"-"`
	Name      string         `gorm:"type:varchar(100)" json:"name"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Cluster *Cluster `gorm:"foreignKey:ClusterID;references:ID" json:"-"`
}

func (ClusterAPIKey) TableName() string {
	return "k_cluster_api_keys"
}
