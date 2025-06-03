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
    // load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("failed to load config: %v", err)
    }

    // initialize PostgreSQL
    if err := db.Init(cfg.DatabaseURL); err != nil {
        log.Fatalf("db init: %v", err)
    }

    // run pending migrations
    if err := db.RunMigrations(cfg.MigrationsPath); err != nil {
        log.Fatalf("db migrate: %v", err)
    }

    // set up gin router
    r := gin.Default()

    // register auth (public) routes first:
    admin := r.Group("/api/admin")
    // pass JWTSecret so auth handlers can issue tokens
    adminapi.RegisterAuthRoutes(admin, cfg.JWTSecret)

    // apply JWTMiddleware for all the admin routes that follow
    admin.Use(auth.JWTMiddleware(cfg.JWTSecret))
    adminapi.RegisterScreenRoutes(admin)
    adminapi.RegisterContentRoutes(admin)
    adminapi.RegisterScheduleRoutes(admin)

    // TV routes remain as before (they’ll also see the JWTMiddleware)
    tv := r.Group("/api/tv")
    tv.Use(auth.JWTMiddleware(cfg.JWTSecret))
    tvapi.RegisterScreenRoutes(tv)

    // start
    log.Printf("listening on %s", cfg.ServerAddress)
    if err := r.Run(cfg.ServerAddress); err != nil {
        log.Fatalf("server error: %v", err)
    }
}

