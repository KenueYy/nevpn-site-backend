package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/KenueYy/nevpn-site-backend/internal/remna"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const trialDuration = 72 * time.Hour
const trialDeviceLimit uint = 1

// defaultSquads — те же, что используются при обычном создании пользователя через PaymentSuccess.
var trialSquads = []string{
	"9842a1b9-384d-4f6d-ab15-576b287f07bb",
	"5b3872a5-e484-4299-87cc-0fa980c1c510",
}

func StartTrial(c *gin.Context) {
	logger := c.MustGet("logger").(*slog.Logger)
	rdb := c.MustGet("rdb").(*redis.Client)
	remnaSvc := c.MustGet("remna_service").(remna.IRemnaService)

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

	ctx := c.Request.Context()

	// Check if trial already used
	trialUsed, err := rdb.Get(ctx, "trial_used:"+email).Result()
	if err == nil && trialUsed != "" {
		c.JSON(http.StatusConflict, gin.H{"error": "trial_already_used", "message": "Вы уже использовали пробный доступ"})
		return
	}

	now := time.Now()
	expireAt := now.Add(trialDuration)

	// Тот же способ, что и PaymentSuccess:
	// если пользователь уже есть в Remna — продлеваем, если нет — создаём.
	bgCtx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	rUser, err := remnaSvc.GetUserByEmail(bgCtx, email)
	if err != nil {
		// Не найден — создаём нового (так же как в PaymentSuccess)
		_, err := remnaSvc.CreateNewUser(ctx, &remna.RemnaUserRequest{
			Username:             userID,
			UUID:                 userID,
			Status:               "ACTIVE",
			Email:                email,
			HwidDeviceLimit:      trialDeviceLimit,
			ExpireAt:             &expireAt,
			ActiveInternalSquads: trialSquads,
		})
		if err != nil {
			logger.Error("failed to create trial user in remna", "email", email, "error", err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to activate trial"})
			return
		}
	} else {
		// Найден — продлеваем (так же как в PaymentSuccess)
		newExpire := rUser.ExpireAt
		if newExpire.Before(now) {
			newExpire = now
		}
		newExpire = newExpire.Add(trialDuration)
		_, err = remnaSvc.UpdateUser(bgCtx, &remna.RemnaUserRequest{
			UUID:     rUser.UUID,
			ExpireAt: &newExpire,
			Username: rUser.Username,
		})
		if err != nil {
			logger.Error("failed to update trial user in remna", "email", email, "error", err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to activate trial"})
			return
		}
	}

	// Mark trial as used (permanent key — never expires)
	if err := rdb.Set(ctx, "trial_used:"+email, "1", 0).Err(); err != nil {
		logger.Error("failed to set trial_used key", "email", email, "error", err)
	}

	logger.Info("trial activated", "email", email, "user_id", userID, "expire_at", expireAt)

	c.JSON(http.StatusOK, gin.H{
		"message":   "trial_activated",
		"expire_at": expireAt,
		"status":    "active",
	})
}
