package main

import (
    "github.com/gin-gonic/gin"
    "github.com/Nixie-Tech-LLC/medusa/internal/api"
    "github.com/Nixie-Tech-LLC/medusa/internal/auth"
    "github.com/Nixie-Tech-LLC/medusa/internal/db"
)

func main() {
    // 1. Connect to DB
    if err := db.Init(); err != nil {
        log.Fatalf("db init: %v", err)
    }

    // 2. Set up router
    r := gin.Default()
    r.Use(auth.JWTMiddleware())

    // 3. Register endpoints
    api.RegisterScreenRoutes(r.Group("/api/screens"))
    api.RegisterContentRoutes(r.Group("/api/content"))
    api.RegisterScheduleRoutes(r.Group("/api/schedules"))

    // 4. Start server
    addr := ":8080"
    log.Printf("listening on %s", addr)
    if err := r.Run(addr); err != nil {
        log.Fatalf("server error: %v", err)
    }
}

