package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"

	"github.com/KenueYy/nevpn-site-backend/internal/config"
	"github.com/KenueYy/nevpn-site-backend/internal/db"
	"github.com/KenueYy/nevpn-site-backend/internal/handlers"
	"github.com/KenueYy/nevpn-site-backend/internal/middleware"
	"github.com/KenueYy/nevpn-site-backend/internal/remna"
	"github.com/KenueYy/nevpn-site-backend/internal/yookassa"
	"github.com/gin-contrib/sessions"
	redisstore "github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

var (
	ctx     = context.Background()
	rdb     = redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	limiter *redis_rate.Limiter
	store   sessions.Store
	cfg     = config.Load()
	logger  = slog.New(slog.NewJSONHandler(os.Stdout, nil))
)

func initRedis() {
	limiter = redis_rate.NewLimiter(rdb)

	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Error("redis ping failed", "error", err)
		panic(err)
	}

	var err error
	store, err = redisstore.NewStore(
		10,
		"tcp",
		rdb.Options().Addr,
		"",
		"",
		[]byte(cfg.RedisSecret),
	)
	if err != nil {
		logger.Error("session store init failed", "error", err)
		panic(err)
	}

	logger.Info("redis store initialized")
}

func main() {
	initRedis()

	if err := db.Init(cfg); err != nil {
		logger.Error("failed to initialize database", "error", err.Error())
		os.Exit(1)
	}

	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Set("rdb", rdb)
		c.Set("ctx", ctx)
		c.Set("logger", logger)
		c.Set("limiter", limiter)
		c.Set("db", db.DB)
		c.Set("cfg", cfg)
		c.Next()
	})

	r.Use(sessions.Sessions("auth", store))

	v1 := r.Group("/api/v1")
	v1.POST("/sendcode", handlers.SendCode)
	v1.POST("/login", handlers.Login)
	v1.GET("/plans", handlers.GetPlans)
	v1.GET("/plans/:id", handlers.GetPlan)

	remnaClient := remna.NewClient(cfg, logger)
	remnaService := remna.NewService(remnaClient, logger)
	remnaHandler := remna.NewHandler(remnaService, logger)

	yookassaClient := yookassa.NewClient(cfg, logger)
	yookassaService := yookassa.NewService(yookassaClient, remnaService, logger)
	yookassaHandler := yookassa.NewHandler(yookassaService, logger)

	v1.POST("yookassa/webhook", yookassaHandler.Webhook)

	v1.GET("/remna/users", remnaHandler.GetAllUsers)
	v1.GET("/remna/users/:uuid", remnaHandler.GetUserByUUID)
	v1.GET("/remna/users/by-email/:email", remnaHandler.GetUserByEmail)
	v1.GET("/remna/users/by-telegram/:id", remnaHandler.GetUserByTelegramID)
	v1.PATCH("/remna/users", remnaHandler.UpdateUser)
	v1.POST("/remna/users", remnaHandler.CreateNewUser)

	v1.Use(middleware.RequireAuth)
	{
		v1.GET("/profile", handlers.Profile)
		v1.POST("/yookassa/payment/create", yookassaHandler.CreatePayment)

	}

	admin := v1.Group("/admin")
	admin.Use(middleware.RequireAuth, middleware.RequireAdmin)
	{
		admin.POST("/plans", handlers.AddPlan)
		admin.DELETE("/plans/:id", handlers.DeletePlanWithID)
		admin.PATCH("/plans/:id", handlers.UpdatePlan)
	}

	logger.Info("server starting", "port", cfg.Port)
	r.Run(":" + strconv.Itoa(cfg.Port))
}
