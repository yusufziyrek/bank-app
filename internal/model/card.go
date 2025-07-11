package model

import "time"

type Card struct {
	ID         int64     `db:"id" json:"id"`
	AccountID  int64     `db:"account_id" json:"account_id"`
	CardNumber string    `db:"card_number" json:"card_number"`
	CVV        string    `db:"cvv" json:"-"`
	IsActive   bool      `db:"is_active" json:"is_active"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
	ExpiryDate time.Time `db:"expiry_date" json:"expiry_date"`
}
