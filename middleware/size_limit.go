package middleware

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"io"
	"io/ioutil"
	"net/http"
)

const (
	MaxMessageContentSize = 8192 // 8KB, match WebSocket limit
)

func MessageSizeLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only apply to POST requests to /api/messages
		if c.Request.Method == "POST" && c.FullPath() == "/api/messages" {
			// Read the request body
			bodyBytes, err := ioutil.ReadAll(c.Request.Body)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Error reading request body"})
				c.Abort()
				return
			}

			// Restore the request body for future middleware/handlers
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			var message struct {
				Content string `json:"content"`
			}

			// Bind the JSON to check content size
			if err := c.ShouldBindJSON(&message); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
				c.Abort()
				return
			}

			// Check content size
			if len(message.Content) > MaxMessageContentSize {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{
					"error": "Message content exceeds maximum size limit",
				})
				c.Abort()
				return
			}

			// Reset the request body again for the next handler
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
		c.Next()
	}
}
