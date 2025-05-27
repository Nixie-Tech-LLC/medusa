package main

import (
    "log"
    "github.com/gin-gonic/gin"
    "github.com/Nixie-Tech-LLC/medusa/internal/config"
    "github.com/Nixie-Tech-LLC/medusa/internal/db"
    "github.com/Nixie-Tech-LLC/medusa/internal/auth"
    adminapi "github.com/Nixie-Tech-LLC/medusa/internal/api/admin"
    tvapi    "github.com/Nixie-Tech-LLC/medusa/internal/api/tv"

)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("failed to load config: %v", err)
    }

    // Initialize PostgreSQL connection
    if err := db.Init(cfg.DatabaseURL); err != nil {
        log.Fatalf("db init: %v", err)
    }

    // Run pending migrations
    if err := db.RunMigrations(cfg.MigrationsPath); err != nil {
        log.Fatalf("db migrate: %v", err)
    }

    // Set up Gin router
    r := gin.Default()
    r.Use(auth.JWTMiddleware(cfg.JWTSecret))

    // Admin (webapp) routes
    admin := r.Group("/api/admin")
    adminapi.RegisterScreenRoutes(admin)
    adminapi.RegisterContentRoutes(admin)
    adminapi.RegisterScheduleRoutes(admin)

    // TV (device) routes
    tv := r.Group("/api/tv")
    tvapi.RegisterScreenRoutes(tv)

    // Start server
    log.Printf("listening on %s", cfg.ServerAddress)
    if err := r.Run(cfg.ServerAddress); err != nil {
        log.Fatalf("server error: %v", err)
    }
}

