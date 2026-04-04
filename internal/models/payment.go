package models

import (
	"time"

	"github.com/google/uuid"
)

type Payment struct {
	ID                uuid.UUID
	SubscriptionID    uuid.UUID
	Provider          string
	ProviderPaymentID string
	Amount            float64
	Currency          string
	Status            string
	PaidAt            *time.Time
	CreatedAt         time.Time
}
