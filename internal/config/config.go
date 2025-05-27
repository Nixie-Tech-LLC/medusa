package config

import (
    "fmt"
    "os"
)

// Config holds environment-based settings
type Config struct {
    DatabaseURL    string
    MigrationsPath string
    JWTSecret      string
    ServerAddress  string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        return nil, fmt.Errorf("DATABASE_URL is required")
    }
    jwt := os.Getenv("JWT_SECRET")
    if jwt == "" {
        return nil, fmt.Errorf("JWT_SECRET is required")
    }
    addr := os.Getenv("SERVER_ADDRESS")
    if addr == "" {
        addr = ":8080"
    }
    migrations := os.Getenv("MIGRATIONS_PATH")
    if migrations == "" {
        migrations = "./migrations"
    }
    return &Config{
        DatabaseURL:    dbURL,
        MigrationsPath: migrations,
        JWTSecret:      jwt,
        ServerAddress:  addr,
    }, nil
}
