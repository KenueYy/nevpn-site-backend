package models

import (
	"time"

	"github.com/google/uuid"
)

type Payment struct {
	ID       string    `gorm:"primaryKey"`
	UserID   uuid.UUID `gorm:"type:uuid;not null;index"`
	PlanID   uint      `gorm:"not null;index"`
	Provider string    `gorm:"not null"`
	Amount   int64     `gorm:"not null"`
	Status   string    `gorm:"not null"`

	CreatedAt time.Time
}
