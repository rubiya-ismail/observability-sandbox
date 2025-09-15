package models

import (
	"sync"
	"time"
)

type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusConfirmed OrderStatus = "confirmed"
	StatusShipped   OrderStatus = "shipped"
	StatusDelivered OrderStatus = "delivered"
	StatusCancelled OrderStatus = "cancelled"
)

type Order struct {
	ID         int         `json:"id"`
	UserID     int         `json:"user_id"`
	Items      []OrderItem `json:"items"`
	TotalPrice float64     `json:"total_price"`
	Status     OrderStatus `json:"status"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

type OrderItem struct {
	ProductID int     `json:"product_id"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Quantity  int     `json:"quantity"`
}

type CreateOrderRequest struct {
	UserID int         `json:"user_id" validate:"required"`
	Items  []OrderItem `json:"items" validate:"required,min=1"`
}

type UpdateOrderStatusRequest struct {
	Status OrderStatus `json:"status" validate:"required"`
}

// In-memory storage for MVP
var (
	orders = make(map[int]Order)
	nextID = 1
	mu     sync.Mutex
)

// Helpers

// Reset order info for testing
func ResetOrders() {
	mu.Lock()
	defer mu.Unlock()

	orders = make(map[int]Order)
	nextID = 1
}

// GetAllOrders returns all orders
func GetAllOrders() []Order {
	mu.Lock()
	defer mu.Unlock()

	orderList := make([]Order, 0, len(orders))
	for _, order := range orders {
		orderList = append(orderList, order)
	}
	return orderList
}

// GetOrdersByUserID returns orders for a specific user
func GetOrdersByUserID(userID int) []Order {
	mu.Lock()
	defer mu.Unlock()

	var userOrders []Order
	for _, order := range orders {
		if order.UserID == userID {
			userOrders = append(userOrders, order)
		}
	}
	return userOrders
}

// GetOrderByID returns a specific order
func GetOrderByID(id int) (Order, bool) {
	mu.Lock()
	defer mu.Unlock()

	order, exists := orders[id]
	return order, exists
}

// CreateOrder adds a new order and returns it
func CreateOrder(userID int, items []OrderItem) Order {
	mu.Lock()
	defer mu.Unlock()

	// Calculate total price
	var totalPrice float64
	for _, item := range items {
		totalPrice += item.Price * float64(item.Quantity)
	}

	now := time.Now()
	order := Order{
		ID:         nextID,
		UserID:     userID,
		Items:      items,
		TotalPrice: totalPrice,
		Status:     StatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	orders[nextID] = order
	nextID++
	return order
}

// UpdateOrderStatus updates the status of an order
func UpdateOrderStatus(id int, status OrderStatus) (Order, bool) {
	mu.Lock()
	defer mu.Unlock()

	order, exists := orders[id]
	if !exists {
		return Order{}, false
	}

	order.Status = status
	order.UpdatedAt = time.Now()
	orders[id] = order

	return order, true
}
