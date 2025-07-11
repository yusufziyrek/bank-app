package model

import "time"

type Account struct {
	ID            int64     `db:"id" json:"id"`
	UserID        int64     `db:"user_id" json:"user_id"`
	AccountNumber string    `db:"account_number" json:"account_number"`
	Balance       float64   `db:"balance" json:"balance"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}
