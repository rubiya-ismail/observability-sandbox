package handlers

import (
	"net/http"

	"services/user-service/internal/models"

	"shared/util"

	"github.com/gin-gonic/gin"
)

// GetUsers handles GET /users
func GetUsers(ctxt *gin.Context) {
	users := models.GetAllUsers()

	util.SendJSONResponse(ctxt, http.StatusOK, gin.H{
		"users": users,
		"count": len(users),
	})
}

// CreateUser handles POST /users
func CreateUser(ctxt *gin.Context) {
	var request struct {
		Name  string `json:"name" binding:"required"`
		Email string `json:"email" binding:"required"`
	}
	if err := ctxt.ShouldBindJSON(&request); err != nil {
		util.SendErrorResponse(ctxt, http.StatusBadRequest, err.Error())
		return
	}
	user := models.CreateUser(request.Name, request.Email)
	util.SendJSONResponse(ctxt, http.StatusCreated, gin.H{"users": user})
}

// HealthCheck handles GET /health
func HealthCheck(ctxt *gin.Context) {
	util.SendJSONResponse(ctxt, http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "user-service",
	})
}
