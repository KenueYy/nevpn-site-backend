package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/KenueYy/nevpn-site-backend/internal/models"
	"github.com/KenueYy/nevpn-site-backend/internal/remna"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EmailSender interface {
	SendSubscriptionNotification(ctx context.Context, email, notifType string, expireDate time.Time, renewalURL string) error
}

type HTTPSender struct {
	Client     *http.Client
	ServiceURL string
	Logger     *slog.Logger
}

func (s *HTTPSender) SendSubscriptionNotification(ctx context.Context, email, notifType string, expireDate time.Time, renewalURL string) error {
	payload := map[string]interface{}{
		"email":       email,
		"type":        notifType,
		"expire_date": expireDate,
		"renewal_url": renewalURL,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.ServiceURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("send to smtp service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("smtp service returned status %d", resp.StatusCode)
	}

	return nil
}

type CheckExpiringSubscriptionsJob struct {
	RemnaService remna.IRemnaService
	DB           *gorm.DB
	Sender       EmailSender
	RenewalURL   string
	Logger       *slog.Logger
}

func NewCheckExpiringSubscriptionsJob(
	remnaSvc remna.IRemnaService,
	db *gorm.DB,
	sender EmailSender,
	renewalURL string,
	logger *slog.Logger,
) *CheckExpiringSubscriptionsJob {
	return &CheckExpiringSubscriptionsJob{
		RemnaService: remnaSvc,
		DB:           db,
		Sender:       sender,
		RenewalURL:   renewalURL,
		Logger:       logger,
	}
}

func (j *CheckExpiringSubscriptionsJob) Run(ctx context.Context) {
	j.Logger.Info("CheckExpiringSubscriptionsJob: started")

	users, err := j.RemnaService.GetAllUsers(ctx)
	if err != nil {
		j.Logger.Error("CheckExpiringSubscriptionsJob: failed to fetch users from remna", "error", err)
		return
	}

	total := len(users.Users)
	var checked, sentExpiringSoon, sentExpired, skippedNoEmail, skippedError int

	now := time.Now()
	threeDaysFromNow := now.Add(3 * 24 * time.Hour)

	for _, user := range users.Users {
		checked++

		if user.Email == "" {
			skippedNoEmail++
			continue
		}

		if user.ExpireAt.IsZero() {
			continue
		}

		var notifType models.NotificationType

		if user.ExpireAt.Before(now) {
			notifType = models.NotifExpired
		} else if isWithinDay(user.ExpireAt, threeDaysFromNow) {
			notifType = models.NotifExpiringSoon
		} else {
			continue
		}

		if j.alreadyNotified(user.UUID, notifType) {
			continue
		}

		if err := j.Sender.SendSubscriptionNotification(ctx, user.Email, string(notifType), user.ExpireAt, j.RenewalURL); err != nil {
			j.Logger.Error("CheckExpiringSubscriptionsJob: failed to send notification",
				"remna_uuid", user.UUID,
				"email", user.Email,
				"type", notifType,
				"error", err,
			)
			skippedError++
			continue
		}

		record := models.SubscriptionNotification{
			ID:        uuid.New(),
			RemnaUUID: user.UUID,
			Email:     user.Email,
			Type:      notifType,
			SentAt:    now,
		}
		if err := j.DB.Create(&record).Error; err != nil {
			j.Logger.Warn("CheckExpiringSubscriptionsJob: failed to save notification record",
				"remna_uuid", user.UUID,
				"email", user.Email,
				"type", notifType,
				"error", err,
			)
		}

		switch notifType {
		case models.NotifExpiringSoon:
			sentExpiringSoon++
		case models.NotifExpired:
			sentExpired++
		}
	}

	j.Logger.Info("CheckExpiringSubscriptionsJob: finished",
		"total_users", total,
		"checked", checked,
		"sent_expiring_soon", sentExpiringSoon,
		"sent_expired", sentExpired,
		"skipped_no_email", skippedNoEmail,
		"skipped_error", skippedError,
	)
}

func isWithinDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}

func (j *CheckExpiringSubscriptionsJob) alreadyNotified(remnaUUID string, notifType models.NotificationType) bool {
	var count int64
	j.DB.Model(&models.SubscriptionNotification{}).
		Where("remna_uuid = ? AND type = ?", remnaUUID, string(notifType)).
		Count(&count)
	return count > 0
}
