package models

import (
	"time"
)

type Plan struct {
	ID           uint   `gorm:"type:uint;primaryKey;"`
	Name         string `gorm:"type:varchar(100);not null"`
	Description  string `gorm:"type:text"`
	ImageURL     string `gorm:"type:text"`
	Price        int64  `gorm:"not null"`
	DurationDays int    `gorm:"not null"`
	MaxDevices   int    `gorm:"not null"`
	Tag          string

	CreatedAt time.Time
	UpdatedAt time.Time
}
