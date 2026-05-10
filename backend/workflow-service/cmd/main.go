package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"

	"atlasflow/backend/shared/auth"
	"atlasflow/backend/shared/db"
	"atlasflow/backend/shared/middleware"
	"atlasflow/backend/workflow-service/internal/handler"
	"atlasflow/backend/workflow-service/internal/repository"
	workflowruntime "atlasflow/backend/workflow-service/internal/runtime"
	"atlasflow/backend/workflow-service/internal/service"
)

func main() {
	// Load configuration
	dbConfig := db.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     5432,
		User:     getEnv("DB_USER", "atlasflow"),
		Password: getEnv("DB_PASSWORD", "atlasflow_dev"),
		Database: getEnv("DB_NAME", "atlasflow"),
		SSLMode:  getEnv("DB_SSL_MODE", "disable"),
	}

	jwtSecret := getEnv("JWT_SECRET", "your-secret-key")
	natsURL := getEnv("NATS_URL", "nats://localhost:4222")
	port := getEnv("PORT", "8002")

	// Connect to database
	database, err := db.NewConnection(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Initialize JWT manager
	jwtManager := auth.NewJWTManager(jwtSecret, 15*time.Minute)

	natsConn, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer natsConn.Close()

	// Initialize repositories
	workflowRepo := repository.NewWorkflowRepository(database)
	workflowPublisher := workflowruntime.NewNATSPublisher(natsConn)
	orchestrator := workflowruntime.NewOrchestrator(workflowRepo, workflowPublisher)

	// Initialize services
	workflowService := service.NewWorkflowService(workflowRepo, orchestrator)

	// Initialize handlers
	workflowHandler := handler.NewWorkflowHandler(workflowService)

	// Setup router
	router := gin.Default()

	// Middleware
	router.Use(middleware.LoggingMiddleware())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		middleware.RespondJSON(c.Writer, http.StatusOK, map[string]string{"status": "healthy"})
	})

	// Protected routes
	protected := router.Group("")
	protected.Use(middleware.AuthMiddleware(jwtManager))
	{
		protected.POST("/workflows", workflowHandler.CreateWorkflow)
		protected.POST("/workflows/:id/execute", workflowHandler.ExecuteWorkflow)
		protected.POST("/workflows/:id/cancel", workflowHandler.CancelWorkflow)
		protected.GET("/workflows/:id", workflowHandler.GetWorkflow)
		protected.GET("/workflows", workflowHandler.ListWorkflows)
		protected.GET("/workflows/:id/tasks", workflowHandler.ListWorkflowTasks)
		protected.GET("/workflows/:id/history", workflowHandler.ListWorkflowHistory)
		protected.GET("/workflows/:id/status", workflowHandler.GetWorkflowExecutionStatus)
		protected.PUT("/workflows/:id/status", workflowHandler.UpdateWorkflowStatus)
	}

	// Start server
	fmt.Printf("Workflow Service running on port %s\n", port)
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
