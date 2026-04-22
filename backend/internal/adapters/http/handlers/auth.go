package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/purya/emaildash/backend/internal/domain"
	"github.com/purya/emaildash/backend/internal/usecase/auth"
)

type AuthHandler struct {
	service    auth.Service
	cookieName string
}

func NewAuthHandler(service auth.Service, cookieName string) AuthHandler {
	return AuthHandler{service: service, cookieName: cookieName}
}

func (h AuthHandler) Login(c *gin.Context) {
	var body struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	token, session, err := h.service.Login(c.Request.Context(), body.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	secureCookie := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(h.cookieName, token, int(time.Until(session.ExpiresAt).Seconds()), "/", "", secureCookie, true)
	c.JSON(http.StatusOK, gin.H{"csrfToken": session.CSRFToken, "expiresAt": session.ExpiresAt})
}

func (h AuthHandler) Logout(c *gin.Context) {
	token, _ := c.Cookie(h.cookieName)
	_ = h.service.Logout(c.Request.Context(), token)
	secureCookie := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetCookie(h.cookieName, "", -1, "/", "", secureCookie, true)
	c.Status(http.StatusNoContent)
}

func (h AuthHandler) Me(c *gin.Context) {
	sessionValue, exists := c.Get("session")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session required"})
		return
	}
	session := sessionValue.(domain.Session)
	c.JSON(http.StatusOK, gin.H{"authenticated": true, "expiresAt": session.ExpiresAt, "csrfToken": session.CSRFToken})
}

func (h AuthHandler) ChangePassword(c *gin.Context) {
	var body struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	apiKey, err := h.service.ChangePassword(c.Request.Context(), body.OldPassword, body.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"apiKey": apiKey})
}
