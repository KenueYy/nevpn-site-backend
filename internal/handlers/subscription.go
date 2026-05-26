package handlers

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/KenueYy/nevpn-site-backend/internal/db"
	"github.com/KenueYy/nevpn-site-backend/internal/models"
	"github.com/KenueYy/nevpn-site-backend/internal/remna"
	"github.com/gin-gonic/gin"
)

func subscriptionStatus(remnaStatus string, expireAt time.Time) string {
	st := strings.ToUpper(strings.TrimSpace(remnaStatus))
	if st == "ACTIVE" {
		return "active"
	}
	if !expireAt.IsZero() && expireAt.After(time.Now()) {
		return "active"
	}
	if !expireAt.IsZero() {
		return "expired"
	}
	return "none"
}

func GetSubscription(c *gin.Context) {
	logger := c.MustGet("logger").(*slog.Logger)
	remnaSvc := c.MustGet("remna_service").(remna.IRemnaService)

	emailVal, ok := c.Get("email")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	email, _ := emailVal.(string)

	empty := gin.H{
		"status":            "none",
		"expire_at":         nil,
		"remna_status":      nil,
		"subscription_url":  nil,
		"plan_tag":          nil,
		"plan_id":           nil,
		"plan_name":         nil,
	}

	user, err := remnaSvc.GetUserByEmail(c.Request.Context(), email)
	if err != nil {
		logger.Info("remna user not found for subscription", "email", email, "error", err)
		c.JSON(http.StatusOK, empty)
		return
	}

	var plan models.Plan
	var planID *uint
	var planName *string
	if user.Tag != "" {
		if err := db.DB.Where("tag = ?", user.Tag).First(&plan).Error; err == nil {
			id := plan.ID
			name := plan.Name
			planID = &id
			planName = &name
		}
	}

	expire := user.ExpireAt
	var expirePtr *time.Time
	if !expire.IsZero() {
		expirePtr = &expire
	}

	c.JSON(http.StatusOK, gin.H{
		"status":           subscriptionStatus(user.Status, expire),
		"expire_at":        expirePtr,
		"remna_status":     user.Status,
		"subscription_url": user.SubscriptionUrl,
		"plan_tag":         user.Tag,
		"plan_id":          planID,
		"plan_name":        planName,
	})
}
