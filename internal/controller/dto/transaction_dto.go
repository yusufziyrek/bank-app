package dto

type CreateTransactionRequest struct {
	AccountID   int64   `json:"account_id" validate:"required,gt=0"`
	ToAccountID *int64  `json:"to_account_id,omitempty" validate:"omitempty,gt=0"`
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	Type        string  `json:"type" validate:"required,oneof=deposit withdraw transfer"`
	Description string  `json:"description"`
}

type UpdateTransactionRequest struct {
	ToAccountID *int64  `json:"to_account_id,omitempty" validate:"omitempty,gt=0"`
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	Type        string  `json:"type" validate:"required,oneof=deposit withdraw transfer"`
	Description string  `json:"description"`
}

type TransactionResponse struct {
	ID          int64   `json:"id"`
	AccountID   int64   `json:"account_id"`
	ToAccountID *int64  `json:"to_account_id,omitempty"`
	Amount      float64 `json:"amount"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	CreatedAt   string  `json:"created_at"`
}

type TransactionsResponse struct {
	Transactions []TransactionResponse `json:"transactions"`
	Count        int                   `json:"count"`
}
