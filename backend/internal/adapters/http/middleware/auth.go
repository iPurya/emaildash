package middleware

import (
	"context"
	"net/http"
	"strings"

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
			message := err.Error()
			if strings.Contains(message, "expired") || strings.Contains(message, "revoked") {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": message})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
			return
		}
		c.Set("session", session)
		c.Next()
	}
}

func RequirePageAuth(cookieName string, authService interface{ Authenticate(ctx context.Context, token string) (domain.Session, error) }) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(cookieName)
		if err != nil || token == "" {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		session, err := authService.Authenticate(c.Request.Context(), token)
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Set("session", session)
		c.Next()
	}
}
