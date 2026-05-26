package handlers

import (
	"net/http"

	"github.com/KenueYy/nevpn-site-backend/internal/config"
	"github.com/gin-gonic/gin"
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
