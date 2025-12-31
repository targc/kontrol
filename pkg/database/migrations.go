package database

import (
	"log"

	"gorm.io/gorm"
)

// RunMigrations runs all database migrations including triggers
func RunMigrations(db *gorm.DB) error {
	log.Println("Running database migrations...")

	if err := createGenerationTrigger(db); err != nil {
		return err
	}

	if err := createGlobalResourceGenerationTrigger(db); err != nil {
		return err
	}

	log.Println("Database migrations completed")
	return nil
}

// createGenerationTrigger creates a trigger to auto-increment generation on UPDATE
func createGenerationTrigger(db *gorm.DB) error {
	// Create trigger function
	functionSQL := `
CREATE OR REPLACE FUNCTION increment_resource_generation()
RETURNS TRIGGER AS $$
BEGIN
    -- Only increment generation on UPDATE (not INSERT)
    IF TG_OP = 'UPDATE' THEN
        -- Increment if desired_spec, revision, or deleted_at changed
        IF (NEW.desired_spec IS DISTINCT FROM OLD.desired_spec) OR
           (NEW.revision IS DISTINCT FROM OLD.revision) OR
           (NEW.deleted_at IS DISTINCT FROM OLD.deleted_at) THEN
            NEW.generation := OLD.generation + 1;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
`

	if err := db.Exec(functionSQL).Error; err != nil {
		return err
	}

	// Create trigger (DROP IF EXISTS ensures idempotency)
	triggerSQL := `
DROP TRIGGER IF EXISTS resources_increment_generation ON resources;
CREATE TRIGGER resources_increment_generation
    BEFORE UPDATE ON resources
    FOR EACH ROW
    EXECUTE FUNCTION increment_resource_generation();
`

	if err := db.Exec(triggerSQL).Error; err != nil {
		return err
	}

	log.Println("Generation auto-increment trigger created")
	return nil
}

// createGlobalResourceGenerationTrigger creates a trigger to auto-increment generation on global_resources
func createGlobalResourceGenerationTrigger(db *gorm.DB) error {
	// Reuse the same function, just create a trigger for global_resources table
	triggerSQL := `
DROP TRIGGER IF EXISTS global_resources_increment_generation ON global_resources;
CREATE TRIGGER global_resources_increment_generation
    BEFORE UPDATE ON global_resources
    FOR EACH ROW
    EXECUTE FUNCTION increment_resource_generation();
`

	if err := db.Exec(triggerSQL).Error; err != nil {
		return err
	}

	log.Println("GlobalResource generation auto-increment trigger created")
	return nil
}
