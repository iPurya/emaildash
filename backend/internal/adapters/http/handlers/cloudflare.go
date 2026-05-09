package handlers

import (
	"net/http"
	"strings"

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

func (h CloudflareHandler) Domains(c *gin.Context) {
	var (
		domains []domain.ReceivingDomain
		err     error
	)
	if isRefreshRequested(c) {
		domains, err = h.service.RefreshReceivingDomains(c.Request.Context())
	} else {
		domains, err = h.service.ReceivingDomains(c.Request.Context())
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	writeDomains(c, domains)
}

func (h CloudflareHandler) Provision(c *gin.Context) {
	h.EnableReceiving(c)
}

func (h CloudflareHandler) EnableReceiving(c *gin.Context) {
	status, err := h.service.EnableReceiving(c.Request.Context(), c.Param("zoneId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h CloudflareHandler) DisableReceiving(c *gin.Context) {
	status, err := h.service.DisableReceiving(c.Request.Context(), c.Param("zoneId"))
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

func writeDomains(c *gin.Context, domains []domain.ReceivingDomain) {
	readyDomains := make([]string, 0, len(domains))
	for _, item := range domains {
		if item.Ready {
			readyDomains = append(readyDomains, item.Domain)
		}
	}
	c.JSON(http.StatusOK, gin.H{"domains": domains, "readyDomains": readyDomains})
}

func isRefreshRequested(c *gin.Context) bool {
	value := strings.ToLower(strings.TrimSpace(c.Query("refresh")))
	return value == "1" || value == "true" || value == "yes"
}
