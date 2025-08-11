package dto

import (
	"time"

	"github.com/yusufziyrek/bank-app/internal/model"
)

type CreateCardRequest struct {
	AccountID  int64  `json:"account_id" validate:"required,gt=0"`
	CardNumber string `json:"card_number" validate:"required,len=16,numeric"`
	CVV        string `json:"cvv" validate:"required,len=3,numeric"`
}

type UpdateCardRequest struct {
	CardNumber string     `json:"card_number,omitempty" validate:"omitempty,len=16,numeric"`
	CVV        string     `json:"cvv,omitempty" validate:"omitempty,len=3,numeric"`
	ExpiryDate *time.Time `json:"expiry_date,omitempty" validate:"omitempty"`
}

type UpdateCardStatusRequest struct {
	IsActive bool `json:"is_active" validate:"required"`
}

type CardResponse struct {
	ID         int64     `json:"id"`
	AccountID  int64     `json:"account_id"`
	CardNumber string    `json:"card_number"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	ExpiryDate time.Time `json:"expiry_date"`
}

type CardsResponse struct {
	Cards []CardResponse `json:"cards"`
	Count int            `json:"count"`
}

func maskCardNumber(num string) string {
	// Keep last 4 digits, mask the rest; group by 4 for readability
	if len(num) < 4 {
		return "****"
	}
	last4 := num[len(num)-4:]
	// Determine masked length (non-last4)
	maskedLen := len(num) - 4
	masked := make([]rune, 0, len(num)+3)
	for i := 0; i < maskedLen; i++ {
		// insert separator every 4 chars for readability
		if i > 0 && i%4 == 0 {
			masked = append(masked, '-')
		}
		masked = append(masked, '*')
	}
	if maskedLen > 0 {
		masked = append(masked, '-')
	}
	masked = append(masked, []rune(last4)...)
	return string(masked)
}

func CardResponseFromModel(c model.Card) CardResponse {
	return CardResponse{
		ID:         c.ID,
		AccountID:  c.AccountID,
		CardNumber: maskCardNumber(c.CardNumber),
		IsActive:   c.IsActive,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
		ExpiryDate: c.ExpiryDate,
	}
}

func CardsResponseFromModels(cards []model.Card) CardsResponse {
	resp := make([]CardResponse, len(cards))
	for i, c := range cards {
		resp[i] = CardResponseFromModel(c)
	}
	return CardsResponse{
		Cards: resp,
		Count: len(resp),
	}
}
