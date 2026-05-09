package handlers

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/purya/emaildash/backend/internal/domain"
	usecase "github.com/purya/emaildash/backend/internal/usecase/cloudflare"
)

type CloudflareHandler struct {
	service     usecase.Service
	domainCache *receivingDomainsCache
}

func NewCloudflareHandler(service usecase.Service) CloudflareHandler {
	return CloudflareHandler{service: service, domainCache: &receivingDomainsCache{}}
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
	h.clearDomainCache()
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
	if !isRefreshRequested(c) {
		if domains, ok := h.domainCache.get(); ok {
			writeDomains(c, domains)
			return
		}
	}

	domains, err := h.service.ReceivingDomains(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.domainCache.set(domains)
	writeDomains(c, domains)
}

func (h CloudflareHandler) Provision(c *gin.Context) {
	status, err := h.service.ProvisionZone(c.Request.Context(), c.Param("zoneId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.clearDomainCache()
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

const receivingDomainsCacheTTL = time.Hour

type receivingDomainsCache struct {
	mu        sync.Mutex
	domains   []domain.ReceivingDomain
	expiresAt time.Time
	valid     bool
}

func (cache *receivingDomainsCache) get() ([]domain.ReceivingDomain, bool) {
	if cache == nil {
		return nil, false
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if !cache.valid || !time.Now().UTC().Before(cache.expiresAt) {
		return nil, false
	}
	return append([]domain.ReceivingDomain(nil), cache.domains...), true
}

func (cache *receivingDomainsCache) set(domains []domain.ReceivingDomain) {
	if cache == nil {
		return
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.domains = append([]domain.ReceivingDomain(nil), domains...)
	cache.expiresAt = time.Now().UTC().Add(receivingDomainsCacheTTL)
	cache.valid = true
}

func (cache *receivingDomainsCache) clear() {
	if cache == nil {
		return
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.domains = nil
	cache.expiresAt = time.Time{}
	cache.valid = false
}

func (h CloudflareHandler) clearDomainCache() {
	h.domainCache.clear()
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
