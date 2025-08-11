package model

import "time"

const (
	TransactionTypeDeposit  = "deposit"
	TransactionTypeWithdraw = "withdraw"
	TransactionTypeTransfer = "transfer"
)

type Transaction struct {
	ID          int64     `db:"id" json:"id"`
	AccountID   int64     `db:"account_id" json:"account_id"`
	ToAccountID *int64    `db:"to_account_id" json:"to_account_id,omitempty"`
	Amount      float64   `db:"amount" json:"amount"`
	Type        string    `db:"type" json:"type"`
	Description string    `db:"description" json:"description,omitempty"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}
