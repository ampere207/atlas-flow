package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"atlasflow/backend/gateway-service/internal/handler"
	"atlasflow/backend/gateway-service/internal/proxy"
	"atlasflow/backend/shared/auth"
	"atlasflow/backend/shared/middleware"
)

func main() {
	// Load configuration
	authServiceURL := getEnv("AUTH_SERVICE_URL", "http://localhost:8001")
	workflowServiceURL := getEnv("WORKFLOW_SERVICE_URL", "http://localhost:8002")
	workerServiceURL := getEnv("WORKER_SERVICE_URL", "http://localhost:8003")
	eventServiceURL := getEnv("EVENT_SERVICE_URL", "http://localhost:8004")
	jwtSecret := getEnv("JWT_SECRET", "your-secret-key")
	port := getEnv("PORT", "8000")

	// Initialize JWT manager
	jwtManager := auth.NewJWTManager(jwtSecret, 15*time.Minute)

	// Initialize proxy
	gatewayProxy := proxy.NewGatewayProxy(
		authServiceURL,
		workflowServiceURL,
		workerServiceURL,
		eventServiceURL,
	)

	// Initialize handlers
	gatewayHandler := handler.NewGatewayHandler(gatewayProxy)

	// Setup router
	router := gin.Default()

	// Middleware
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.LoggingMiddleware())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		middleware.RespondJSON(c.Writer, http.StatusOK, map[string]string{"status": "healthy"})
	})

	// Public routes
	router.POST("/auth/signup", gatewayHandler.SignUp)
	router.POST("/auth/login", gatewayHandler.Login)
	router.POST("/auth/refresh", gatewayHandler.RefreshToken)

	// Protected routes
	protected := router.Group("")
	protected.Use(middleware.AuthMiddleware(jwtManager))
	{
		// Workflow routes
		protected.POST("/workflows", gatewayHandler.CreateWorkflow)
		protected.POST("/workflows/:id/execute", gatewayHandler.ExecuteWorkflow)
		protected.POST("/workflows/:id/cancel", gatewayHandler.CancelWorkflow)
		protected.GET("/workflows/:id", gatewayHandler.GetWorkflow)
		protected.GET("/workflows", gatewayHandler.ListWorkflows)
		protected.GET("/workflows/:id/tasks", gatewayHandler.ListWorkflowTasks)
		protected.GET("/workflows/:id/history", gatewayHandler.ListWorkflowHistory)
		protected.GET("/workflows/:id/status", gatewayHandler.GetWorkflowExecutionStatus)
		protected.GET("/workflows/:id/stream", gatewayHandler.StreamWorkflowExecution)
		protected.PUT("/workflows/:id/status", gatewayHandler.UpdateWorkflowStatus)

		// Worker routes
		protected.POST("/workers", gatewayHandler.RegisterWorker)
		protected.GET("/workers/:id", gatewayHandler.GetWorker)
		protected.GET("/workers", gatewayHandler.ListWorkers)
		protected.POST("/workers/:id/heartbeat", gatewayHandler.RecordHeartbeat)
	}

	// Start server
	fmt.Printf("API Gateway running on port %s\n", port)
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
