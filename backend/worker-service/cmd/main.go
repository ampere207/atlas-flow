package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"

	"atlasflow/backend/shared/auth"
	"atlasflow/backend/shared/db"
	"atlasflow/backend/shared/middleware"
	sharedruntime "atlasflow/backend/shared/runtime"
	"atlasflow/backend/worker-service/internal/handler"
	"atlasflow/backend/worker-service/internal/repository"
	workerruntime "atlasflow/backend/worker-service/internal/runtime"
	"atlasflow/backend/worker-service/internal/service"
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
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	workerID := getEnv("WORKER_INSTANCE_ID", uuid.New().String())
	port := getEnv("PORT", "8003")

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

	redisClient := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%s", redisHost, redisPort)})
	leaseManager := sharedruntime.NewLeaseManager(redisClient)

	// Initialize repositories
	workerRepo := repository.NewWorkerRepository(database)
	workerPublisher := workerruntime.NewNATSPublisher(natsConn)

	// Initialize task dispatcher (in-memory for now, can switch to Redis later)
	taskDispatcher := sharedruntime.NewInMemoryTaskDispatcher()

	// Create event publisher wrapper for shared runtime events
	eventPublisher := &NATSEventPublisher{nc: natsConn}

	runtimeLoop := workerruntime.NewWorkerRuntime(workerRepo, leaseManager, workerPublisher, taskDispatcher, eventPublisher, workerID)

	// Initialize services
	workerService := service.NewWorkerService(workerRepo)

	// Initialize handlers
	workerHandler := handler.NewWorkerHandler(workerService)

	// Setup router
	router := gin.Default()
	go runtimeLoop.Start(context.Background())

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
		protected.POST("/workers", workerHandler.RegisterWorker)
		protected.GET("/workers/:id", workerHandler.GetWorker)
		protected.GET("/workers", workerHandler.ListWorkers)
		protected.POST("/workers/:id/heartbeat", workerHandler.RecordHeartbeat)
	}

	// Start server
	fmt.Printf("Worker Service running on port %s\n", port)
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

// NATSEventPublisher wraps NATS connection for publishing runtime events
type NATSEventPublisher struct {
	nc *nats.Conn
}

// PublishEvent publishes an event to NATS
func (p *NATSEventPublisher) PublishEvent(ctx context.Context, event *sharedruntime.ExecutionEvent) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	subject := fmt.Sprintf("execution.%s.%s", event.EventType, event.WorkflowID)
	payload, _ := json.Marshal(event)
	return p.nc.Publish(subject, payload)
}

// PublishEventBatch publishes multiple events to NATS
func (p *NATSEventPublisher) PublishEventBatch(ctx context.Context, events []*sharedruntime.ExecutionEvent) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	for _, event := range events {
		if err := p.PublishEvent(ctx, event); err != nil {
			return err
		}
	}
	return nil
}
