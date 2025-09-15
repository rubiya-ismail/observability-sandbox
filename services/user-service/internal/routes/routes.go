package routes

import (
	"services/user-service/internal/handlers"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

/*
GET /users - List all users
GET /users/{id} - Get user by ID
POST /users - Create new user
GET /health - Health check
GET /metrics - Prometheus metrics endpoint
*/

func SetupRoutes(router *gin.Engine) {

	// API routes
	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", handlers.HealthCheck)

		v1.GET("/metrics", gin.WrapH(promhttp.Handler()))

		// TODO: Implmement other APIs
		// User routes
		v1.GET("/users", handlers.GetUsers)
		//v1.GET("/users/:id", getUser)
		v1.POST("/users", handlers.CreateUser)
		//v1.PUT("/users/:id", updateUser)
		//v1.DELETE("/users/:id", deleteUser)
	}

	// TODO: v2....
}
