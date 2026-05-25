package models

import "time"

type Income struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `gorm:"index" json:"deleted_at,omitempty"`
	UserID          uint       `gorm:"not null;index" json:"user_id"`
	AccountID       *uint      `gorm:"index" json:"account_id"`
	Amount          float64    `gorm:"not null" json:"amount"`
	Currency        string     `gorm:"size:3;not null;default:'VND'" json:"currency"`
	Description     string     `gorm:"size:255;not null" json:"description"`
	CategoryID      *uint      `gorm:"index" json:"category_id"`
	Source          string     `gorm:"size:255" json:"source"`
	Date            time.Time  `gorm:"not null;index" json:"date"`
	IsRecurring     bool       `gorm:"not null;default:false" json:"is_recurring"`
	RecurringPeriod string     `gorm:"size:20" json:"recurring_period"`
	Note            string     `gorm:"size:1000" json:"note"`
}
