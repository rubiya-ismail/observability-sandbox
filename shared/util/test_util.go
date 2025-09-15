package util

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func SetupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func ProcessTestRequest(method, url string, requestBody io.Reader, router *gin.Engine) *httptest.ResponseRecorder {

	req := httptest.NewRequest(method, url, requestBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	return w
}

func JsonDictToReader(t *testing.T, jsonDict map[string]string) *bytes.Reader {

	jsonBytes, err := json.Marshal(jsonDict)
	assert.Nil(t, err)
	return bytes.NewReader(jsonBytes)
}
