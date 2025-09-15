package main

import (
	"context"
	"log"

	"app/config"
	"app/docs"
	"app/internal"
	"app/internal/cache"
	"app/internal/cities"
	"app/internal/db"
	"app/internal/logger"
	custommiddleware "app/internal/middleware"
	"app/internal/redis"
	"app/internal/scheduler"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           MyApp API
// @version         1.0
// @description     Api for SmartCity project
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
	// Config
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Logger
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

	logger.Info(context.TODO(), "Starting application",
		"app_name", cfg.AppName,
		"version", cfg.AppVersion,
		"environment", cfg.Environment,
		"debug", cfg.Debug,
	)

	// DB
	database, err := db.NewConnection(cfg)
	if err != nil {
		logger.Error(context.TODO(), "Failed to connect to database", err)
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	// Redis
	redisClient, err := redis.NewConnection(cfg)
	if err != nil {
		logger.Error(context.TODO(), "Failed to connect to Redis", err)
		log.Fatal("Failed to connect to Redis:", err)
	}
	defer redisClient.Close()

	// Cache
	cacheService := cache.NewRedisCache(redisClient, cfg.AppName+":")

	// Gin
	r := gin.New()

	r.RedirectTrailingSlash = false

	// Middleware
	r.Use(custommiddleware.RequestID(logger))
	r.Use(custommiddleware.Recovery(logger))
	// r.Use(custommiddleware.RequestLogging(logger))
	r.Use(custommiddleware.ErrorHandler(logger))
	r.Use(cors.Default())

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"app":     cfg.AppName,
			"version": cfg.AppVersion,
			"env":     cfg.Environment,
		})
	})

	api := r.Group("/api/v1")

	app := &internal.App{
		Config:  cfg,
		DB:      database,
		Queries: db.New(database),
		Cache:   cacheService,
		Logger:  logger,
		Api:     api,
	}

	// Initialize scheduler if enabled
	if cfg.EnableScheduler {
		deps := &scheduler.Dependencies{
			Config:  cfg,
			DB:      database,
			Queries: app.Queries,
			Logger:  logger,
		}

		cronScheduler := scheduler.NewScheduler(deps)
		if err := cronScheduler.RegisterJobs(); err != nil {
			logger.Error(context.TODO(), "Failed to register scheduler jobs", err)
			log.Fatal("Failed to register scheduler jobs:", err)
		}

		cronScheduler.Start()
		logger.Info(context.TODO(), "Scheduler started in integrated mode")

		// Ensure graceful shutdown of scheduler
		defer cronScheduler.Stop()
	}

	// Register module routes
	cities.RegisterRoutes(app)

	// Swagger route - set host dynamically
	docs.SwaggerInfo.Host = cfg.AppURL
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Start server
	address := ":" + cfg.Port
	logger.Info(context.TODO(), "Server starting", "address", address)
	if err := r.Run(address); err != nil {
		logger.Error(context.TODO(), "Server failed to start", err, "address", address)
		log.Fatal(err)
	}
}
