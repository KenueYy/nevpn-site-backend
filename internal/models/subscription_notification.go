package models

import (
	"time"

	"github.com/google/uuid"
)

// NotificationType defines the type of subscription notification.
type NotificationType string

const (
	NotifExpiringSoon NotificationType = "expiring_soon"
	NotifExpired      NotificationType = "expired"
)

// SubscriptionNotification tracks sent subscription notifications to prevent duplicates.
type SubscriptionNotification struct {
	ID        uuid.UUID        `gorm:"type:uuid;primaryKey"`
	RemnaUUID string           `gorm:"type:varchar(36);not null;index"`
	Email     string           `gorm:"type:varchar(255);not null;index"`
	Type      NotificationType `gorm:"type:varchar(20);not null"`
	SentAt    time.Time        `gorm:"not null;default:now()"`
}
