package models

import (
	"time"

	"github.com/google/uuid"
)

type Subscription struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	PlanID    int
	Status    string
	StartedAt time.Time
	ExpiresAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}
