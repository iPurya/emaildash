package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/purya/emaildash/backend/internal/domain"
)

func RequireAuth(cookieName string, authService interface{ Authenticate(ctx context.Context, token string) (domain.Session, error) }) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(cookieName)
		if err != nil || token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		session, err := authService.Authenticate(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
			return
		}
		c.Set("session", session)
		c.Next()
	}
}

func RequireCSRF(headerName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		value := c.GetHeader(headerName)
		sessionValue, exists := c.Get("session")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session required"})
			return
		}
		session := sessionValue.(domain.Session)
		if value == "" || value != session.CSRFToken {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "csrf token mismatch"})
			return
		}
		c.Next()
	}
}
