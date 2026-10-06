package api

import "github.com/gin-gonic/gin"

// JSONResponse is the envelope every /api/v1 JSON response is written in,
// success and error alike. Data is null on an error and on a success with
// nothing to return; the HTTP status still carries the outcome.
type JSONResponse[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func respondSuccess[T any](c *gin.Context, status int, message string, data T) {
	c.JSON(status, JSONResponse[T]{Success: true, Message: message, Data: data})
}

// respondError aborts the chain, so middleware can use it as well as handlers.
// message is a flat human-readable string, never a per-field error list.
func respondError(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, JSONResponse[any]{Success: false, Message: message})
}
