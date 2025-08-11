package dto

import "github.com/yusufziyrek/bank-app/internal/model"

type CreateAccountRequest struct {
	Balance float64 `json:"balance" validate:"required,gte=0"`
}

type UpdateAccountRequest struct {
	Balance float64 `json:"balance" validate:"required,gte=0"`
}

type AccountResponse struct {
	ID            int64          `json:"id"`
	UserID        int64          `json:"user_id"`
	AccountNumber string         `json:"account_number"`
	Balance       float64        `json:"balance"`
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"`
	Cards         []CardResponse `json:"cards,omitempty"`
}

type AccountsResponse struct {
	Accounts []AccountResponse `json:"accounts"`
	Count    int               `json:"count"`
}

func MapCardsToDTO(cards []model.Card) []CardResponse {
	if len(cards) == 0 {
		return nil
	}
	out := make([]CardResponse, len(cards))
	for i, c := range cards {
		out[i] = CardResponseFromModel(c)
	}
	return out
}
