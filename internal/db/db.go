package db

import (
    "database/sql"
    "fmt"
    _ "github.com/lib/pq"
    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

var DB *sql.DB

// Init opens a PostgreSQL connection pool
func Init(databaseURL string) error {
    db, err := sql.Open("postgres", databaseURL)
    if err != nil {
        return err
    }
    if err := db.Ping(); err != nil {
        return err
    }
    DB = db
    return nil
}

// RunMigrations executes SQL migrations from the given path
func RunMigrations(migrationsPath string) error {
    driver, err := postgres.WithInstance(DB, &postgres.Config{})
    if err != nil {
        return fmt.Errorf("create migration driver: %w", err)
    }
    m, err := migrate.NewWithDatabaseInstance(
        "file://"+migrationsPath,
        "postgres",
        driver,
    )
    if err != nil {
        return fmt.Errorf("migrate new instance: %w", err)
    }
    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return fmt.Errorf("migrations up: %w", err)
    }
    return nil
}

