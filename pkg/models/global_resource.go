package models

import (
	"time"

	"gorm.io/gorm"
)

type GlobalResource struct {
	ID         uint   `gorm:"primaryKey"`
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
	return "global_resources"
}
