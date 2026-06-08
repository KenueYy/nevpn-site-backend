package models

import (
	"time"

	"github.com/google/uuid"
)

type SupportTicket struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID        *uuid.UUID `gorm:"type:uuid;index" json:"user_id,omitempty"`
	Subject       string     `gorm:"type:varchar(500);not null;default:''" json:"subject"`
	Description   string     `gorm:"type:text;not null" json:"description"`
	ContactMethod string     `gorm:"type:varchar(50);not null" json:"contact_method"`
	Contact       string     `gorm:"type:varchar(255);not null" json:"contact"`
	Status        string     `gorm:"type:varchar(50);not null;default:'open'" json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}
