package routes

import (
	"services/order-service/internal/handlers"

	"github.com/gin-gonic/gin"
)

/*
POST /orders - Create order
GET /orders/{id} - Get order details
GET /orders?user_id={id} - Get user's orders
PUT /orders/{id}/status - Update order status
GET /orders/stats - Get order statistics
*/

func SetupRoutes(router *gin.Engine) {

	// API routes
	v1 := router.Group("/api/v1")
	{
		orderHandler := handlers.NewOrderHandler()

		// TODO: Add health and metrics endpoint

		// Order routes
		v1.GET("/orders", orderHandler.GetAllOrders)
		v1.GET("/orders/:id", orderHandler.GetOrderByID)
		v1.GET("/orders/?user_id={id}", orderHandler.GetOrdersByUserID)

		v1.POST("/orders", orderHandler.CreateOrder)
		v1.PUT("/orders/{id}/status", orderHandler.UpdateOrderStatus)
	}
}
