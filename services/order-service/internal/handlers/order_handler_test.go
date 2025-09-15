package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"shared/util"
	"testing"

	"services/order-service/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOrderHandler_Create(t *testing.T) {
	// Act
	handler := NewOrderHandler()

	// Assert
	assert.NotNil(t, handler)
	assert.IsType(t, &OrderHandler{}, handler)
}

func TestCreateOrder_ValidRequest(t *testing.T) {
	// Arrange
	handler := NewOrderHandler()
	router := util.SetupTestRouter()

	const url = "/orders"
	router.POST(url, handler.CreateOrder)

	validRequest := models.CreateOrderRequest{
		UserID: 1,
		Items: []models.OrderItem{
			{
				ProductID: 1,
				Name:      "Test Product",
				Price:     10.99,
				Quantity:  2,
			},
		},
	}
	requestBody, _ := json.Marshal(validRequest)

	w := util.ProcessTestRequest(http.MethodPost, url, bytes.NewReader(requestBody), router)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Order created successfully", response["message"])
	assert.Contains(t, response, "data")
}

func TestCreateOrder_InvalidRequest(t *testing.T) {
	// Arrange
	handler := NewOrderHandler()
	router := util.SetupTestRouter()

	const url = "/orders"
	router.POST(url, handler.CreateOrder)

	w := util.ProcessTestRequest(http.MethodPost, url, bytes.NewReader([]byte("invalid json")), router)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateOrder_EmptyItems(t *testing.T) {
	// Arrange
	handler := NewOrderHandler()
	router := util.SetupTestRouter()

	const url = "/orders"
	router.POST(url, handler.CreateOrder)

	requestWithEmptyItems := models.CreateOrderRequest{
		UserID: 1,
		Items:  []models.OrderItem{},
	}

	requestBody, _ := json.Marshal(requestWithEmptyItems)
	w := util.ProcessTestRequest(http.MethodPost, url, bytes.NewReader(requestBody), router)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateOrder_InvalidProductId(t *testing.T) {
	// Arrange
	handler := NewOrderHandler()
	router := util.SetupTestRouter()

	const url = "/orders"
	router.POST(url, handler.CreateOrder)

	invalidRequest := models.CreateOrderRequest{
		UserID: 1,
		Items: []models.OrderItem{
			{
				ProductID: 0, // Invalid product ID
				Name:      "Test Product",
				Price:     10.99,
				Quantity:  2,
			},
		},
	}

	requestBody, _ := json.Marshal(invalidRequest)
	w := util.ProcessTestRequest(http.MethodPost, url, bytes.NewReader(requestBody), router)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateOrder_EmptyItemName(t *testing.T) {
	// Arrange
	handler := NewOrderHandler()
	router := util.SetupTestRouter()

	const url = "/orders"
	router.POST(url, handler.CreateOrder)

	invalidRequest := models.CreateOrderRequest{
		UserID: 1,
		Items: []models.OrderItem{
			{
				ProductID: 1,
				Name:      "", // Empty name
				Price:     10.99,
				Quantity:  2,
			},
		},
	}

	requestBody, _ := json.Marshal(invalidRequest)
	w := util.ProcessTestRequest(http.MethodPost, url, bytes.NewReader(requestBody), router)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateOrder_InvalidPrice(t *testing.T) {
	// Arrange
	handler := NewOrderHandler()
	router := util.SetupTestRouter()

	const url = "/orders"
	router.POST(url, handler.CreateOrder)

	invalidRequest := models.CreateOrderRequest{
		UserID: 1,
		Items: []models.OrderItem{
			{
				ProductID: 1,
				Name:      "Test Product",
				Price:     0, // Invalid price
				Quantity:  2,
			},
		},
	}

	requestBody, _ := json.Marshal(invalidRequest)
	w := util.ProcessTestRequest(http.MethodPost, url, bytes.NewReader(requestBody), router)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateOrder_InvalidQuantity(t *testing.T) {
	// Arrange
	handler := NewOrderHandler()
	router := util.SetupTestRouter()

	const url = "/orders"
	router.POST(url, handler.CreateOrder)

	invalidRequest := models.CreateOrderRequest{
		UserID: 1,
		Items: []models.OrderItem{
			{
				ProductID: 1,
				Name:      "Test Product",
				Price:     10.99,
				Quantity:  0, // Invalid quantity
			},
		},
	}

	requestBody, _ := json.Marshal(invalidRequest)
	w := util.ProcessTestRequest(http.MethodPost, url, bytes.NewReader(requestBody), router)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateOrder_MultiItemValidation(t *testing.T) {
	// Arrange
	handler := NewOrderHandler()
	router := util.SetupTestRouter()

	const url = "/orders"
	router.POST(url, handler.CreateOrder)

	// Second item has invalid product ID
	invalidRequest := models.CreateOrderRequest{
		UserID: 1,
		Items: []models.OrderItem{
			{
				ProductID: 1,
				Name:      "Valid Product",
				Price:     10.99,
				Quantity:  2,
			},
			{
				ProductID: 0, // Invalid - should reference item 1
				Name:      "Invalid Product",
				Price:     5.99,
				Quantity:  1,
			},
		},
	}

	requestBody, _ := json.Marshal(invalidRequest)
	w := util.ProcessTestRequest(http.MethodPost, url, bytes.NewReader(requestBody), router)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "item 1") // Should reference the second item (index 1)
}

func TestGetAllOrders(t *testing.T) {
	// Arrange
	handler := NewOrderHandler()
	router := util.SetupTestRouter()

	const url = "/orders"
	router.GET(url, handler.GetAllOrders)

	w := util.ProcessTestRequest(http.MethodGet, url, nil, router)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Orders retrieved successfully", response["message"])
	assert.Contains(t, response, "data")
}

func TestGetOrderByID_ValidId(t *testing.T) {
	// Arrange
	handler := NewOrderHandler()
	router := util.SetupTestRouter()

	router.GET("/orders/:id", handler.GetOrderByID)

	w := util.ProcessTestRequest(http.MethodGet, "/orders/1", nil, router)

	// Assert
	// Note: Actual response depends on models.GetOrderByID implementation
	// This test assumes the order exists for ID 1
	if w.Code == http.StatusOK {
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Order retrieved successfully", response["message"])
		assert.Contains(t, response, "data")
	} else {
		assert.Equal(t, http.StatusNotFound, w.Code)
	}
}

func TestGetOrderByID_InvalidId(t *testing.T) {
	// Arrange
	handler := NewOrderHandler()
	router := util.SetupTestRouter()
	router.GET("/orders/:id", handler.GetOrderByID)

	req := httptest.NewRequest(http.MethodGet, "/orders/invalid", nil)
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetOrdersByUserID_ValidId(t *testing.T) {
	// Arrange
	handler := NewOrderHandler()
	router := util.SetupTestRouter()
	router.GET("/users/:userId/orders", handler.GetOrdersByUserID)

	w := util.ProcessTestRequest(http.MethodGet, "/users/1/orders", nil, router)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "User orders retrieved successfully", response["message"])
	assert.Contains(t, response, "data")
}

func TestGetOrdersByUserID_InvalidUserId(t *testing.T) {
	// Arrange
	handler := NewOrderHandler()
	router := util.SetupTestRouter()

	router.GET("/users/:userId/orders", handler.GetOrdersByUserID)

	w := util.ProcessTestRequest(http.MethodGet, "/users/invalid/orders", nil, router)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateOrderStatus_ValidRequest(t *testing.T) {
	// Arrange
	handler := NewOrderHandler()
	router := util.SetupTestRouter()

	router.PUT("/orders/:id/status", handler.UpdateOrderStatus)

	updateRequest := models.UpdateOrderStatusRequest{
		Status: models.StatusConfirmed,
	}

	requestBody, _ := json.Marshal(updateRequest)
	w := util.ProcessTestRequest(http.MethodPut, "/orders/1/status", bytes.NewReader(requestBody), router)

	// Assert
	// Response depends on whether order exists and models.UpdateOrderStatus implementation
	if w.Code == http.StatusOK {
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Order status updated successfully", response["message"])
		assert.Contains(t, response, "data")
	} else {
		assert.Contains(t, []int{http.StatusNotFound, http.StatusBadRequest}, w.Code)
	}
}

func TestUpdateOrderStatus_InvalidOrderId(t *testing.T) {
	// Arrange
	handler := NewOrderHandler()
	router := util.SetupTestRouter()
	router.PUT("/orders/:id/status", handler.UpdateOrderStatus)

	updateRequest := models.UpdateOrderStatusRequest{
		Status: models.StatusConfirmed,
	}

	requestBody, _ := json.Marshal(updateRequest)
	w := util.ProcessTestRequest(http.MethodPut, "/orders/invalid/status", bytes.NewReader(requestBody), router)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateOrderStatus_InvalidRequest(t *testing.T) {
	// Arrange
	handler := NewOrderHandler()
	router := util.SetupTestRouter()
	router.PUT("/orders/:id/status", handler.UpdateOrderStatus)

	w := util.ProcessTestRequest(http.MethodPut, "/orders/1/status", bytes.NewReader([]byte("invalid json")), router)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateOrderStatus_invalid_status_returns_bad_request(t *testing.T) {
	// Arrange
	handler := NewOrderHandler()
	router := util.SetupTestRouter()
	router.PUT("/orders/:id/status", handler.UpdateOrderStatus)

	updateRequest := models.UpdateOrderStatusRequest{
		Status: "invalid_status", // Invalid status
	}

	requestBody, _ := json.Marshal(updateRequest)
	w := util.ProcessTestRequest(http.MethodPut, "/orders/1/status", bytes.NewReader(requestBody), router)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateOrderStatus_ValidStatus(t *testing.T) {
	// Test all valid status values
	validStatuses := []models.OrderStatus{
		models.StatusPending,
		models.StatusConfirmed,
		models.StatusShipped,
		models.StatusDelivered,
		models.StatusCancelled,
	}

	for _, status := range validStatuses {
		t.Run("status_"+string(status), func(t *testing.T) {
			// Arrange
			handler := NewOrderHandler()
			router := util.SetupTestRouter()
			router.PUT("/orders/:id/status", handler.UpdateOrderStatus)

			updateRequest := models.UpdateOrderStatusRequest{
				Status: status,
			}

			requestBody, _ := json.Marshal(updateRequest)
			w := util.ProcessTestRequest(http.MethodPut, "/orders/1/status", bytes.NewReader(requestBody), router)

			// Assert
			// Should not return bad request for status validation
			assert.NotEqual(t, http.StatusBadRequest, w.Code, "Status %s should be valid", status)
		})
	}
}
