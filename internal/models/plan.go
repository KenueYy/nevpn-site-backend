package models

import "time"

type Plan struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Name         string    `json:"name" gorm:"type:varchar(100);not null"`
	Description  string    `json:"description" gorm:"type:text"`
	ImageURL     string    `json:"image_url" gorm:"type:text"`
	Price        int64     `json:"price" gorm:"not null"`
	DurationDays int       `json:"duration_days" gorm:"not null"`
	MaxDevices   int       `json:"max_devices" gorm:"not null"`
	Tag          string    `json:"tag"`
	SortOrder    int       `json:"sort_order" gorm:"default:0"`
	Active       bool      `json:"active" gorm:"default:true"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
