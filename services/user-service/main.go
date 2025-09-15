package main

import (
	"log"
	"services/user-service/internal/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	routes.SetupRoutes(router)

	// TODO: Use product quality logger, like zap or logrus (already in telemetry)
	log.Println("Starting user service on :8082")
	if err := router.Run(":8082"); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
