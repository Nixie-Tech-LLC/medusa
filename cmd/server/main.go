package main

import (
	"log"
	"os"

	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	adminapi "github.com/Nixie-Tech-LLC/medusa/internal/http/api/admin/endpoints"
	authapi "github.com/Nixie-Tech-LLC/medusa/internal/http/api/auth/endpoints"
	tvapi "github.com/Nixie-Tech-LLC/medusa/internal/http/api/tv/endpoints"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/middleware"
	redisclient "github.com/Nixie-Tech-LLC/medusa/internal/redis"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/jmoiron/sqlx"
)

func main() {
	// load configuration only if not running app locally
	if os.Getenv("APP_ENV") != "local" {
		err := godotenv.Load()
		if err != nil {
			log.Fatalf("failed to load config: %v", err)
		}
	}

	databaseUrl := os.Getenv("DATABASE_URL")
	secretKey := os.Getenv("JWT_SECRET")
	serverAddress := os.Getenv("SERVER_ADDRESS")
	migrationsPath := os.Getenv("MIGRATIONS_PATH")

	// initialize PostgreSQL
	if err := db.Init(databaseUrl); err != nil {
		log.Fatalf("db init: %v", err)
	}

	// run pending migrations
	if err := db.RunMigrations(migrationsPath); err != nil {
		log.Fatalf("db migrate: %v", err)
	}

	// set up gin router
	r := gin.Default()
	
	sqlxDB, err := sqlx.Connect("postgres", databaseUrl)

	if err != nil {
		log.Fatalf("Failed to connect to db via sqlx: %v", err)
	}
	store := db.NewStore(sqlxDB)
	redisclient.InitRedis()
	// register auth (public) routes first:
	admin := r.Group("/api/admin")

	// pass JWTSecret so auth handlers can issue tokens
	authapi.RegisterAuthRoutes(admin, secretKey, store)

	protected := admin.Group("/")
	protected.Use(middleware.JWTMiddleware(secretKey))
	// apply JWTMiddleware for all the admin routes that follow
	adminapi.RegisterContentRoutes(protected, store)
	adminapi.RegisterScheduleRoutes(protected)

	tv := r.Group("/api/tv")
	tv.Use(middleware.JWTMiddleware(secretKey))
	tvapi.RegisterScreenRoutes(tv, store)
	tvapi.RegisterPairingRoutes(tv)

	// start
	log.Printf("listening on %s", serverAddress)
	if err := r.Run(serverAddress); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
