package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ResourceCurrentState struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	ResourceID uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex" json:"resource_id"`

	Spec                []byte         `gorm:"type:jsonb" json:"spec"`
	Generation          int            `json:"generation"`
	Revision            int            `json:"revision"`
	K8sResourceVersion  string         `gorm:"type:varchar(100)" json:"k8s_resource_version"`

	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Resource            *Resource      `gorm:"foreignKey:ResourceID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}

func (ResourceCurrentState) TableName() string {
	return "k_resource_current_states"
}
