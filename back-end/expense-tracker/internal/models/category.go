package models

import "time"

// CategoryType constants
const (
	CategoryTypeExpense = "expense"
	CategoryTypeIncome  = "income"
)

type Category struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`
	UserID    *uint      `gorm:"index" json:"user_id"` // NULL = system default category
	Name      string     `gorm:"size:100;not null" json:"name"`
	Type      string     `gorm:"size:10;not null" json:"type"`
	Icon      string     `gorm:"size:50" json:"icon"`
	Color     string     `gorm:"size:20" json:"color"`
}
