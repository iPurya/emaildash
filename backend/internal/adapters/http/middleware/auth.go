package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/purya/emaildash/backend/internal/domain"
)

func RequireAuth(cookieName string, authService interface {
	Authenticate(ctx context.Context, token string) (domain.Session, error)
	AuthenticateAPIKey(ctx context.Context, apiKey string) error
}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if apiKey := requestAPIKey(c); apiKey != "" {
			if err := authService.AuthenticateAPIKey(c.Request.Context(), apiKey); err == nil {
				c.Set("apiKeyAuthenticated", true)
				c.Next()
				return
			}
		}
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

func requestAPIKey(c *gin.Context) string {
	if value := strings.TrimSpace(c.GetHeader("Authorization")); strings.HasPrefix(strings.ToLower(value), "bearer ") {
		return strings.TrimSpace(value[len("bearer "):])
	}
	if value := strings.TrimSpace(c.GetHeader("X-API-Key")); value != "" {
		return value
	}
	return strings.TrimSpace(c.Query("api_key"))
}

func RequirePageAuth(cookieName string, authService interface {
	Authenticate(ctx context.Context, token string) (domain.Session, error)
}) gin.HandlerFunc {
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
