package models

import "time"

// AccountType constants
const (
	AccountTypeCash       = "cash"
	AccountTypeBank       = "bank"
	AccountTypeCreditCard = "credit_card"
	AccountTypeEWallet    = "e_wallet"
)

type Account struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`
	UserID    uint       `gorm:"not null;index" json:"user_id"`
	Name      string     `gorm:"size:100;not null" json:"name"`
	Type      string     `gorm:"size:20;not null" json:"type"`
	Balance   float64    `gorm:"not null;default:0" json:"balance"`
	Currency  string     `gorm:"size:3;not null;default:'VND'" json:"currency"`
	IsDefault bool       `gorm:"not null;default:false" json:"is_default"`
}
