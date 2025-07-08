package main

import (
	"log"

	"myapp/config"
	_ "myapp/docs"
	"myapp/internal"
	"myapp/internal/cache"
	"myapp/internal/db"
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
	log.Printf("Starting %s v%s in %s mode", cfg.AppName, cfg.AppVersion, cfg.Environment)
	if cfg.Debug {
		log.Println("Debug mode enabled")
	}

	// DB
	database, err := db.NewConnection(cfg)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	// Redis
	redisClient, err := redis.NewConnection(cfg)
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}
	defer redisClient.Close()

	// Cache
	cacheService := cache.NewRedisCache(redisClient, cfg.AppName+":")

	// Echo
	e := echo.New()

	// Middleware
	if cfg.LogLevel == "info" || cfg.LogLevel == "debug" {
		e.Use(middleware.Logger())
	}
	e.Use(middleware.Recover())
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
		Api:    api,
	}

	// Register module routes
	users.RegisterRoutes(app)

	// Swagger route
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// Start server
	address := ":" + cfg.Port
	log.Printf("Server starting on %s", address)
	log.Fatal(e.Start(address))
}
