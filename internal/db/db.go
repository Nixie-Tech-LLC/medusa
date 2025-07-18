package db

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

/*
TODO: Handle errors inside getter and setter functions
so try catch isn't necessary for each usage
*/

var (
	DB *sqlx.DB
)

// opens a PostgreSQL connection and assigns it to DB.
func Init(databaseURL string) error {
	const maxRetries = 10
	const retryInterval = 2 * time.Second
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		DB, err = sqlx.Connect("postgres", databaseURL)
		if err == nil {
			log.Info().Msg("connected to database")
			return nil
		}

		log.Error().Err(err).
			Int("attempt", attempt).
			Msgf("failed to connect to database, retrying in %s", retryInterval)

		time.Sleep(retryInterval)
	}

	return fmt.Errorf("could not connect to database after %d attempts: %w", maxRetries, err)
}


// finds all “*.up.sql” files in migrationsPath (sorted by name)
// and executes their SQL contents in order. It ignores “*.down.sql” files.
// returns that error immediately upon execution failure
func RunMigrations(migrationsPath string) error {
	// find all files matching “*.up.sql”
	pattern := filepath.Join(migrationsPath, "*.up.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		log.Error().Msg("failed to list up migrations")
		return fmt.Errorf("failed to glob migrations: %w", err)
	}
	if len(files) == 0 {
		// nothing to do
		return nil
	}

	// sort file names so that they run in deterministic order
	sort.Strings(files)

	// for each file, read its contents and execute as a single SQL statement
	for _, file := range files {
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			log.Error().Msg("failed to read migration file")
			return fmt.Errorf("could not read migration %q: %w", file, err)
		}
		sqlStmt := string(sqlBytes)
		if sqlStmt == "" {
			continue
		}
		if _, err := DB.Exec(sqlStmt); err != nil {
			return fmt.Errorf("error executing migration %q: %w", file, err)
		}
	}
	return nil
}

