package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/purya/emaildash/backend/internal/domain"
	usecase "github.com/purya/emaildash/backend/internal/usecase/cloudflare"
)

type CloudflareHandler struct {
	service usecase.Service
}

func NewCloudflareHandler(service usecase.Service) CloudflareHandler {
	return CloudflareHandler{service: service}
}

func (h CloudflareHandler) SaveCredentials(c *gin.Context) {
	var body struct {
		Email  string `json:"email"`
		APIKey string `json:"apiKey"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	zones, err := h.service.SaveCredentials(c.Request.Context(), domain.CloudflareCredentials{Email: body.Email, APIKey: body.APIKey})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"zones": zones})
}

func (h CloudflareHandler) ListZones(c *gin.Context) {
	zones, err := h.service.ListCachedZones(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"zones": zones})
}

func (h CloudflareHandler) Provision(c *gin.Context) {
	status, err := h.service.ProvisionZone(c.Request.Context(), c.Param("zoneId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h CloudflareHandler) Status(c *gin.Context) {
	status, err := h.service.Status(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}
