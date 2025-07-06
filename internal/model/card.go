package model

import "time"

type Card struct {
	ID         int64     `json:"id"`
	AccountID  int64     `json:"account_id"`
	CardNumber string    `json:"card_number"`
	CVV        string    `json:"-"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	ExpiryDate time.Time `json:"expiry_date"`
}
