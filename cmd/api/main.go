package main

import (
	"flag"
	"log"
	"os"

	"app/config"
	"app/docs"
	"app/internal"
	"app/internal/auth"
	"app/internal/cache"
	"app/internal/db"
	"app/internal/example"
	"app/internal/logger"
	custommiddleware "app/internal/middleware"
	"app/internal/redis"
	"app/internal/scheduler"
	"app/internal/uploads"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           Gogo API Template
// @version         1.0
// @description     A production-ready Go API template with authentication and CRUD examples
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8181
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	var (
		port      = flag.String("port", "", "Server port (overrides config)")
		useTestDB = flag.Bool("test-db", false, "Use TEST_DATABASE_URL instead of DATABASE_URL")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	if *useTestDB {
		testDBURL := os.Getenv("TEST_DATABASE_URL")
		if testDBURL == "" {
			log.Fatal("TEST_DATABASE_URL environment variable is required when using --test-db flag")
		}
		cfg.DatabaseURL = testDBURL
		log.Println("Using TEST_DATABASE_URL for database connection")
	}

	if *port != "" {
		cfg.Port = *port
	}

	logger, err := logger.New(logger.Config{
		Level:     cfg.LogLevel,
		Format:    cfg.LogFormat,
		Output:    cfg.LogOutput,
		AddSource: cfg.Debug,
		RequestID: true,
	})
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}

	logger.Info("Starting application",
		"app_name", cfg.AppName,
		"version", cfg.AppVersion,
		"environment", cfg.Environment,
		"debug", cfg.Debug,
	)

	database, err := db.NewConnection(cfg)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	queries := db.New(database)

	redisClient, err := redis.NewConnection(cfg)
	if err != nil {
		logger.Error("Failed to connect to Redis", "error", err)
		log.Fatal("Failed to connect to Redis:", err)
	}
	defer redisClient.Close()

	cacheService := cache.NewRedisCache(redisClient, cfg.AppName+":")

	r := gin.New()
	r.RedirectTrailingSlash = false

	r.Use(custommiddleware.RequestID(logger))
	r.Use(custommiddleware.Recovery(logger))
	// r.Use(custommiddleware.RequestLogging(logger))
	r.Use(custommiddleware.ErrorHandler(logger))
	r.Use(cors.Default())

	api := r.Group("/api/v1")

	app := &internal.App{
		Config:  cfg,
		DB:      database,
		Queries: queries,
		Cache:   cacheService,
		Logger:  logger,
		Api:     api,
		Images:  internal.NewImageService(cfg.FilesBaseURL),
	}

	if cfg.EnableScheduler {
		deps := &scheduler.Dependencies{
			Config:  cfg,
			DB:      database,
			Queries: app.Queries,
			Logger:  logger,
		}

		cronScheduler := scheduler.NewScheduler(deps)
		if err := cronScheduler.RegisterJobs(); err != nil {
			logger.Error("Failed to register scheduler jobs", "error", err)
			log.Fatal("Failed to register scheduler jobs:", err)
		}

		cronScheduler.Start()
		logger.Info("Scheduler started in integrated mode")
		defer cronScheduler.Stop()
	}

	if cfg.JWTSecret == "" {
		logger.Error("JWT_SECRET is required")
		log.Fatal("JWT_SECRET environment variable is required")
	}

	authService := auth.RegisterRoutes(app)
	example.RegisterRoutes(app, authService)
	uploads.RegisterRoutes(app, authService)
	uploads.RegisterPublicRoutes(r, app)

	// Healthcheck (DB ping)
	healthHandler := func(c *gin.Context) {
		if err := app.Queries.Healthcheck(c.Request.Context()); err != nil {
			c.JSON(503, gin.H{
				"status":  "unhealthy",
				"app":     cfg.AppName,
				"version": cfg.AppVersion,
				"env":     cfg.Environment,
			})
			return
		}
		c.JSON(200, gin.H{
			"status":  "healthy",
			"app":     cfg.AppName,
			"version": cfg.AppVersion,
			"env":     cfg.Environment,
		})
	}
	r.Match([]string{"GET", "HEAD"}, "/health", healthHandler)
	api.Match([]string{"GET", "HEAD"}, "/health", healthHandler)

	docs.SwaggerInfo.Host = cfg.AppURL
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	address := ":" + cfg.Port
	logger.Info("Server starting", "address", address)
	if err := r.Run(address); err != nil {
		logger.Error("Server failed to start", "error", err, "address", address)
		log.Fatal(err)
	}
}
