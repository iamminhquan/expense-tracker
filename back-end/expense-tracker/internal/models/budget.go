package models

import "time"

// BudgetPeriod constants
const (
	BudgetPeriodMonthly = "monthly"
	BudgetPeriodYearly  = "yearly"
)

type Budget struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `gorm:"index" json:"deleted_at,omitempty"`
	UserID     uint       `gorm:"not null;index" json:"user_id"`
	CategoryID uint       `gorm:"not null;index" json:"category_id"`
	Amount     float64    `gorm:"not null" json:"amount"`
	Period     string     `gorm:"size:10;not null" json:"period"`
	Year       int        `gorm:"not null" json:"year"`
	Month      *int       `gorm:"" json:"month"` // NULL when period = yearly
}
