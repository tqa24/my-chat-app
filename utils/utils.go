package utils

import "github.com/gin-gonic/gin"

// RespondWithError sends a JSON error response
func RespondWithError(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{"error": message})
}
