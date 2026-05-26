package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"time"

	"github.com/KenueYy/nevpn-site-backend/internal/config"
	"github.com/KenueYy/nevpn-site-backend/internal/db"
	"github.com/KenueYy/nevpn-site-backend/internal/models"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

type EmailReq struct {
	Email string `json:"email" binding:"required,email"`
}

type LoginRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}

func SendCode(c *gin.Context) {
	rdb := c.MustGet("rdb").(*redis.Client)
	ctx := c.MustGet("ctx").(context.Context)
	limiter := c.MustGet("limiter").(*redis_rate.Limiter)
	logger := c.MustGet("logger").(*slog.Logger)
	cfg := c.MustGet("cfg").(*config.Config)

	var eReq EmailReq
	if err := c.ShouldBindJSON(&eReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	limit := redis_rate.Limit{
		Rate:   1,
		Burst:  1,
		Period: time.Minute / 10,
	}
	res, err := limiter.Allow(ctx, "sendcode:"+eReq.Email, limit)
	if err != nil || res.Allowed != 1 {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests"})
		return
	}

	code := fmt.Sprintf("%06d", rand.Intn(1000000))

	if err := rdb.Set(ctx, "code:"+eReq.Email, code, 15*time.Minute).Err(); err != nil {
		logger.Error("redis set failed", "email", eReq.Email, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
		return
	}

	sendReq := map[string]string{"Email": eReq.Email, "Code": code}
	sendJSON, _ := json.Marshal(sendReq)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(cfg.MailServiceURL, "application/json", bytes.NewBuffer(sendJSON))
	if err != nil {
		logger.Error("mail service request failed", "email", eReq.Email, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send code"})
		return
	}
	resp.Body.Close()

	logger.Info("code sent", "email", eReq.Email)

	c.JSON(http.StatusOK, gin.H{"message": "Code sent", "retry_after": 60})
}

func Login(c *gin.Context) {
	rdb := c.MustGet("rdb").(*redis.Client)
	ctx := c.MustGet("ctx").(context.Context)
	limiter := c.MustGet("limiter").(*redis_rate.Limiter)
	logger := c.MustGet("logger").(*slog.Logger)
	cfg := c.MustGet("cfg").(*config.Config)

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	limit := redis_rate.Limit{
		Rate:   1,
		Burst:  1,
		Period: time.Minute / 10,
	}
	res, err := limiter.Allow(ctx, "login:"+req.Email, limit)
	if err != nil || res.Allowed != 1 {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests"})
		return
	}

	savedCode, err := rdb.Get(ctx, "code:"+req.Email).Result()
	if err != nil || savedCode != req.Code {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid code"})
		return
	}
	_ = rdb.Del(ctx, "code:"+req.Email).Err()

	var user models.User
	if err := db.DB.
		Where("email = ?", req.Email).
		FirstOrCreate(&user, models.User{Email: req.Email}).
		Error; err != nil {
		logger.Error("firstOrCreate user failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	role := "user"
	if cfg.IsAdmin(user.Email) {
		role = "admin"
	}

	session := sessions.Default(c)
	session.Set("user_id", user.ID.String())
	session.Set("email", user.Email)
	session.Set("role", role)
	session.Options(sessions.Options{
		Path:     "/",
		MaxAge:   24 * 3600,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
	if err := session.Save(); err != nil {
		logger.Error("session save failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Logged in",
		"user": gin.H{
			"id":         user.ID,
			"email":      user.Email,
			"role":       role,
			"created_at": user.CreatedAt,
		},
	})
}

func Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Options(sessions.Options{
		Path:   "/",
		MaxAge: -1,
	})
	_ = session.Save()
	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}
