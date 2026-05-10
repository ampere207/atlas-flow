package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"

	"atlasflow/backend/event-service/internal/handler"
	"atlasflow/backend/event-service/internal/publisher"
	"atlasflow/backend/event-service/internal/subscriber"
	"atlasflow/backend/shared/middleware"
)

func main() {
	// Load configuration
	natsURL := getEnv("NATS_URL", "nats://localhost:4222")
	port := getEnv("PORT", "8004")

	// Connect to NATS
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// Initialize publisher
	eventPublisher := publisher.NewNATSPublisher(nc)

	// Initialize handlers
	eventHandler := handler.NewEventHandler(eventPublisher)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := subscriber.StartSubscriptions(ctx, nc); err != nil {
		log.Fatalf("Failed to start event subscriptions: %v", err)
	}

	// Setup router
	router := gin.Default()

	// Middleware
	router.Use(middleware.LoggingMiddleware())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		middleware.RespondJSON(c.Writer, http.StatusOK, map[string]string{"status": "healthy"})
	})

	// Event routes
	router.POST("/events/publish", eventHandler.PublishEvent)
	router.GET("/events/status", eventHandler.GetStatus)

	// Start server
	fmt.Printf("Event Service running on port %s\n", port)
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
