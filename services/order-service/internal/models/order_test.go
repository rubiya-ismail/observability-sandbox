package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOrder_Create(t *testing.T) {
	ResetOrders()

	items := []OrderItem{
		{ProductID: 1, Name: "Laptop", Price: 999.99, Quantity: 1},
		{ProductID: 2, Name: "Mouse", Price: 29.99, Quantity: 2},
	}
	userID := 123
	order := CreateOrder(userID, items)

	expectedTotal := 1059.97
	now := time.Now()

	assert.Equal(t, 1, order.ID, "Order ID should start at 1")
	assert.Equal(t, userID, order.UserID, "User ID should match input")
	assert.Len(t, order.Items, 2, "Order should contain 2 items")
	assert.InDelta(t, expectedTotal, order.TotalPrice, 0.01, "Total price should be calculated correctly")
	assert.Equal(t, StatusPending, order.Status, "Initial status should be pending")
	assert.False(t, order.CreatedAt.IsZero(), "CreatedAt should be set")
	assert.False(t, order.UpdatedAt.IsZero(), "UpdatedAt should be set")
	assert.WithinDuration(t, now, order.CreatedAt, time.Minute, "CreatedAt should be recent")
}

func TestOrder_CreateMultiple(t *testing.T) {
	ResetOrders()

	items1 := []OrderItem{{ProductID: 1, Name: "Item1", Price: 10.0, Quantity: 1}}
	items2 := []OrderItem{{ProductID: 2, Name: "Item2", Price: 20.0, Quantity: 1}}

	order1 := CreateOrder(1, items1)
	order2 := CreateOrder(2, items2)

	assert.Equal(t, 1, order1.ID, "First order ID should be 1")
	assert.Equal(t, 2, order2.ID, "Second order ID should be 2")

	allOrders := GetAllOrders()
	assert.Len(t, allOrders, 2, "There should be 2 orders in storage")
}

func TestOrder_GetAll(t *testing.T) {
	ResetOrders()

	// Test empty state
	orders := GetAllOrders()
	assert.Len(t, orders, 0, "Expected 0 orders initially")

	// Add some orders
	items := []OrderItem{{ProductID: 1, Name: "Test", Price: 10.0, Quantity: 1}}
	CreateOrder(1, items)
	CreateOrder(2, items)
	CreateOrder(3, items)

	// Test with orders
	orders = GetAllOrders()
	assert.Len(t, orders, 3, "Expected 3 orders after creation")

	// Verify order contents are correct
	orderIDs := make(map[int]bool)
	for _, order := range orders {
		orderIDs[order.ID] = true
	}

	for i := 1; i <= 3; i++ {
		assert.True(t, orderIDs[i], "Expected to find order with ID %d", i)
	}
}

func TestOrder_GetByID(t *testing.T) {
	ResetOrders()

	items := []OrderItem{{ProductID: 1, Name: "Test Product", Price: 15.50, Quantity: 2}}
	originalOrder := CreateOrder(42, items)

	order, exists := GetOrderByID(1)
	assert.True(t, exists, "Expected to find order with ID 1")
	assert.Equal(t, originalOrder.ID, order.ID, "Order ID mismatch")
	assert.Equal(t, originalOrder.UserID, order.UserID, "User ID mismatch")
	assert.Equal(t, originalOrder.TotalPrice, order.TotalPrice, "Total price mismatch")

	_, exists = GetOrderByID(999)
	assert.False(t, exists, "Expected not to find order with ID 999")
}

func TestOrder_GetByUserID(t *testing.T) {
	ResetOrders()

	items := []OrderItem{{ProductID: 1, Name: "Test", Price: 10.0, Quantity: 1}}
	CreateOrder(1, items)
	CreateOrder(2, items)
	CreateOrder(1, items)
	CreateOrder(3, items)
	CreateOrder(1, items)

	user1Orders := GetOrdersByUserID(1)
	assert.Len(t, user1Orders, 3, "Expected 3 orders for user 1")
	for _, order := range user1Orders {
		assert.Equal(t, 1, order.UserID, "Expected all orders to belong to user 1")
	}

	user2Orders := GetOrdersByUserID(2)
	assert.Len(t, user2Orders, 1, "Expected 1 order for user 2")

	user999Orders := GetOrdersByUserID(999)
	assert.Len(t, user999Orders, 0, "Expected 0 orders for user 999")
}

func TestOrder_UpdateStatus(t *testing.T) {
	ResetOrders()

	items := []OrderItem{{ProductID: 1, Name: "Test", Price: 10.0, Quantity: 1}}
	originalOrder := CreateOrder(1, items)

	assert.Equal(t, StatusPending, originalOrder.Status, "Initial status should be pending")

	updatedOrder, exists := UpdateOrderStatus(1, StatusConfirmed)
	assert.True(t, exists, "Expected to find and update order with ID 1")
	assert.Equal(t, StatusConfirmed, updatedOrder.Status, "Updated status mismatch")
	assert.True(t, updatedOrder.UpdatedAt.After(originalOrder.UpdatedAt), "UpdatedAt should be modified")

	storedOrder, _ := GetOrderByID(1)
	assert.Equal(t, StatusConfirmed, storedOrder.Status, "Stored order status mismatch")

	_, exists = UpdateOrderStatus(999, StatusShipped)
	assert.False(t, exists, "Expected not to find order with ID 999 for update")
}

func TestOrder_StatusProgression(t *testing.T) {
	ResetOrders()

	items := []OrderItem{{ProductID: 1, Name: "Test", Price: 10.0, Quantity: 1}}
	CreateOrder(1, items)

	statuses := []OrderStatus{StatusPending, StatusConfirmed, StatusShipped, StatusDelivered}
	for i, status := range statuses {
		if i == 0 {
			continue
		}
		updatedOrder, exists := UpdateOrderStatus(1, status)
		assert.True(t, exists, "Failed to update order to status %s", status)
		assert.Equal(t, status, updatedOrder.Status, "Expected status %s, got %s", status, updatedOrder.Status)
	}

	cancelledOrder, exists := UpdateOrderStatus(1, StatusCancelled)
	assert.True(t, exists, "Failed to update order to cancelled status")
	assert.Equal(t, StatusCancelled, cancelledOrder.Status, "Expected status %s, got %s", StatusCancelled, cancelledOrder.Status)
}

func TestOrder_TotalPriceCalculation(t *testing.T) {
	ResetOrders()

	testCases := []struct {
		name          string
		items         []OrderItem
		expectedTotal float64
	}{
		{"Single item", []OrderItem{{ProductID: 1, Name: "Item1", Price: 25.50, Quantity: 1}}, 25.50},
		{"Multiple quantities", []OrderItem{{ProductID: 1, Name: "Item1", Price: 10.00, Quantity: 5}}, 50.00},
		{"Multiple items", []OrderItem{
			{ProductID: 1, Name: "Item1", Price: 15.75, Quantity: 2},
			{ProductID: 2, Name: "Item2", Price: 8.25, Quantity: 3},
			{ProductID: 3, Name: "Item3", Price: 100.00, Quantity: 1},
		}, 156.25},
		{"Decimal precision", []OrderItem{
			{ProductID: 1, Name: "Item1", Price: 9.99, Quantity: 3},
			{ProductID: 2, Name: "Item2", Price: 0.01, Quantity: 1},
		}, 29.98},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			order := CreateOrder(1, tc.items)
			assert.Equal(t, tc.expectedTotal, order.TotalPrice, "Total price mismatch")
		})
	}
}

func TestOrder_ConcurrentCreation(t *testing.T) {
	ResetOrders()

	items := []OrderItem{{ProductID: 1, Name: "Test", Price: 10.0, Quantity: 1}}

	const numGoroutines = 10
	done := make(chan bool)

	for i := range numGoroutines {
		go func(userID int) {
			CreateOrder(userID, items)
			done <- true
		}(i + 1)
	}
	for range numGoroutines {
		<-done
	}

	allOrders := GetAllOrders()
	assert.Len(t, allOrders, numGoroutines, "Expected %d orders", numGoroutines)

	idMap := make(map[int]bool)
	for _, order := range allOrders {
		assert.False(t, idMap[order.ID], "Found duplicate order ID: %d", order.ID)
		idMap[order.ID] = true
	}
}
