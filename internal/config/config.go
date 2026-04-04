package config

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost      string
	DBPort      int
	DBUser      string
	DBPassword  string
	DBName      string
	DBSSLMode   string
	Port        int
	RedisSecret string
}

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func Load() *Config {
	if err := godotenv.Load("config.env"); err != nil {
		logger.Error("failed to load config.env",
			"file", "config.env",
			"error", err.Error(),
		)
		os.Exit(1)
	}

	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		logger.Error("invalid PORT in env",
			"value", os.Getenv("PORT"),
			"error", err.Error(),
		)
		os.Exit(1)
	}

	dbPort, err := strconv.Atoi(os.Getenv("DB_PORT"))
	if err != nil {
		logger.Error("invalid DB_PORT in env",
			"value", os.Getenv("DB_PORT"),
			"error", err.Error(),
		)
		os.Exit(1)
	}

	config := Config{
		DBHost:      os.Getenv("DB_HOST"),
		DBPort:      dbPort,
		DBUser:      os.Getenv("DB_USER"),
		DBPassword:  os.Getenv("DB_PASSWORD"),
		DBName:      os.Getenv("DB_NAME"),
		DBSSLMode:   os.Getenv("DB_SSLMODE"),
		RedisSecret: os.Getenv("REDIS_SECRET"),
		Port:        port,
	}

	logger.Info("config loaded",
		"db_host", config.DBHost,
		"db_port", config.DBPort,
		"db_name", config.DBName,
		"port", config.Port,
	)

	return &config
}
