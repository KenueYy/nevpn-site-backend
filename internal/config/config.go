package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

var CFG *Config

type Config struct {
	DBHost            string
	DBPort            int
	DBUser            string
	DBPassword        string
	DBName            string
	DBSSLMode         string
	Port              int
	RedisAddr         string
	RedisSecret       string
	YooKassaShopID    string
	YooKassaSecretKey string
	RemnaToken        string

	AdminEmails []string
	CorsOrigins []string

	SupportTelegramURL    string
	SupportTelegramHandle string
	SupportEmail          string
	MailServiceURL        string
}

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func Load() *Config {
	loadEnvFiles()

	cfg := &Config{
		DBHost:            envOr("DB_HOST", "localhost"),
		DBPort:            envInt("DB_PORT", 5432),
		DBUser:            envOr("DB_USER", "postgres"),
		DBPassword:        envOr("DB_PASSWORD", "password"),
		DBName:            envOr("DB_NAME", "nevpn"),
		DBSSLMode:         envOr("DB_SSLMODE", "disable"),
		Port:              envInt("PORT", 7080),
		RedisAddr:         envOr("REDIS_ADDR", "localhost:6379"),
		RedisSecret:       os.Getenv("REDIS_SECRET"),
		YooKassaShopID:    os.Getenv("YOOKASSA_SHOP_ID"),
		YooKassaSecretKey: os.Getenv("YOOKASSA_SECRET_KEY"),
		RemnaToken:        os.Getenv("REMNA_TOKEN"),

		AdminEmails: csvOr("ADMIN_EMAILS", []string{
			"kenueyy@gmail.com",
		}),
		CorsOrigins: csvOr("CORS_ORIGINS", []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
			"http://localhost:8080",
			"http://127.0.0.1:8080",
		}),

		SupportTelegramURL:    envOr("SUPPORT_TELEGRAM_URL", "https://t.me/KenueYx"),
		SupportTelegramHandle: envOr("SUPPORT_TELEGRAM_HANDLE", "@nevpn_support"),
		SupportEmail:          envOr("SUPPORT_EMAIL", "support@nevpn.shop"),
		MailServiceURL:        envOr("MAIL_SERVICE_URL", "http://localhost:4444/api/v1/sendcode"),
	}

	validate(cfg)

	logger.Info("config loaded",
		"db_host", cfg.DBHost,
		"db_port", cfg.DBPort,
		"db_name", cfg.DBName,
		"port", cfg.Port,
		"redis_addr", cfg.RedisAddr,
		"admin_emails_count", len(cfg.AdminEmails),
		"cors_origins", cfg.CorsOrigins,
	)

	CFG = cfg
	return cfg
}

func loadEnvFiles() {
	candidates := make([]string, 0, 5)

	if custom := strings.TrimSpace(os.Getenv("ENV_FILE")); custom != "" {
		candidates = append(candidates, custom)
	}

	candidates = append(candidates,
		"config.env",
		".env",
		"cmd/server/config.env",
	)

	for _, file := range candidates {
		if _, err := os.Stat(file); err == nil {
			if err := godotenv.Load(file); err == nil {
				logger.Info("env file loaded", "file", file)
				return
			}
			logger.Warn("env file exists but failed to load", "file", file)
		}
	}

	logger.Info("using process environment only (no env file loaded)")
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}

	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		logger.Error("invalid integer env", "key", key, "value", v, "error", err.Error())
		os.Exit(1)
	}
	return n
}

func csvOr(key string, fallback []string) []string {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}

	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}

	if len(out) == 0 {
		return fallback
	}

	return out
}

func validate(cfg *Config) {
	require("DB_HOST", cfg.DBHost)
	require("DB_USER", cfg.DBUser)
	require("DB_NAME", cfg.DBName)

	if cfg.DBPort <= 0 {
		logger.Error("invalid DB_PORT", "value", cfg.DBPort)
		os.Exit(1)
	}
	if cfg.Port <= 0 {
		logger.Error("invalid PORT", "value", cfg.Port)
		os.Exit(1)
	}
}

func require(name, value string) {
	if strings.TrimSpace(value) == "" {
		logger.Error("missing required config", "key", name)
		os.Exit(1)
	}
}

func (c *Config) IsAdmin(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	for _, a := range c.AdminEmails {
		if strings.ToLower(strings.TrimSpace(a)) == email {
			return true
		}
	}
	return false
}
