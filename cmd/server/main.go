package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/KenueYy/nevpn-site-backend/internal/config"
	"github.com/KenueYy/nevpn-site-backend/internal/db"
	"github.com/KenueYy/nevpn-site-backend/internal/handlers"
	"github.com/KenueYy/nevpn-site-backend/internal/jobs"
	"github.com/KenueYy/nevpn-site-backend/internal/middleware"
	"github.com/KenueYy/nevpn-site-backend/internal/remna"
	"github.com/KenueYy/nevpn-site-backend/internal/yookassa"
	"github.com/gin-contrib/sessions"
	redisstore "github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
)

var (
	ctx     = context.Background()
	rdb     *redis.Client
	limiter *redis_rate.Limiter
	store   sessions.Store
	cfg     *config.Config
	logger  = slog.New(slog.NewJSONHandler(os.Stdout, nil))
)

func initRedis(c *config.Config) {
	rdb = redis.NewClient(&redis.Options{Addr: c.RedisAddr})

	limiter = redis_rate.NewLimiter(rdb)

	var pingErr error
	for i := 0; i < 30; i++ {
		pingErr = rdb.Ping(ctx).Err()
		if pingErr == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if pingErr != nil {
		logger.Error("redis ping failed", "addr", c.RedisAddr, "error", pingErr)
		panic(pingErr)
	}

	var err error
	store, err = redisstore.NewStore(
		10,
		"tcp",
		c.RedisAddr,
		"",
		"",
		[]byte(c.RedisSecret),
	)
	if err != nil {
		logger.Error("session store init failed", "error", err)
		panic(err)
	}

	logger.Info("redis store initialized", "addr", c.RedisAddr)
}

func main() {
	cfg = config.Load()
	initRedis(cfg)

	if err := db.Init(cfg); err != nil {
		logger.Error("failed to initialize database", "error", err.Error())
		os.Exit(1)
	}

	r := gin.Default()

	r.Use(middleware.CORS(cfg))
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

	remnaClient := remna.NewClient(cfg, logger)
	remnaService := remna.NewService(remnaClient, logger)
	remnaHandler := remna.NewHandler(remnaService, logger)

	yookassaClient := yookassa.NewClient(cfg, logger)
	yookassaService := yookassa.NewService(yookassaClient, remnaService, logger)
	yookassaHandler := yookassa.NewHandler(yookassaService, logger)

	r.Use(func(c *gin.Context) {
		c.Set("remna_service", remnaService)
		c.Next()
	})

	v1 := r.Group("/api/v1")

	v1.POST("/sendcode", handlers.SendCode)
	v1.POST("/login", handlers.Login)
	v1.GET("/plans", handlers.GetPlans)
	v1.GET("/plans/:id", handlers.GetPlan)
	v1.POST("/plans/calculate", handlers.CalculatePlan)
	v1.GET("/support", handlers.GetSupport)
	v1.POST("/support/tickets", handlers.CreateTicket)
	v1.POST("/yookassa/webhook", yookassaHandler.Webhook)

	auth := v1.Group("")
	auth.Use(middleware.RequireAuth)
	{
		auth.GET("/profile", handlers.Profile)
		auth.GET("/me", handlers.Me)
		auth.POST("/logout", handlers.Logout)
		auth.GET("/subscription", handlers.GetSubscription)
		auth.POST("/yookassa/payment/create", yookassaHandler.CreatePayment)
		auth.POST("/yookassa/payment/create-custom", yookassaHandler.CreateCustomPayment)
		auth.POST("/trial", handlers.StartTrial)
		auth.GET("/tickets", handlers.GetTickets)
	}

	admin := v1.Group("/admin")
	admin.Use(middleware.RequireAuth, middleware.RequireAdmin)
	{
		admin.GET("/plans", handlers.AdminListPlans)
		admin.POST("/plans", handlers.AddPlan)
		admin.PATCH("/plans/reorder", handlers.AdminReorderPlans)
		admin.DELETE("/plans/:id", handlers.DeletePlanWithID)
		admin.PATCH("/plans/:id", handlers.UpdatePlan)

		admin.GET("/tickets", handlers.AdminGetAllTickets)
		admin.PATCH("/tickets/:id", handlers.AdminUpdateTicketStatus)

		admin.GET("/remna/users", remnaHandler.GetAllUsers)
		admin.GET("/remna/users/:uuid", remnaHandler.GetUserByUUID)
		admin.GET("/remna/users/by-email/:email", remnaHandler.GetUserByEmail)
		admin.GET("/remna/users/by-telegram/:id", remnaHandler.GetUserByTelegramID)
		admin.PATCH("/remna/users", remnaHandler.UpdateUser)
		admin.POST("/remna/users", remnaHandler.CreateNewUser)
	}

	// Cron scheduler for subscription notifications — runs daily at 12:00 server time.
	c := cron.New(cron.WithLocation(time.Local))

	subscriptionSender := &jobs.HTTPSender{
		Client:     &http.Client{Timeout: 15 * time.Second},
		ServiceURL: cfg.SmtpSubscriptionURL,
		Logger:     logger,
	}

	subscriptionJob := jobs.NewCheckExpiringSubscriptionsJob(
		remnaService,
		db.DB,
		subscriptionSender,
		cfg.SubscriptionRenewalURL,
		logger,
	)

	// Run once at startup for immediate check (non-blocking)
	go func() {
		logger.Info("running initial subscription notification check")
		subscriptionJob.Run(ctx)
	}()

	_, err := c.AddFunc("0 12 * * *", func() {
		logger.Info("cron: starting subscription notification check")
		subscriptionJob.Run(ctx)
	})
	if err != nil {
		logger.Error("failed to register cron job", "error", err)
		os.Exit(1)
	}

	c.Start()

	logger.Info("server starting", "port", cfg.Port)
	r.Run(":" + strconv.Itoa(cfg.Port))
}
