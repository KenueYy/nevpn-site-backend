package models

import (
	"time"

	"github.com/google/uuid"
)

type SupportTicket struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Description   string    `gorm:"type:text;not null" json:"description"`
	ContactMethod string    `gorm:"type:varchar(50);not null" json:"contact_method"`
	Contact       string    `gorm:"type:varchar(255);not null" json:"contact"`
	CreatedAt     time.Time `json:"created_at"`
}
