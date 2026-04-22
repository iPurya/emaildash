package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/purya/emaildash/backend/internal/usecase/setup"
)

type SetupHandler struct {
	service setup.Service
}

func NewSetupHandler(service setup.Service) SetupHandler {
	return SetupHandler{service: service}
}

func (h SetupHandler) Status(c *gin.Context) {
	status, err := h.service.Status(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h SetupHandler) Initialize(c *gin.Context) {
	var body struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if err := h.service.Initialize(c.Request.Context(), body.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusCreated)
}
