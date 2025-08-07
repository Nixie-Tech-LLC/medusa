package main

import (
	"log"
	"os"
)

type Environment struct {
	Environment     string
	ServerAddress   string
	SecretKey       string
	DatabaseURL     string
	MigrationsPath  string
	RedisAddress    string
	RedisUsername   string
	RedisPassword   string
	UseSpaces       bool
	SpacesEndpoint  string
	SpacesRegion    string
	SpacesBucket    string
	SpacesCDNURL    string
	SpacesAccessKey string
	SpacesSecretKey string
}

// LoadEnvironment reads and validates env vars
func LoadEnvironment() Environment {
	env := Environment{
		Environment:     os.Getenv("APP_ENV"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		SecretKey:       os.Getenv("JWT_SECRET"),
		ServerAddress:   os.Getenv("SERVER_ADDRESS"),

		RedisAddress:    os.Getenv("REDIS_ADDRESS"),
		RedisUsername:   os.Getenv("REDIS_USERNAME"),
		RedisPassword:   os.Getenv("REDIS_PASSWORD"),

		MigrationsPath:  os.Getenv("MIGRATIONS_PATH"),

		UseSpaces:       os.Getenv("USE_SPACES") == "true",
		SpacesEndpoint:  os.Getenv("SPACES_ENDPOINT"),
		SpacesRegion:    os.Getenv("SPACES_REGION"),
		SpacesBucket:    os.Getenv("SPACES_BUCKET"),
		SpacesCDNURL:    os.Getenv("SPACES_CDN_URL"),
		SpacesAccessKey: os.Getenv("SPACES_ACCESS_KEY"),
		SpacesSecretKey: os.Getenv("SPACES_SECRET_KEY"),
	}

	// Basic validation
	if env.DatabaseURL == "" || env.SecretKey == "" || env.ServerAddress == "" {
		log.Fatal("Missing required environment variables")
	}

	return env
}
