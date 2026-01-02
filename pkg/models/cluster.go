package models

import (
	"time"
)

type Cluster struct {
	ID        string    `gorm:"primaryKey;type:varchar(100)" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Cluster) TableName() string {
	return "k_clusters"
}
