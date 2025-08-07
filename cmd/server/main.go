package main

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	"github.com/Nixie-Tech-LLC/medusa/internal/redis"
	"github.com/gin-gonic/gin"
)

func main() {
	env := LoadEnvironment()

	// Database setup
	if err := db.Init(env.DatabaseURL); err != nil {
		log.Fatalf("db init: %v", err)
	}
	if err := db.RunMigrations(env.MigrationsPath); err != nil {
		log.Fatalf("db migrate: %v", err)
	}

	sqlxDB, err := sqlx.Connect("postgres", env.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to db via sqlx: %v", err)
	}
	store := db.NewStore(sqlxDB)

	// Redis
	redis.InitRedis(env.RedisAddress, env.RedisUsername, env.RedisPassword)

	// Storage
	storageSystem := InitStorage(env)

	// Templates
	tmpl := LoadTemplates()

	// Gin router
	r := gin.Default()

	// Register routes
	RegisterRoutes(r, env, store, storageSystem, tmpl)

	// Start server
	log.Printf("Listening on %s", env.ServerAddress)
	if err := r.Run(env.ServerAddress); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

