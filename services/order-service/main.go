package main

import (
	"log"
	"services/order-service/internal/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	routes.SetupRoutes(router)

	// TODO: Use product quality logger, like zap or logrus (already in telemetry)
	// TODO: Configure port
	log.Println("Starting order service on :8083")
	if err := router.Run(":8083"); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
