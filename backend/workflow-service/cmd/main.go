package main

import (
	"context"
	"database/sql"
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
	sharedruntime "atlasflow/backend/shared/runtime"
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

	if err := ensureWorkflowSchema(database); err != nil {
		log.Fatalf("Failed to ensure workflow schema: %v", err)
	}

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

	// Initialize event bus (NATS-backed for distributed visibility)
	eventBus := sharedruntime.NewNATSEventPublisher(natsConn)

	// Initialize worker connection manager (tracks connected workers via NATS heartbeats)
	workerConnMgr := sharedruntime.NewWorkerConnectionManager(natsConn, eventBus)
	defer workerConnMgr.Stop()

	// Initialize NATS orchestrator (sends tasks to real workers)
	natsOrchestrator := workflowruntime.NewNATSOrchestrator(workflowRepo, natsConn, workerConnMgr, eventBus)

	// Start orchestration loop in background
	go func() {
		if err := natsOrchestrator.Start(context.Background()); err != nil {
			log.Printf("Orchestrator error: %v", err)
		}
	}()

	// Keep old orchestrator for backward compatibility
	orchestrator := workflowruntime.NewOrchestrator(workflowRepo, workflowPublisher)

	// Initialize services
	workflowService := service.NewWorkflowService(workflowRepo, orchestrator)

	// Initialize handlers
	workflowHandler := handler.NewWorkflowHandler(workflowService, workerConnMgr, natsConn)

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
		protected.GET("/workers", workflowHandler.GetWorkers)
		protected.GET("/cluster/metrics", workflowHandler.GetClusterMetrics)
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

func ensureWorkflowSchema(database *sql.DB) error {
	_, err := database.Exec(`
		ALTER TABLE workflows
		ADD COLUMN IF NOT EXISTS definition TEXT NOT NULL DEFAULT ''
	`)
	return err
}
