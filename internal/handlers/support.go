package handlers

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/KenueYy/nevpn-site-backend/internal/config"
	"github.com/KenueYy/nevpn-site-backend/internal/db"
	"github.com/KenueYy/nevpn-site-backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
)

func GetSupport(c *gin.Context) {
	cfg := c.MustGet("cfg").(*config.Config)

	c.JSON(http.StatusOK, gin.H{
		"telegram": gin.H{
			"label":  "Telegram",
			"url":    cfg.SupportTelegramURL,
			"handle": cfg.SupportTelegramHandle,
		},
		"email": gin.H{
			"label":   "Email",
			"address": cfg.SupportEmail,
		},
		"faq": gin.H{
			"label": "Частые вопросы",
			"items": []gin.H{
				{
					"q": "Как активировать тариф после оплаты?",
					"a": "После успешной оплаты доступ активируется автоматически. Статус можно проверить в личном кабинете.",
				},
				{
					"q": "Сколько устройств можно подключить?",
					"a": "Лимит указан в описании каждого тарифа.",
				},
				{
					"q": "Как связаться с поддержкой?",
					"a": "Напишите в Telegram или на email — мы отвечаем в рабочее время.",
				},
			},
		},
	})
}

var allowedContactMethods = map[string]bool{
	"phone":    true,
	"email":    true,
	"telegram": true,
	"whatsapp": true,
	"discord":  true,
	"other":    true,
}

func CreateTicket(c *gin.Context) {
	logger := c.MustGet("logger").(*slog.Logger)
	limiter := c.MustGet("limiter").(*redis_rate.Limiter)
	cfg := c.MustGet("cfg").(*config.Config)

	// Rate limit: 3 tickets per 15 minutes per IP
	limit := redis_rate.Limit{
		Rate:   3,
		Burst:  3,
		Period: 1 * time.Minute,
	}
	res, err := limiter.Allow(c.Request.Context(), "ticket:"+c.ClientIP(), limit)
	if err != nil || res.Allowed != 1 {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too_many_requests", "message": "Слишком много обращений. Попробуйте позже."})
		return
	}

	var req struct {
		Description   string `json:"description" binding:"required,min=10,max=5000"`
		ContactMethod string `json:"contact_method" binding:"required"`
		Contact       string `json:"contact" binding:"required,min=2,max=255"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		msg := "Заполните все поля: описание (10-5000 символов), способ связи, контакт (2-255 символов)"
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_error", "message": msg})
		return
	}

	if !allowedContactMethods[req.ContactMethod] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_contact_method", "message": "Недопустимый способ связи"})
		return
	}

	ticket := models.SupportTicket{
		Description:   req.Description,
		ContactMethod: req.ContactMethod,
		Contact:       req.Contact,
	}

	if err := db.DB.Create(&ticket).Error; err != nil {
		logger.Error("failed to create support ticket", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Не удалось создать обращение"})
		return
	}

	logger.Info("support ticket created",
		"ticket_id", ticket.ID,
		"contact_method", ticket.ContactMethod,
		"contact", ticket.Contact,
		"description_len", len(ticket.Description),
	)

	// Fire-and-forget: notify support team via smtp-service
	go func() {
		payload := map[string]string{
			"description":    ticket.Description,
			"contact_method": ticket.ContactMethod,
			"contact":        ticket.Contact,
			"to_email":       cfg.SupportEmail,
		}
		body, _ := json.Marshal(payload)
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Post(cfg.SmtpSupportTicketURL, "application/json", bytes.NewBuffer(body))
		if err != nil {
			logger.Error("failed to send support ticket to smtp", "ticket_id", ticket.ID, "error", err)
			return
		}
		resp.Body.Close()
		logger.Info("support ticket email sent", "ticket_id", ticket.ID, "smtp_status", resp.StatusCode)
	}()

	c.JSON(http.StatusCreated, gin.H{"message": "ticket_created", "ticket_id": ticket.ID})
}
