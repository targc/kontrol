package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ResourceAppliedState struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	ResourceID uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex" json:"resource_id"`

	Spec         []byte         `gorm:"type:jsonb" json:"spec"`
	Generation   int            `json:"generation"`
	Revision     int            `json:"revision"`

	Status       string         `gorm:"type:varchar(50)" json:"status"`
	ErrorMessage *string        `gorm:"type:text" json:"error_message,omitempty"`

	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Resource     *Resource      `gorm:"foreignKey:ResourceID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}

func (ResourceAppliedState) TableName() string {
	return "k_resource_applied_states"
}
