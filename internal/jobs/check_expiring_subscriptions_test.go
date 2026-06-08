package jobs

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/KenueYy/nevpn-site-backend/internal/models"
	"github.com/KenueYy/nevpn-site-backend/internal/remna"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var testLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

// mockRemnaService implements remna.IRemnaService for testing.
type mockRemnaService struct {
	users []remna.RemnaUserResponse
	err   error
}

func (m *mockRemnaService) GetAllUsers(ctx context.Context) (*remna.AllUserResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &remna.AllUserResponse{
		Total: uint(len(m.users)),
		Users: m.users,
	}, nil
}

func (m *mockRemnaService) GetUserByUUID(ctx context.Context, id string) (*remna.RemnaUserResponse, error) {
	return nil, nil
}
func (m *mockRemnaService) GetUserByEmail(ctx context.Context, email string) (*remna.RemnaUserResponse, error) {
	return nil, nil
}
func (m *mockRemnaService) GetUserByTelegramID(ctx context.Context, tgID string) (*remna.RemnaUserResponse, error) {
	return nil, nil
}
func (m *mockRemnaService) CreateNewUser(ctx context.Context, user *remna.RemnaUserRequest) (*remna.RemnaUserResponse, error) {
	return nil, nil
}
func (m *mockRemnaService) UpdateUser(ctx context.Context, user *remna.RemnaUserRequest) (*remna.RemnaUserResponse, error) {
	return nil, nil
}

// mockSender records sent notifications for assertions.
type mockSender struct {
	sent []sentNotification
}

type sentNotification struct {
	Email      string
	NotifType  string
	ExpireDate time.Time
	RenewalURL string
}

func (s *mockSender) SendSubscriptionNotification(ctx context.Context, email, notifType string, expireDate time.Time, renewalURL string) error {
	s.sent = append(s.sent, sentNotification{
		Email:      email,
		NotifType:  notifType,
		ExpireDate: expireDate,
		RenewalURL: renewalURL,
	})
	return nil
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}

	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS subscription_notifications (
			id TEXT PRIMARY KEY,
			remna_uuid TEXT NOT NULL,
			email TEXT NOT NULL,
			type TEXT NOT NULL,
			sent_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_sub_notif_remna_uuid ON subscription_notifications(remna_uuid);
		CREATE INDEX IF NOT EXISTS idx_sub_notif_email ON subscription_notifications(email);
	`).Error; err != nil {
		t.Fatalf("failed to create table: %v", err)
	}
	return db
}

func TestCheckExpiringSubscriptions_SendsExpiringSoon(t *testing.T) {
	db := setupTestDB(t)
	mockRemna := &mockRemnaService{
		users: []remna.RemnaUserResponse{
			{
				UUID:     "uuid-1",
				Email:    "test@example.com",
				ExpireAt: time.Now().Add(3 * 24 * time.Hour), // 3 days from now
				Status:   "ACTIVE",
			},
		},
	}
	mockSend := &mockSender{}

	job := NewCheckExpiringSubscriptionsJob(mockRemna, db, mockSend, "https://nevpn.shop/account", testLogger)
	job.Run(context.Background())

	if len(mockSend.sent) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(mockSend.sent))
	}
	if mockSend.sent[0].NotifType != "expiring_soon" {
		t.Errorf("expected expiring_soon, got %s", mockSend.sent[0].NotifType)
	}
	if mockSend.sent[0].Email != "test@example.com" {
		t.Errorf("expected test@example.com, got %s", mockSend.sent[0].Email)
	}
}

func TestCheckExpiringSubscriptions_SendsExpired(t *testing.T) {
	db := setupTestDB(t)
	mockRemna := &mockRemnaService{
		users: []remna.RemnaUserResponse{
			{
				UUID:     "uuid-2",
				Email:    "expired@example.com",
				ExpireAt: time.Now().Add(-1 * time.Hour), // already expired
				Status:   "ACTIVE",
			},
		},
	}
	mockSend := &mockSender{}

	job := NewCheckExpiringSubscriptionsJob(mockRemna, db, mockSend, "https://nevpn.shop/account", testLogger)
	job.Run(context.Background())

	if len(mockSend.sent) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(mockSend.sent))
	}
	if mockSend.sent[0].NotifType != "expired" {
		t.Errorf("expected expired, got %s", mockSend.sent[0].NotifType)
	}
}

func TestCheckExpiringSubscriptions_SkipsNoEmail(t *testing.T) {
	db := setupTestDB(t)
	mockRemna := &mockRemnaService{
		users: []remna.RemnaUserResponse{
			{
				UUID:     "uuid-3",
				Email:    "", // no email
				ExpireAt: time.Now().Add(3 * 24 * time.Hour),
				Status:   "ACTIVE",
			},
		},
	}
	mockSend := &mockSender{}

	job := NewCheckExpiringSubscriptionsJob(mockRemna, db, mockSend, "https://nevpn.shop/account", testLogger)
	job.Run(context.Background())

	if len(mockSend.sent) != 0 {
		t.Errorf("expected 0 notifications for user without email, got %d", len(mockSend.sent))
	}
}

func TestCheckExpiringSubscriptions_SkipsFutureExpiry(t *testing.T) {
	db := setupTestDB(t)
	mockRemna := &mockRemnaService{
		users: []remna.RemnaUserResponse{
			{
				UUID:     "uuid-4",
				Email:    "future@example.com",
				ExpireAt: time.Now().Add(30 * 24 * time.Hour), // far in future
				Status:   "ACTIVE",
			},
		},
	}
	mockSend := &mockSender{}

	job := NewCheckExpiringSubscriptionsJob(mockRemna, db, mockSend, "https://nevpn.shop/account", testLogger)
	job.Run(context.Background())

	if len(mockSend.sent) != 0 {
		t.Errorf("expected 0 notifications for future expiry, got %d", len(mockSend.sent))
	}
}

func TestCheckExpiringSubscriptions_DedupPreventsResend(t *testing.T) {
	db := setupTestDB(t)

	// Pre-create a notification record
	db.Create(&models.SubscriptionNotification{
		ID:        uuid.New(),
		RemnaUUID: "uuid-5",
		Email:     "dup@example.com",
		Type:      models.NotifExpiringSoon,
		SentAt:    time.Now(),
	})

	mockRemna := &mockRemnaService{
		users: []remna.RemnaUserResponse{
			{
				UUID:     "uuid-5",
				Email:    "dup@example.com",
				ExpireAt: time.Now().Add(3 * 24 * time.Hour),
				Status:   "ACTIVE",
			},
		},
	}
	mockSend := &mockSender{}

	job := NewCheckExpiringSubscriptionsJob(mockRemna, db, mockSend, "https://nevpn.shop/account", testLogger)
	job.Run(context.Background())

	if len(mockSend.sent) != 0 {
		t.Errorf("expected 0 notifications due to dedup, got %d", len(mockSend.sent))
	}
}

func TestCheckExpiringSubscriptions_MultipleUsers(t *testing.T) {
	db := setupTestDB(t)
	mockRemna := &mockRemnaService{
		users: []remna.RemnaUserResponse{
			{
				UUID:     "uuid-a",
				Email:    "a@example.com",
				ExpireAt: time.Now().Add(3 * 24 * time.Hour),
			},
			{
				UUID:     "uuid-b",
				Email:    "b@example.com",
				ExpireAt: time.Now().Add(-1 * time.Hour),
			},
			{
				UUID:     "uuid-c",
				Email:    "", // no email
				ExpireAt: time.Now().Add(3 * 24 * time.Hour),
			},
			{
				UUID:     "uuid-d",
				Email:    "d@example.com",
				ExpireAt: time.Now().Add(10 * 24 * time.Hour), // not yet
			},
		},
	}
	mockSend := &mockSender{}

	job := NewCheckExpiringSubscriptionsJob(mockRemna, db, mockSend, "https://nevpn.shop/account", testLogger)
	job.Run(context.Background())

	if len(mockSend.sent) != 2 {
		t.Fatalf("expected 2 notifications, got %d", len(mockSend.sent))
	}

	hasExpiringSoon := false
	hasExpired := false
	for _, s := range mockSend.sent {
		if s.NotifType == "expiring_soon" {
			hasExpiringSoon = true
		}
		if s.NotifType == "expired" {
			hasExpired = true
		}
	}
	if !hasExpiringSoon {
		t.Error("missing expiring_soon notification")
	}
	if !hasExpired {
		t.Error("missing expired notification")
	}
}

func TestCheckExpiringSubscriptions_ZeroExpireAt(t *testing.T) {
	db := setupTestDB(t)
	mockRemna := &mockRemnaService{
		users: []remna.RemnaUserResponse{
			{
				UUID:     "uuid-zero",
				Email:    "zero@example.com",
				ExpireAt: time.Time{}, // zero time
				Status:   "ACTIVE",
			},
		},
	}
	mockSend := &mockSender{}

	job := NewCheckExpiringSubscriptionsJob(mockRemna, db, mockSend, "https://nevpn.shop/account", testLogger)
	job.Run(context.Background())

	if len(mockSend.sent) != 0 {
		t.Errorf("expected 0 notifications for zero expireAt, got %d", len(mockSend.sent))
	}
}

func TestCheckExpiringSubscriptions_ExpiringSoonDaily(t *testing.T) {
	// Notification sent yesterday should NOT prevent sending again today.
	db := setupTestDB(t)

	// Pre-create a notification record from yesterday
	db.Create(&models.SubscriptionNotification{
		ID:        uuid.New(),
		RemnaUUID: "uuid-daily",
		Email:     "daily@example.com",
		Type:      models.NotifExpiringSoon,
		SentAt:    time.Now().Add(-24 * time.Hour),
	})

	mockRemna := &mockRemnaService{
		users: []remna.RemnaUserResponse{
			{
				UUID:     "uuid-daily",
				Email:    "daily@example.com",
				ExpireAt: time.Now().Add(2 * 24 * time.Hour), // 2 days left
				Status:   "ACTIVE",
			},
		},
	}
	mockSend := &mockSender{}

	job := NewCheckExpiringSubscriptionsJob(mockRemna, db, mockSend, "https://nevpn.shop/account", testLogger)
	job.Run(context.Background())

	if len(mockSend.sent) != 1 {
		t.Fatalf("expected 1 notification (yesterday's should not block today), got %d", len(mockSend.sent))
	}
}

func TestCheckExpiringSubscriptions_ExpiredOnce(t *testing.T) {
	// Expired notification sent yesterday should NOT be sent again (all-time dedup).
	db := setupTestDB(t)

	db.Create(&models.SubscriptionNotification{
		ID:        uuid.New(),
		RemnaUUID: "uuid-expired-once",
		Email:     "expired-once@example.com",
		Type:      models.NotifExpired,
		SentAt:    time.Now().Add(-24 * time.Hour),
	})

	mockRemna := &mockRemnaService{
		users: []remna.RemnaUserResponse{
			{
				UUID:     "uuid-expired-once",
				Email:    "expired-once@example.com",
				ExpireAt: time.Now().Add(-1 * time.Hour), // expired
				Status:   "ACTIVE",
			},
		},
	}
	mockSend := &mockSender{}

	job := NewCheckExpiringSubscriptionsJob(mockRemna, db, mockSend, "https://nevpn.shop/account", testLogger)
	job.Run(context.Background())

	if len(mockSend.sent) != 0 {
		t.Errorf("expected 0 expired notifications (all-time dedup), got %d", len(mockSend.sent))
	}
}

func TestIsWithinDay(t *testing.T) {
	tests := []struct {
		a, b     time.Time
		expected bool
	}{
		{time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC), time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC), true},
		{time.Date(2026, 5, 30, 11, 0, 0, 0, time.UTC), time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC), false},
		{time.Date(2026, 5, 30, 23, 59, 59, 0, time.UTC), time.Date(2026, 5, 30, 0, 0, 1, 0, time.UTC), true},
		{time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), false},
	}
	for _, tt := range tests {
		result := isWithinDay(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("isWithinDay(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
		}
	}
}
