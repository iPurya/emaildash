package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/purya/emaildash/backend/internal/domain"
	"github.com/purya/emaildash/backend/internal/usecase/ingest"
)

type IngestHandler struct {
	service ingest.Service
}

func NewIngestHandler(service ingest.Service) IngestHandler {
	return IngestHandler{service: service}
}

func (h IngestHandler) Receive(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unable to read request body"})
		return
	}
	var payload domain.IngestPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	email, err := h.service.Ingest(c.Request.Context(), c.GetHeader("X-Emaildash-Timestamp"), c.GetHeader("X-Emaildash-Signature"), body, payload)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, email)
}
