package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Resource struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	ClusterID  string         `gorm:"type:varchar(100);not null;index" json:"cluster_id"`
	Namespace  string         `gorm:"type:varchar(255);not null" json:"namespace"`
	Kind       string         `gorm:"type:varchar(255);not null" json:"kind"`
	Name       string         `gorm:"type:varchar(255);not null" json:"name"`
	APIVersion string         `gorm:"type:varchar(100)" json:"api_version"`

	DesiredSpec []byte         `gorm:"type:jsonb;not null" json:"desired_spec"`

	Generation  int            `gorm:"default:1;not null" json:"generation"`
	Revision    int            `gorm:"default:1;not null" json:"revision"`

	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (Resource) TableName() string {
	return "k_resources"
}
