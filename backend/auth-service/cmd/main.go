package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"atlasflow/backend/auth-service/internal/handler"
	"atlasflow/backend/auth-service/internal/repository"
	"atlasflow/backend/auth-service/internal/service"
	"atlasflow/backend/shared/auth"
	"atlasflow/backend/shared/db"
	"atlasflow/backend/shared/middleware"
)

func main() {
	// Load configuration from environment
	dbConfig := db.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     5432,
		User:     getEnv("DB_USER", "atlasflow"),
		Password: getEnv("DB_PASSWORD", "atlasflow_dev"),
		Database: getEnv("DB_NAME", "atlasflow"),
		SSLMode:  getEnv("DB_SSL_MODE", "disable"),
	}

	jwtSecret := getEnv("JWT_SECRET", "your-secret-key")
	port := getEnv("PORT", "8001")

	// Connect to database
	database, err := db.NewConnection(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Initialize JWT manager
	jwtManager := auth.NewJWTManager(jwtSecret, 15*time.Minute)

	// Initialize repositories
	userRepo := repository.NewUserRepository(database)

	// Initialize services
	authService := service.NewAuthService(userRepo, jwtManager)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)

	// Setup router
	router := gin.Default()

	// Middleware
	router.Use(middleware.LoggingMiddleware())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		middleware.RespondJSON(c.Writer, http.StatusOK, map[string]string{"status": "healthy"})
	})

	// Auth routes
	router.POST("/auth/signup", authHandler.SignUp)
	router.POST("/auth/login", authHandler.Login)
	router.POST("/auth/refresh", authHandler.RefreshToken)

	// Start server
	fmt.Printf("Auth Service running on port %s\n", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}
