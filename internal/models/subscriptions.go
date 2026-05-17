package models

import (
	"time"

	"github.com/google/uuid"
)

type Subscription struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	PlanID    uint
	Plan      Plan
	StartedAt time.Time
	ExpiresAt time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}
