package handlers

import (
	"net/http"
	"strconv"

	"services/order-service/internal/models"
	"shared/util"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct{}

func NewOrderHandler() *OrderHandler {
	return &OrderHandler{}
}

// CreateOrder creates a new order
func (h *OrderHandler) CreateOrder(ctxt *gin.Context) {
	var req models.CreateOrderRequest
	if err := ctxt.ShouldBindJSON(&req); err != nil {
		util.SendErrorResponse(ctxt, http.StatusBadRequest, "Invalid request data")
		return
	}

	// Validate that items are provided
	if len(req.Items) == 0 {
		util.SendErrorResponse(ctxt, http.StatusBadRequest, "Order must contain at least one item")
		return
	}

	// Validate each item
	for i, item := range req.Items {
		if item.ProductID <= 0 {
			util.SendErrorResponse(ctxt, http.StatusBadRequest, "Invalid product ID in item "+strconv.Itoa(i))
			return
		}
		if item.Name == "" {
			util.SendErrorResponse(ctxt, http.StatusBadRequest, "Item name is required for item "+strconv.Itoa(i))
			return
		}
		if item.Price <= 0 {
			util.SendErrorResponse(ctxt, http.StatusBadRequest, "Invalid item price for item "+strconv.Itoa(i))
			return
		}
		if item.Quantity <= 0 {
			util.SendErrorResponse(ctxt, http.StatusBadRequest, "Invalid item quantity for item "+strconv.Itoa(i))
			return
		}
	}

	order := models.CreateOrder(req.UserID, req.Items)
	util.SendJSONResponse(ctxt, http.StatusCreated, gin.H{
		"message": "Order created successfully",
		"data":    order,
	})
}

// GetAllOrders returns all orders
func (h *OrderHandler) GetAllOrders(ctxt *gin.Context) {
	orders := models.GetAllOrders()
	util.SendJSONResponse(ctxt, http.StatusOK, gin.H{
		"message": "Orders retrieved successfully",
		"data":    orders,
	})
}

// GetOrderByID returns a specific order
func (h *OrderHandler) GetOrderByID(ctxt *gin.Context) {
	idStr := ctxt.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		util.SendErrorResponse(ctxt, http.StatusBadRequest, "Invalid order ID")
		return
	}

	order, exists := models.GetOrderByID(id)
	if !exists {
		util.SendErrorResponse(ctxt, http.StatusNotFound, "Order not found")
		return
	}

	util.SendJSONResponse(ctxt, http.StatusOK, gin.H{
		"message": "Order retrieved successfully",
		"data":    order,
	})
}

// GetOrdersByUserID returns orders for a specific user
func (h *OrderHandler) GetOrdersByUserID(c *gin.Context) {
	userIDStr := c.Param("userId")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		util.SendErrorResponse(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	orders := models.GetOrdersByUserID(userID)
	util.SendJSONResponse(c, http.StatusOK, gin.H{
		"message": "User orders retrieved successfully",
		"data":    orders,
	})
}

// UpdateOrderStatus updates the status of an order
func (h *OrderHandler) UpdateOrderStatus(ctxt *gin.Context) {
	idStr := ctxt.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		util.SendErrorResponse(ctxt, http.StatusBadRequest, "Invalid order ID")
		return
	}

	var req models.UpdateOrderStatusRequest
	if err := ctxt.ShouldBindJSON(&req); err != nil {
		util.SendErrorResponse(ctxt, http.StatusBadRequest, "Invalid request data")
		return
	}

	// Validate status
	validStatuses := []models.OrderStatus{
		models.StatusPending, models.StatusConfirmed, models.StatusShipped,
		models.StatusDelivered, models.StatusCancelled,
	}

	isValidStatus := false
	for _, status := range validStatuses {
		if req.Status == status {
			isValidStatus = true
			break
		}
	}

	if !isValidStatus {
		util.SendErrorResponse(ctxt, http.StatusBadRequest, "Invalid order status: "+string(req.Status))
		return
	}

	order, exists := models.UpdateOrderStatus(id, req.Status)
	if !exists {
		util.SendErrorResponse(ctxt, http.StatusNotFound, "Order not found")
		return
	}

	util.SendJSONResponse(ctxt, http.StatusOK, gin.H{
		"message": "Order status updated successfully",
		"data":    order,
	})
}
