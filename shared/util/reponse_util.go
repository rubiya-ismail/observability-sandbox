package util

import "github.com/gin-gonic/gin"

// Common response structures
type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// Common response helper functions
func SendErrorResponse(ctxt *gin.Context, statusCode int, message string) {
	ctxt.JSON(statusCode, ErrorResponse{Error: message})
}

// func SendSuccessResponse(ctxt *gin.Context, statusCode int, data interface{}) {
// 	cctxtJSON(statusCode, data)
// }

func SendJSONResponse(ctxt *gin.Context, statusCode int, data gin.H) {
	ctxt.JSON(statusCode, data)
}
