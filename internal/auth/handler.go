package auth

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service IAuthService
	logger  *slog.Logger
}

func NewHandler(service IAuthService, logger *slog.Logger) *AuthHandler {
	logger.Info("auth handler created")

	return &AuthHandler{
		service: service,
		logger:  logger,
	}
}

func (h *AuthHandler) SendCode(c *gin.Context) {
	h.logger.Info("AuthHandler: SendCode started")

	type EmailReq struct {
		Email string `json:"email" binding:"required,email"`
	}
	var eReq EmailReq

	if err := c.ShouldBindJSON(&eReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.service.SendCode(c.Request.Context(), eReq.Email)
}
