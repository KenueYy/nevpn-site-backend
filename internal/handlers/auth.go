package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/KenueYy/nevpn-site-backend/internal/models"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"

	"gorm.io/gorm"
)

func SendCode(c *gin.Context) {
	var req models.EmailCode
	rdb := c.MustGet("rdb").(*redis.Client)
	ctx := c.MustGet("ctx").(context.Context)
	limiter := c.MustGet("limiter").(*redis_rate.Limiter)
	logger := c.MustGet("logger").(*slog.Logger)

	type EmailReq struct {
		Email string `json:"email" binding:"required,email"`
	}
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
	res, err := limiter.Allow(ctx, "sendcode:"+req.Email, limit)
	if err != nil || res.Allowed != 1 {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests"})
		return
	}

	n := rand.Intn(999999)
	code := fmt.Sprintf(strconv.Itoa(n))

	if err := rdb.Set(ctx, "code:"+eReq.Email, code, 10000*time.Minute).Err(); err != nil {
		logger.Error("redis set failed", "email", req.Email, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
		return
	}

	client := &http.Client{}
	sendReq := map[string]string{"Email": eReq.Email,
		"Code": code}
	sendJSON, _ := json.Marshal(sendReq)
	resp, _ := client.Post("http://localhost:4444/api/v1/sendcode",
		"application/json", bytes.NewBuffer(sendJSON))
	fmt.Println("Send code:", resp.Status)
	resp.Body.Close()

	logger.Info("code sent", "email", req.Email)

	c.JSON(http.StatusOK, gin.H{"message": "Code sent", "retry_after": 60})
}

type LoginRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}

func Login(c *gin.Context) {
	var req LoginRequest

	rdb := c.MustGet("rdb").(*redis.Client)
	ctx := c.MustGet("ctx").(context.Context)
	limiter := c.MustGet("limiter").(*redis_rate.Limiter)
	logger := c.MustGet("logger").(*slog.Logger)
	db := c.MustGet("db").(*gorm.DB)

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

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	savedCode, err := rdb.Get(ctx, "code:"+req.Email).Result()
	if err != nil || savedCode != req.Code {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid code"})
		return
	}
	_ = rdb.Del(ctx, "code:"+req.Email).Err()

	var user models.User
	if err := db.
		Where("email = ?", req.Email).
		FirstOrCreate(&user, models.User{Email: req.Email}).
		Error; err != nil {
		logger.Error("firstOrCreate user failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	session := sessions.Default(c)
	session.Set("user_id", user.ID.String())
	session.Set("email", user.Email)
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
		"user":    user,
	})
}
