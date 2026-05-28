package yookassa

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type YooKassaHandler struct {
	service IPaymentService
	logger  *slog.Logger
}

func NewHandler(yookassaService IPaymentService, logger *slog.Logger) *YooKassaHandler {
	logger.Info("YooKassa handler created")

	return &YooKassaHandler{
		service: yookassaService,
		logger:  logger,
	}
}

func (h *YooKassaHandler) CreatePayment(c *gin.Context) {
	h.logger.Info("YooKassaHandler: CreatePayment started")

	userIDVal, ok := c.Get("user_id")
	if !ok {
		h.logger.Warn("YooKassaHandler: missing user_id in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, ok := userIDVal.(string)
	if !ok {
		h.logger.Error("YooKassaHandler: invalid user_id type")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user_id type"})
		return
	}

	emailVal, ok := c.Get("email")
	if !ok {
		h.logger.Warn("YooKassaHandler: missing email in context",
			"user_id", userID,
		)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	email, ok := emailVal.(string)
	if !ok {
		h.logger.Error("YooKassaHandler: invalid email type",
			"user_id", userID,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user_id type"})
		return
	}

	var req struct {
		PlanID uint `json:"plan_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("YooKassaHandler: failed to bind CreatePayment request",
			"user_id", userID,
			"email", email,
			"error", err,
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("YooKassaHandler: CreatePayment request validated",
		"user_id", userID,
		"email", email,
		"plan_id", req.PlanID,
	)

	resp, err := h.service.CreatePayment(c.Request.Context(), userID, email, req.PlanID)
	if err != nil {
		h.logger.Error("YooKassaHandler: CreatePayment failed",
			"user_id", userID,
			"email", email,
			"plan_id", req.PlanID,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("YooKassaHandler: CreatePayment succeeded",
		"user_id", userID,
		"email", email,
		"plan_id", req.PlanID,
	)

	c.JSON(http.StatusOK, resp)
}

func (h *YooKassaHandler) CreateCustomPayment(c *gin.Context) {
	h.logger.Info("YooKassaHandler: CreateCustomPayment started")

	userIDVal, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, _ := userIDVal.(string)

	emailVal, ok := c.Get("email")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	email, _ := emailVal.(string)

	var req struct {
		Price       int64  `json:"price" binding:"required"`
		Months      int    `json:"months" binding:"required"`
		Devices     int    `json:"devices" binding:"required"`
		Unlimited   bool   `json:"unlimited"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	desc := req.Description
	if desc == "" {
		desc = fmt.Sprintf("Оплата подписки neVPN (%d мес, %d устр)", req.Months, req.Devices)
	}

	durationDays := req.Months * 30
	devices := req.Devices
	if req.Unlimited {
		devices = 99999
	}

	resp, err := h.service.CreateCustomPayment(
		c.Request.Context(),
		userID, email,
		req.Price, desc,
		durationDays, devices,
	)
	if err != nil {
		h.logger.Error("YooKassaHandler: CreateCustomPayment failed",
			"user_id", userID,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *YooKassaHandler) Webhook(c *gin.Context) {
	h.logger.Info("YooKassaHandler: Webhook started")

	var webhook WebhookNotification
	if err := c.ShouldBindJSON(&webhook); err != nil {
		h.logger.Warn("YooKassaHandler: failed to bind webhook payload",
			"error", err,
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad payload"})
		return
	}

	h.logger.Info("YooKassaHandler: webhook received",
		"event", webhook.Event,
	)

	if webhook.Event != "payment.succeeded" {
		h.logger.Info("YooKassaHandler: webhook ignored",
			"event", webhook.Event,
		)
		c.JSON(http.StatusOK, gin.H{"message": "ignored"})
		return
	}

	userIDStr, ok := webhook.Object.Metadata["user_id"]
	if !ok || userIDStr == "" {
		h.logger.Warn("YooKassaHandler: missing user_id in webhook metadata",
			"event", webhook.Event,
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user_id"})
		return
	}

	planIDStr, ok := webhook.Object.Metadata["plan_id"]
	if !ok || planIDStr == "" {
		h.logger.Warn("YooKassaHandler: missing plan_id in webhook metadata",
			"event", webhook.Event,
			"user_id", userIDStr,
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing plan_id"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.Warn("YooKassaHandler: invalid user_id in webhook metadata",
			"event", webhook.Event,
			"user_id", userIDStr,
			"error", err,
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	planID, err := strconv.ParseUint(planIDStr, 10, 64)
	if err != nil {
		h.logger.Warn("YooKassaHandler: invalid plan_id in webhook metadata",
			"event", webhook.Event,
			"user_id", userID.String(),
			"plan_id", planIDStr,
			"error", err,
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan_id"})
		return
	}

	h.logger.Info("YooKassaHandler: webhook validated",
		"event", webhook.Event,
		"user_id", userID.String(),
		"plan_id", uint(planID),
	)

	h.service.PaymentSuccess(c.Request.Context(), userID, uint(planID), webhook)

	h.logger.Info("YooKassaHandler: PaymentSuccess handled",
		"event", webhook.Event,
		"user_id", userID.String(),
		"plan_id", uint(planID),
	)

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}
