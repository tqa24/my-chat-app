package middleware

import (
	"my-chat-app/services"
	"my-chat-app/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func JWTAuthMiddleware(jwtService services.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// For WebSocket connections, get token from query parameter
		var tokenString string
		if c.Request.URL.Path == "/api/ws" {
			tokenString = c.Query("token")
		} else {
			// Regular API endpoints get token from Authorization header
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				utils.RespondWithError(c, http.StatusUnauthorized, "Authorization header is required")
				c.Abort()
				return
			}
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				utils.RespondWithError(c, http.StatusUnauthorized, "Authorization header format must be Bearer {token}")
				c.Abort()
				return
			}
			tokenString = parts[1]
		}

		// Validate the token
		token, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			utils.RespondWithError(c, http.StatusUnauthorized, "Invalid or expired token")
			c.Abort()
			return
		}

		// Extract user ID from token
		userID, err := jwtService.GetUserIDFromToken(token)
		if err != nil {
			utils.RespondWithError(c, http.StatusUnauthorized, "Failed to extract user information")
			c.Abort()
			return
		}

		// Store user ID in context for later use
		c.Set("userID", userID)
		c.Next()
	}
}
