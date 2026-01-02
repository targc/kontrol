package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GlobalResource struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	Namespace  string `gorm:"type:varchar(255);not null"`
	Kind       string `gorm:"type:varchar(255);not null"`
	Name       string `gorm:"type:varchar(255);not null"`
	APIVersion string `gorm:"type:varchar(100)"`

	DesiredSpec []byte `gorm:"type:jsonb;not null"`

	Generation int `gorm:"default:1;not null"`
	Revision   int `gorm:"default:1;not null"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (GlobalResource) TableName() string {
	return "k_global_resources"
}
