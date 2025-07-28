package main

import (
	"log"
	"os"

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
	"github.com/joho/godotenv"
)

func main() {
	// load configuration only if not running app locally
	if os.Getenv("APP_ENV") != "dev" {
		err := godotenv.Load()
		if err != nil {
			log.Fatalf("failed to load config: %v", err)
		}
	}

	databaseUrl := os.Getenv("DATABASE_URL")
	secretKey := os.Getenv("JWT_SECRET")
	serverAddress := os.Getenv("SERVER_ADDRESS")

	redisAddress := os.Getenv("REDIS_ADDRESS")
	redisUsername := os.Getenv("REDIS_USERNAME")
	redisPassword := os.Getenv("REDIS_PASSWORD")

	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	mqttBrokerURL := os.Getenv("MQTT_BROKER_URL")
	mqttBrokerUser := os.Getenv("MQTT_BROKER_USER")
	mqttBrokerPass := os.Getenv("MQTT_BROKER_PASS")

	// Storage configuration
	useSpaces := os.Getenv("USE_SPACES") == "true"
	spacesEndpoint := os.Getenv("SPACES_ENDPOINT")
	spacesRegion := os.Getenv("SPACES_REGION")
	spacesBucket := os.Getenv("SPACES_BUCKET")
	spacesCDNURL := os.Getenv("SPACES_CDN_URL")
	spacesAccessKey := os.Getenv("SPACES_ACCESS_KEY")
	spacesSecretKey := os.Getenv("SPACES_SECRET_KEY")

	// Set MQTT broker URL if provided
	if mqttBrokerURL != "" {
		middleware.SetBrokerURL(mqttBrokerURL)
		middleware.SetBrokerUser(mqttBrokerUser)
		middleware.SetBrokerPass(mqttBrokerPass)
	}

	// initialize PostgreSQL
	if err := db.Init(databaseUrl); err != nil {
		log.Fatalf("db init: %v", err)
	}

	// run pending migrations
	if err := db.RunMigrations(migrationsPath); err != nil {
		log.Fatalf("db migrate: %v", err)
	}

	// initialize MQTT
	if _, err := middleware.CreateMQTTClient("medusa-app-dev"); err != nil {
		log.Fatalf("mqtt init: %v", err)
	}

	// set up gin router
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			// Allow all origins
			return true
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
	}))

	sqlxDB, err := sqlx.Connect("postgres", databaseUrl)

	if err != nil {
		log.Fatalf("Failed to connect to db via sqlx: %v", err)
	}
	store := db.NewStore(sqlxDB)
	redis.InitRedis(redisAddress, redisUsername, redisPassword)

	// Initialize storage system
	var storageSystem storage.Storage
	if useSpaces {
		spacesStorage, err := storage.NewSpacesStorage(
			spacesEndpoint,
			spacesRegion,
			spacesBucket,
			spacesCDNURL,
			spacesAccessKey,
			spacesSecretKey,
		)
		if err != nil {
			log.Fatalf("failed to initialize Spaces storage: %v", err)
		}
		storageSystem = spacesStorage
		log.Printf("Using DigitalOcean Spaces storage with CDN: %s", spacesCDNURL)
	} else {
		storageSystem = storage.NewLocalStorage("./uploads")
		log.Printf("Using local file storage in ./uploads")
	}
	// register auth (public) routes first:
	admin := r.Group("/api/admin")
	authapi.RegisterAuthRoutes(admin, secretKey, store)
	// register middleware AFTER registering signin/signup routes
	admin.Use(middleware.JWTMiddleware(secretKey))
	authapi.RegisterSessionRoutes(admin, secretKey, store)

	protected := admin.Group("/")
	protected.Use(middleware.JWTMiddleware(secretKey))
	// apply JWTMiddleware for all the admin routes that follow
	adminapi.RegisterContentRoutes(protected, store, storageSystem)
	adminapi.RegisterScreenRoutes(protected, store)
	adminapi.RegisterScheduleRoutes(protected)
	adminapi.RegisterPlaylistRoutes(protected, store)

	tv := r.Group("/api/tv")
	tvapi.RegisterPairingRoutes(tv, store)

	// Only serve static uploads directory when using local storage
	if !useSpaces {
		r.Static("/uploads", "./uploads")
	}

	// start
	log.Printf("listening on %s", serverAddress)
	if err := r.Run(serverAddress); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
