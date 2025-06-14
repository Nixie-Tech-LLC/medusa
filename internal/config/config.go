package config

import (
    "os"
)

type Config struct {
    ServerAddress  string
    DatabaseURL    string
    MigrationsPath string
    JWTSecret      string
}

func Load() (*Config, error) {
    return &Config{
        ServerAddress:  getEnv("SERVER_ADDRESS", ":9000"),
        DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:password@localhost:5432/medusa?sslmode=disable"),
        MigrationsPath: getEnv("MIGRATIONS_PATH", "./migrations"),
        JWTSecret:      getEnv("JWT_SECRET", "replace_me_with_strong_secret"),
    }, nil
}

func getEnv(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}

