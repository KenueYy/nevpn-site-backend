package db

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/KenueYy/nevpn-site-backend/internal/config"
	"github.com/KenueYy/nevpn-site-backend/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DB     *gorm.DB
	logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
)

func Init(cfg *config.Config) error {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort, cfg.DBSSLMode,
	)

	logger.Info("connecting to database",
		"host", cfg.DBHost,
		"port", cfg.DBPort,
		"db_name", cfg.DBName,
		"sslmode", cfg.DBSSLMode,
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Error("failed to database connection",
			"host", cfg.DBHost,
			"port", cfg.DBPort,
			"db_name", cfg.DBName,
			"error", err.Error(),
		)
		return err
	}

	if err := DB.AutoMigrate(models.User{}, models.Plan{}, models.Payment{}, models.SubscriptionNotification{}); err != nil {
		logger.Error("migration failed",
			"db_name", cfg.DBName,
			"error", err.Error(),
		)
		return err
	}

	logger.Info("database initialized",
		"db_name", cfg.DBName,
	)

	return nil
}
