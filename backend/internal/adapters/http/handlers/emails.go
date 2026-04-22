package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/purya/emaildash/backend/internal/domain"
	"github.com/purya/emaildash/backend/internal/usecase/inbox"
)

type EmailsHandler struct {
	service inbox.Service
}

func NewEmailsHandler(service inbox.Service) EmailsHandler {
	return EmailsHandler{service: service}
}

func (h EmailsHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	toMail := c.Query("to_mail")
	if toMail == "" {
		toMail = c.Query("recipient")
	}
	filter := domain.EmailListFilter{
		Recipient:  c.Query("recipient"),
		FromMail:   c.Query("from_mail"),
		ToMail:     toMail,
		UnreadOnly: c.Query("unread") == "true",
		Limit:      limit,
	}
	emails, err := h.service.ListEmails(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"emails": emails})
}

func (h EmailsHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	email, err := h.service.GetEmail(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, email)
}

func (h EmailsHandler) MarkRead(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h EmailsHandler) ListRecipients(c *gin.Context) {
	recipients, err := h.service.ListRecipients(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"recipients": recipients})
}
