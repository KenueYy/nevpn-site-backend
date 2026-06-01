package yookassa

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/KenueYy/nevpn-site-backend/internal/db"
	"github.com/KenueYy/nevpn-site-backend/internal/models"
	"github.com/KenueYy/nevpn-site-backend/internal/remna"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IPaymentService interface {
	CreatePayment(ctx context.Context, userID string, email string, planID uint) (map[string]any, error)
	CreateCustomPayment(ctx context.Context, userID string, email string, price int64, description string, durationDays int, maxDevices int) (map[string]any, error)
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

func (s *YooKassaService) CreateCustomPayment(
	ctx context.Context,
	userID string,
	email string,
	price int64,
	description string,
	durationDays int,
	maxDevices int,
) (map[string]any, error) {
	payload := PaymentRequest{
		Amount: Amount{
			Value:    fmt.Sprintf("%.2f", float64(price)),
			Currency: "RUB",
		},
		Capture: true,
		Confirmation: Confirmation{
			Type:      "redirect",
			ReturnURL: "https://nevpn.shop/profile",
		},
		Description: description,
		Metadata: map[string]string{
			"user_id":       userID,
			"plan_id":       "0",
			"email":         email,
			"duration_days": fmt.Sprint(durationDays),
			"max_devices":   fmt.Sprint(maxDevices),
			"price":         fmt.Sprint(price),
		},
	}

	return s.client.CreatePayment(ctx, payload)
}

func (s *YooKassaService) PaymentSuccess(ctx context.Context, userID uuid.UUID, planID uint, webhook WebhookNotification) error {
	meta := webhook.Object.Metadata

	var existingPayment models.Payment
	if err := db.DB.First(&existingPayment, "id = ?", webhook.Object.ID).Error; err == nil {
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	var durationDays int
	var maxDevices int
	var amount int64
	isCustom := planID == 0

	if isCustom {
		// Custom payment: read params from metadata
		if v, ok := meta["duration_days"]; ok {
			d, _ := strconv.Atoi(v)
			durationDays = d
		}
		if v, ok := meta["max_devices"]; ok {
			d, _ := strconv.Atoi(v)
			maxDevices = d
		}
		if v, ok := meta["price"]; ok {
			p, _ := strconv.ParseInt(v, 10, 64)
			amount = p
		}
	} else {
		var plan models.Plan
		if err := db.DB.First(&plan, planID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("plan not found")
			}
			return fmt.Errorf("db error: %w", err)
		}
		durationDays = plan.DurationDays
		maxDevices = plan.MaxDevices
		amount = plan.Price
	}

	now := time.Now()

	payment := models.Payment{
		ID:        webhook.Object.ID,
		UserID:    userID,
		PlanID:    planID,
		Provider:  "yookassa",
		Amount:    amount,
		Status:    "succeeded",
		CreatedAt: now,
	}

	if err := db.DB.Create(&payment).Error; err != nil {
		return err
	}

	email := meta["email"]

	expire := time.Now().AddDate(0, 0, durationDays)

	bgCtx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	rUser, err := s.remnaService.GetUserByEmail(bgCtx, email)
	if err != nil {
		if err.Error() == "not found user by this email" {
			_, err := s.remnaService.CreateNewUser(ctx, &remna.RemnaUserRequest{
				Username:        userID.String(),
				UUID:            userID.String(),
				Status:          "ACTIVE",
				Email:           email,
				HwidDeviceLimit: uint(maxDevices),
				ExpireAt:        &expire,
				ActiveInternalSquads: []string{
					"9842a1b9-384d-4f6d-ab15-576b287f07bb",
					"5b3872a5-e484-4299-87cc-0fa980c1c510",
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
	newExpire = newExpire.AddDate(0, 0, durationDays)
	_, err = s.remnaService.UpdateUser(bgCtx, &remna.RemnaUserRequest{
		UUID:     rUser.UUID,
		ExpireAt: &newExpire,
		Username: rUser.Username,
	})
	return err
}
