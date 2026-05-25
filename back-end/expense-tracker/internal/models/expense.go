package models

import "time"

type Expense struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `gorm:"index" json:"deleted_at,omitempty"`
	UserID          uint       `gorm:"not null;index" json:"user_id"`
	AccountID       *uint      `gorm:"index" json:"account_id"`
	CategoryID      *uint      `gorm:"index" json:"category_id"`
	Amount          float64    `gorm:"not null" json:"amount"`
	Currency        string     `gorm:"size:3;not null;default:'USD'" json:"currency"`
	Description     string     `gorm:"size:255;not null" json:"description"`
	Category        string     `gorm:"size:120;not null" json:"category"`
	Date            time.Time  `gorm:"not null;index" json:"date"`
	PaymentMethod   string     `gorm:"size:120" json:"payment_method"`
	IsRecurring     bool       `gorm:"not null;default:false" json:"is_recurring"`
	RecurringPeriod string     `gorm:"size:20" json:"recurring_period"`
	Note            string     `gorm:"size:1000" json:"note"`
}
