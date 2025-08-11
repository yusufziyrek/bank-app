package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/yusufziyrek/bank-app/internal/model"
	"github.com/yusufziyrek/bank-app/internal/repository"
)

type CardService interface {
	GetAllCards(ctx context.Context) ([]model.Card, error)
	GetCardByID(ctx context.Context, id int64) (model.Card, error)
	GetCardsByUser(ctx context.Context, userID int64) ([]model.Card, error)
	GetCardsByAccount(ctx context.Context, accountID int64) ([]model.Card, error)
	CreateCard(ctx context.Context, card *model.Card) error
	UpdateCard(ctx context.Context, card *model.Card) error
	UpdateCardStatus(ctx context.Context, id int64, active bool) error
	DeleteCard(ctx context.Context, id int64) error
}

func (s *cardService) GetCardsByAccount(ctx context.Context, accountID int64) ([]model.Card, error) {
	cards, err := s.repo.GetCardsByAccount(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("service:GetCardsByAccount: %w", err)
	}
	return cards, nil
}

type cardService struct {
	repo repository.CardRepository
}

func NewCardService(r repository.CardRepository) CardService {
	return &cardService{repo: r}
}

func (s *cardService) GetAllCards(ctx context.Context) ([]model.Card, error) {
	return s.repo.GetAllCards(ctx)
}

func (s *cardService) GetCardByID(ctx context.Context, id int64) (model.Card, error) {
	c, err := s.repo.GetCardById(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Card{}, ErrCardNotFound
		}
		return model.Card{}, fmt.Errorf("service:GetCardByID: %w", err)
	}
	return c, nil
}

func (s *cardService) GetCardsByUser(ctx context.Context, userID int64) ([]model.Card, error) {
	cards, err := s.repo.GetCardsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("service:GetCardsByUser: %w", err)
	}
	return cards, nil
}

func (s *cardService) CreateCard(ctx context.Context, card *model.Card) error {
	// Limit cards per account to 3
	cards, err := s.repo.GetCardsByAccount(ctx, card.AccountID)
	if err == nil && len(cards) >= 3 {
		return ErrMaxAccountsExceeded
	}
	err = s.repo.AddCard(ctx, card)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrCardAlreadyExists
		}
		return fmt.Errorf("service:CreateCard: %w", err)
	}
	return nil
}

func (s *cardService) UpdateCard(ctx context.Context, card *model.Card) error {
	err := s.repo.UpdateCard(ctx, card)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCardNotFound
		}
		return fmt.Errorf("service:UpdateCard: %w", err)
	}
	return nil
}

func (s *cardService) UpdateCardStatus(ctx context.Context, id int64, active bool) error {
	err := s.repo.UpdateCardActiveStatus(ctx, id, active)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCardNotFound
		}
		return fmt.Errorf("service:UpdateCardStatus: %w", err)
	}
	return nil
}

func (s *cardService) DeleteCard(ctx context.Context, id int64) error {
	err := s.repo.DeleteCard(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCardNotFound
		}
		return fmt.Errorf("service:DeleteCard: %w", err)
	}
	return nil
}
