package database

import (
	"fmt"
	"log"

	"github.com/targc/kontrol/pkg/config"
	"github.com/targc/kontrol/pkg/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DBURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Database connected successfully")

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	log.Println("Running auto migration...")

	err := db.AutoMigrate(
		&models.Cluster{},
		&models.Resource{},
		&models.ResourceCurrentState{},
		&models.ResourceAppliedState{},
		&models.GlobalResource{},
		&models.GlobalResourceSyncedState{},
	)

	if err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	log.Println("Auto migration completed")

	err = RunMigrations(db)

	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
