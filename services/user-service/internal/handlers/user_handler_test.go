package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"services/user-service/internal/models"
	"shared/util"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helpers
func setupTestContext(method, url string, reqBytesBuf *bytes.Buffer) (*gin.Context, *httptest.ResponseRecorder) {

	// TODO: Ensure there is no side-effect
	// Workaround: nil is allowed but not nil bytes.Buffer
	if reqBytesBuf == nil {
		reqBytesBuf = bytes.NewBuffer(nil)
	}

	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctxt, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest(method, url, reqBytesBuf)
	req.Header.Set("Content-Type", "application/json")
	ctxt.Request = req

	return ctxt, w
}

func reqToBytesBuffer(t *testing.T, reqBody map[string]string) *bytes.Buffer {
	jsonBytes, err := json.Marshal(reqBody)
	assert.Nil(t, err)
	return bytes.NewBuffer(jsonBytes)
}

func TestCreateUser_ValidRequest(t *testing.T) {
	// Reset models state
	models.ResetUsers()

	// Set up and configure test router
	router := util.SetupTestRouter()

	const url = "/users"
	router.POST(url, CreateUser)

	// Process test request
	reqBytes := util.JsonDictToReader(t, map[string]string{
		"name":  "John Doe",
		"email": "john@example.com",
	})
	w := util.ProcessTestRequest(http.MethodPost, url, reqBytes, router)

	// Assert response
	assert.Equal(t, http.StatusCreated, w.Code)

	// Parse response JSON
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Validate response structure
	user := response["users"].(map[string]interface{})
	assert.Equal(t, "John Doe", user["name"])
	assert.Equal(t, "john@example.com", user["email"])
	assert.Equal(t, float64(1), user["id"]) // JSON numbers are float64
}

func TestCreateUser_MissingName(t *testing.T) {
	models.ResetUsers()

	// Create JSON request body
	reqBytesBuf := reqToBytesBuffer(t, map[string]string{
		"email": "john@example.com",
	})
	ctxt, w := setupTestContext(http.MethodPost, "/users", reqBytesBuf)

	CreateUser(ctxt)

	// Should return bad request
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "required")
}

func TestCreateUser_MissingEmail(t *testing.T) {
	models.ResetUsers()

	// Create JSON request body
	reqBytesBuf := reqToBytesBuffer(t, map[string]string{
		"name": "John Doe",
	})
	ctxt, w := setupTestContext(http.MethodPost, "/users", reqBytesBuf)

	CreateUser(ctxt)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "required")
}

func TestCreateUser_InvalidJSON(t *testing.T) {
	models.ResetUsers()

	// Invalid JSON - missing closing brace
	invalidJSON := `{"name":"John","email":"john@test.com"`
	ctxt, w := setupTestContext(http.MethodPost, "/users", bytes.NewBuffer([]byte(invalidJSON)))

	CreateUser(ctxt)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetUsers_EmptyList(t *testing.T) {
	models.ResetUsers()

	ctxt, w := setupTestContext(http.MethodGet, "/users", nil)

	GetUsers(ctxt)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	users := response["users"].([]interface{})
	count := response["count"].(float64)

	assert.Equal(t, 0, len(users))
	assert.Equal(t, float64(0), count)
}

func TestGetUsers_WithData(t *testing.T) {
	models.ResetUsers()

	// Add some test data
	models.CreateUser("John", "john@test.com")
	models.CreateUser("Jane", "jane@test.com")

	ctxt, w := setupTestContext(http.MethodGet, "/users", nil)

	GetUsers(ctxt)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	users := response["users"].([]interface{})
	count := response["count"].(float64)

	assert.Equal(t, 2, len(users))
	assert.Equal(t, float64(2), count)
}

func TestHealthCheck(t *testing.T) {

	// Set up and configure test router
	router := util.SetupTestRouter()

	const url = "/health"
	router.GET(url, HealthCheck)

	w := util.ProcessTestRequest(http.MethodGet, url, nil, router)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "user-service", response["service"])
}
