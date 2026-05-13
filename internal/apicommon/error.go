package apicommon

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIErrorResponse is the V2 error envelope.
//
// @name APIErrorResponse
type APIErrorResponse struct {
	Error APIErrorBody `json:"error"` // 错误体
}

// APIErrorBody is the payload of an API error.
//
// @name APIErrorBody
type APIErrorBody struct {
	Code    string      `json:"code"`    // 错误码
	Message string      `json:"message"` // 错误描述
	Details interface{} `json:"details,omitempty" swaggertype:"object"`
}

// V2Error writes a structured V2 error response.
func V2Error(c *gin.Context, status int, code string, message string, details interface{}) {
	c.JSON(status, APIErrorResponse{
		Error: APIErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// V2BadRequest writes a 400 BAD_REQUEST V2 error.
func V2BadRequest(c *gin.Context, message string, details interface{}) {
	V2Error(c, http.StatusBadRequest, "BAD_REQUEST", message, details)
}
