package main

import (
	"context"
	"log"

	"myapp/config"
	_ "myapp/docs"
	"myapp/internal"
	"myapp/internal/cache"
	"myapp/internal/db"
	"myapp/internal/logger"
	custommiddleware "myapp/internal/middleware"
	"myapp/internal/redis"
	"myapp/internal/users"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// @title           MyApp API
// @version         1.0
// @description     A simple web API built with Go, Echo, and sqlc
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /

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

	// Echo
	e := echo.New()

	// Custom error handler
	e.HTTPErrorHandler = custommiddleware.ErrorHandler(logger)

	// Middleware
	e.Use(custommiddleware.RequestID(logger))
	e.Use(custommiddleware.Recovery(logger))
	e.Use(custommiddleware.RequestLogging(logger))
	e.Use(middleware.CORS())

	// Health check endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]interface{}{
			"status":  "healthy",
			"app":     cfg.AppName,
			"version": cfg.AppVersion,
			"env":     cfg.Environment,
		})
	})

	api := e.Group("/api/v1")

	app := &internal.App{
		Config: cfg,
		DB:     database,
		Cache:  cacheService,
		Logger: logger,
		Api:    api,
	}

	// Register module routes
	users.RegisterRoutes(app)

	// Swagger route
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// Start server
	address := ":" + cfg.Port
	logger.Info(context.TODO(), "Server starting", "address", address)
	if err := e.Start(address); err != nil {
		logger.Error(context.TODO(), "Server failed to start", err, "address", address)
		log.Fatal(err)
	}
}
