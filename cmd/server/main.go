package main

import (
	"log"
	"os"
    "path/filepath"
    "net/http"
    "strings"

	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	adminapi "github.com/Nixie-Tech-LLC/medusa/internal/http/api/admin/endpoints"
	authapi "github.com/Nixie-Tech-LLC/medusa/internal/http/api/auth/endpoints"
	tvapi "github.com/Nixie-Tech-LLC/medusa/internal/http/api/tv/endpoints"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/middleware"
	"github.com/Nixie-Tech-LLC/medusa/internal/redis"
	"github.com/Nixie-Tech-LLC/medusa/internal/storage"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type Environment struct {
	environment   string
	serverAddress string
	secretKey     string

	databaseURL    string
	migrationsPath string

	redisAddress  string
	redisUsername string
	redisPassword string

	useSpaces       bool
	spacesEndpoint  string
	spacesRegion    string
	spacesBucket    string
	spacesAccessKey string
	spacesSecretKey string
	spacesCDNURL    string
}

func main() {
	env := Environment{
		environment:   os.Getenv("APP_ENV"),
		databaseURL:   os.Getenv("DATABASE_URL"),
		secretKey:     os.Getenv("JWT_SECRET"),
		serverAddress: os.Getenv("SERVER_ADDRESS"),

		redisAddress:  os.Getenv("REDIS_ADDRESS"),
		redisUsername: os.Getenv("REDIS_USERNAME"),
		redisPassword: os.Getenv("REDIS_PASSWORD"),

		migrationsPath: os.Getenv("MIGRATIONS_PATH"),

		useSpaces:       os.Getenv("USE_SPACES") == "true",
		spacesEndpoint:  os.Getenv("SPACES_ENDPOINT"),
		spacesRegion:    os.Getenv("SPACES_REGION"),
		spacesBucket:    os.Getenv("SPACES_BUCKET"),
		spacesCDNURL:    os.Getenv("SPACES_CDN_URL"),
		spacesAccessKey: os.Getenv("SPACES_ACCESS_KEY"),
		spacesSecretKey: os.Getenv("SPACES_SECRET_KEY"),
	}

	// initialize PostgreSQL
	if err := db.Init(env.databaseURL); err != nil {
		log.Fatalf("db init: %v", err)
	}

	// run pending migrations
	if err := db.RunMigrations(env.migrationsPath); err != nil {
		log.Fatalf("db migrate: %v", err)
	}

	// set up gin router
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			// Allow all origins
			return true
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept", "If-None-Match", "X-If-None-Match"},
		ExposeHeaders:    []string{"Content-Length", "ETag", "X-Content-ETag"},
		AllowCredentials: false,
	}))

	sqlxDB, err := sqlx.Connect("postgres", env.databaseURL)

	if err != nil {
		log.Fatalf("Failed to connect to db via sqlx: %v", err)
	}
	store := db.NewStore(sqlxDB)
	redis.InitRedis(env.redisAddress, env.redisUsername, env.redisPassword)

	// Initialize storage system
	var storageSystem storage.Storage
	if env.useSpaces {
		spacesStorage, err := storage.NewSpacesStorage(
			env.spacesEndpoint,
			env.spacesRegion,
			env.spacesBucket,
			env.spacesCDNURL,
			env.spacesAccessKey,
			env.spacesSecretKey,
		)
		if err != nil {
			log.Fatalf("failed to initialize Spaces storage: %v", err)
		}
		storageSystem = spacesStorage
		log.Printf("Using DigitalOcean Spaces storage with CDN: %s", env.spacesCDNURL)
	} else {
		storageSystem = storage.NewLocalStorage("./uploads")
		log.Printf("Using local file storage in ./uploads")
	}
	// register auth (public) routes first:
	admin := r.Group("/api/admin")
	authapi.RegisterAuthRoutes(admin, env.secretKey, store)
	// register middleware AFTER registering signin/signup routes
	admin.Use(middleware.JWTMiddleware(env.secretKey))
	authapi.RegisterSessionRoutes(admin, env.secretKey, store)

	protected := admin.Group("/")
	protected.Use(middleware.JWTMiddleware(env.secretKey))
	// apply JWTMiddleware for all the admin routes that follow
	adminapi.RegisterContentRoutes(protected, store, storageSystem)
	adminapi.RegisterScreenRoutes(protected, store)
	adminapi.RegisterScheduleRoutes(protected)
	adminapi.RegisterPlaylistRoutes(protected, store)

	tv := r.Group("/api/tv")
	tvapi.RegisterPairingRoutes(tv, store)

	// Only serve static uploads directory when using local storage
    if !env.useSpaces {
        // keep your uploads static handler
        r.Static("/uploads", "./uploads")

        // unified integrations handler:
        r.GET("/integrations/*filepath", func(c *gin.Context) {
            // filepath contains a leading “/”, e.g. “/athan” or “/athan/foo.js”
            rel := strings.TrimPrefix(c.Param("filepath"), "/")
            full := filepath.Join("integrations", rel)

            info, err := os.Stat(full)
            if err != nil {
                c.Status(http.StatusNotFound)
                return
            }

            // if it’s a folder, serve its index.html
            if info.IsDir() {
                full = filepath.Join(full, "index.html")
                if _, err := os.Stat(full); err != nil {
                    c.Status(http.StatusNotFound)
                    return
                }
            }

            c.File(full)
        })
    }

	// start
	log.Printf("listening on %s", env.serverAddress)
	if err := r.Run(env.serverAddress); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
