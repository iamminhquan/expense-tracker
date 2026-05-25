package models

import "time"

type SavingsGoal struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `gorm:"index" json:"deleted_at,omitempty"`
	UserID        uint       `gorm:"not null;index" json:"user_id"`
	Name          string     `gorm:"size:100;not null" json:"name"`
	TargetAmount  float64    `gorm:"not null" json:"target_amount"`
	CurrentAmount float64    `gorm:"not null;default:0" json:"current_amount"`
	Currency      string     `gorm:"size:3;not null;default:'VND'" json:"currency"`
	Deadline      *time.Time `json:"deadline"`
	Icon          string     `gorm:"size:50" json:"icon"`
	Color         string     `gorm:"size:20" json:"color"`
}
