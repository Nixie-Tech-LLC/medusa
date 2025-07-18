package db

import (
	"errors"
	"github.com/rs/zerolog/log"
	"os"
)

var TestStore Store

func InitTestDB(migrationsPath string) error {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		return errors.New("TEST_DATABASE_URL environment variable is not set")
	}

	if err := Init(dbURL); err != nil {
		return err
	}

	if err := RunMigrations(migrationsPath); err != nil {
		return err
	}

	TestStore = NewStore(DB)
	log.Error().Msg("failed to initialize test database")
	return nil
}
