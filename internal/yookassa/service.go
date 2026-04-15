package yookassa

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/KenueYy/nevpn-site-backend/internal/db"
	"github.com/KenueYy/nevpn-site-backend/internal/models"
	"github.com/KenueYy/nevpn-site-backend/internal/remna"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IPaymentService interface {
	CreatePayment(ctx context.Context, userID string, email string, planID uint) (map[string]any, error)
	PaymentSuccess(ctx context.Context, userID uuid.UUID, planID uint, webhook WebhookNotification) error
}

type YooKassaService struct {
	client       IPaymentClient
	remnaService remna.IRemnaService
	logger       *slog.Logger
}

func NewService(yooKassaClient IPaymentClient, remna remna.IRemnaService, logger *slog.Logger) *YooKassaService {
	return &YooKassaService{
		client:       yooKassaClient,
		remnaService: remna,
		logger:       logger,
	}
}

func (s *YooKassaService) CreatePayment(ctx context.Context, userID string, email string, planID uint) (map[string]any, error) {
	var plan models.Plan
	if err := db.DB.First(&plan, planID).Error; err != nil {
		return nil, err
	}

	payload := PaymentRequest{
		Amount: Amount{
			Value:    fmt.Sprintf("%.2f", float64(plan.Price)),
			Currency: "RUB",
		},
		Capture: true,
		Confirmation: Confirmation{
			Type:      "redirect",
			ReturnURL: "https://nevpn.shop/",
		},
		Description: fmt.Sprintf("Оплата подписки %s", plan.Name),
		Metadata: map[string]string{
			"user_id": userID,
			"plan_id": fmt.Sprint(plan.ID),
			"email":   email,
		},
	}

	return s.client.CreatePayment(ctx, payload)
}

func (s *YooKassaService) PaymentSuccess(ctx context.Context, userID uuid.UUID, planID uint, webhook WebhookNotification) error {

	var existingPayment models.Payment
	if err := db.DB.First(&existingPayment, "id = ?", webhook.Object.ID).Error; err == nil {
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	var plan models.Plan
	if err := db.DB.First(&plan, planID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("plan not found")
		}
		return fmt.Errorf("db error: %w", err)
	}

	now := time.Now()

	payment := models.Payment{
		ID:        webhook.Object.ID,
		UserID:    userID,
		PlanID:    plan.ID,
		Provider:  "yookassa",
		Amount:    plan.Price,
		Status:    "succeeded",
		CreatedAt: now,
	}

	if err := db.DB.Create(&payment).Error; err != nil {
		return err
	}

	email := webhook.Object.Metadata["email"]

	expire := time.Now().AddDate(0, 0, plan.DurationDays)

	bgCtx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	rUser, err := s.remnaService.GetUserByEmail(bgCtx, email)
	if err != nil {
		if err.Error() == "Not found user by this email" {
			_, err := s.remnaService.CreateNewUser(ctx, &remna.RemnaUserRequest{
				Username:        userID.String(),
				UUID:            userID.String(),
				Status:          "ACTIVE",
				Email:           email,
				Tag:             plan.Tag,
				HwidDeviceLimit: uint(plan.MaxDevices),
				ExpireAt:        &expire,
				ActiveInternalSquads: []string{
					"9842a1b9-384d-4f6d-ab15-576b287f07bb",
				},
			})
			return err
		}
		return err
	}

	newExpire := rUser.ExpireAt
	if newExpire.Before(now) {
		newExpire = now
	}
	newExpire = newExpire.AddDate(0, 0, plan.DurationDays)
	_, err = s.remnaService.UpdateUser(bgCtx, &remna.RemnaUserRequest{
		UUID:     rUser.UUID,
		ExpireAt: &newExpire,
		Username: rUser.Username,
	})
	return err
}
