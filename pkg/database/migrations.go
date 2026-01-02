package database

import (
	"log"

	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) error {
	log.Println("Running database migrations...")

	err := createGenerationTrigger(db)

	if err != nil {
		return err
	}

	err = createGlobalResourceGenerationTrigger(db)

	if err != nil {
		return err
	}

	log.Println("Database migrations completed")

	return nil
}

func createGenerationTrigger(db *gorm.DB) error {
	functionSQL := `
CREATE OR REPLACE FUNCTION increment_resource_generation()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
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

	err := db.Exec(functionSQL).Error

	if err != nil {
		return err
	}

	triggerSQL := `
DROP TRIGGER IF EXISTS k_resources_increment_generation ON k_resources;
CREATE TRIGGER k_resources_increment_generation
    BEFORE UPDATE ON k_resources
    FOR EACH ROW
    EXECUTE FUNCTION increment_resource_generation();
`

	err = db.Exec(triggerSQL).Error

	if err != nil {
		return err
	}

	log.Println("Generation auto-increment trigger created")

	return nil
}

func createGlobalResourceGenerationTrigger(db *gorm.DB) error {
	triggerSQL := `
DROP TRIGGER IF EXISTS k_global_resources_increment_generation ON k_global_resources;
CREATE TRIGGER k_global_resources_increment_generation
    BEFORE UPDATE ON k_global_resources
    FOR EACH ROW
    EXECUTE FUNCTION increment_resource_generation();
`

	err := db.Exec(triggerSQL).Error

	if err != nil {
		return err
	}

	log.Println("GlobalResource generation auto-increment trigger created")

	return nil
}
