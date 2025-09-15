package handlers

import (
	"net/http"
	"shared/util"
	"time"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	startTime time.Time
}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{
		startTime: time.Now(),
	}
}

func (h *HealthHandler) HealthCheck(ctxt *gin.Context) {
	uptime := time.Since(h.startTime)

	util.SendJSONResponse(ctxt, http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "order-service",
		"version":   "1.0.0",
		"timestamp": time.Now(),
		"uptime":    uptime.String(),
	})
}

func (h *HealthHandler) ReadinessCheck(ctxt *gin.Context) {
	util.SendJSONResponse(ctxt, http.StatusOK, gin.H{
		"status":  "ready",
		"service": "order-service",
	})
}

func (h *HealthHandler) LivenessCheck(ctxt *gin.Context) {
	util.SendJSONResponse(ctxt, http.StatusOK, gin.H{
		"status":  "alive",
		"service": "order-service",
	})
}
