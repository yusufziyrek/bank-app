package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yusufziyrek/bank-app/internal/model"
)

const (
	queryGetAllCards = `
		SELECT id, account_id, card_number, cvv, is_active, created_at, updated_at, expiry_date
		FROM cards
	`
	queryGetCardByID = `
		SELECT id, account_id, card_number, cvv, is_active, created_at, updated_at, expiry_date
		FROM cards
		WHERE id = $1
	`
	queryGetCardsByUser = `
		SELECT c.id, c.account_id, c.card_number, c.cvv, c.is_active, c.created_at, c.updated_at, c.expiry_date
		FROM cards c
		INNER JOIN accounts a ON c.account_id = a.id
		WHERE a.user_id = $1
	`
	queryAddCard = `
		INSERT INTO cards (account_id, card_number, cvv, is_active, created_at, updated_at, expiry_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	queryUpdateCard = `
		UPDATE cards
		   SET account_id  = $1,
			   card_number  = $2,
			   cvv          = $3,
			   is_active    = $4,
			   updated_at   = $5,
			   expiry_date  = $6
		 WHERE id = $7
	`
	queryUpdateCardStatus = `
		UPDATE cards
		   SET is_active  = $1,
			   updated_at = $2
		 WHERE id         = $3
	`
	queryDeleteCard = `
		DELETE FROM cards WHERE id = $1
	`
)

type CardRepository interface {
	GetAllCards(ctx context.Context) ([]model.Card, error)
	GetCardById(ctx context.Context, id int64) (model.Card, error)
	GetCardsByUser(ctx context.Context, userId int64) ([]model.Card, error)
	GetCardsByAccount(ctx context.Context, accountId int64) ([]model.Card, error)
	AddCard(ctx context.Context, card *model.Card) error
	UpdateCard(ctx context.Context, card *model.Card) error
	UpdateCardActiveStatus(ctx context.Context, id int64, isActive bool) error
	DeleteCard(ctx context.Context, id int64) error
}

const queryGetCardsByAccount = `
	SELECT id, account_id, card_number, cvv, is_active, created_at, updated_at, expiry_date
	FROM cards
	WHERE account_id = $1
`

func (r *cardRepo) GetCardsByAccount(ctx context.Context, accountId int64) ([]model.Card, error) {
	rows, err := r.pool.Query(ctx, queryGetCardsByAccount, accountId)
	if err != nil {
		return nil, fmt.Errorf("repo:GetCardsByAccount: %w", err)
	}
	defer rows.Close()
	cards, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Card])
	if err != nil {
		return nil, fmt.Errorf("repo:GetCardsByAccount:scan: %w", err)
	}
	return cards, nil
}

type cardRepo struct {
	pool *pgxpool.Pool
}

func NewCardRepository(pool *pgxpool.Pool) CardRepository {
	return &cardRepo{pool: pool}
}

func (r *cardRepo) GetAllCards(ctx context.Context) ([]model.Card, error) {
	rows, err := r.pool.Query(ctx, queryGetAllCards)
	if err != nil {
		return nil, fmt.Errorf("repo:GetAllCards: %w", err)
	}
	defer rows.Close()

	cards, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Card])
	if err != nil {
		return nil, fmt.Errorf("repo:GetAllCards:scan: %w", err)
	}
	return cards, nil
}

func (r *cardRepo) GetCardById(ctx context.Context, id int64) (model.Card, error) {
	var card model.Card
	err := r.pool.QueryRow(ctx, queryGetCardByID, id).Scan(
		&card.ID,
		&card.AccountID,
		&card.CardNumber,
		&card.CVV,
		&card.IsActive,
		&card.CreatedAt,
		&card.UpdatedAt,
		&card.ExpiryDate,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return card, pgx.ErrNoRows
		}
		return card, fmt.Errorf("repo:GetCardById: %w", err)
	}
	return card, nil
}

func (r *cardRepo) GetCardsByUser(ctx context.Context, userId int64) ([]model.Card, error) {
	rows, err := r.pool.Query(ctx, queryGetCardsByUser, userId)
	if err != nil {
		return nil, fmt.Errorf("repo:GetCardsByUser: %w", err)
	}
	defer rows.Close()

	cards, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Card])
	if err != nil {
		return nil, fmt.Errorf("repo:GetCardsByUser:scan: %w", err)
	}
	return cards, nil
}

func (r *cardRepo) AddCard(ctx context.Context, card *model.Card) error {
	now := time.Now()
	card.CreatedAt = now
	card.UpdatedAt = now

	err := r.pool.QueryRow(
		ctx,
		queryAddCard,
		card.AccountID,
		card.CardNumber,
		card.CVV,
		card.IsActive,
		card.CreatedAt,
		card.UpdatedAt,
		card.ExpiryDate,
	).Scan(&card.ID)
	if err != nil {
		return fmt.Errorf("repo:AddCard: %w", err)
	}
	return nil
}

func (r *cardRepo) UpdateCard(ctx context.Context, card *model.Card) error {
	card.UpdatedAt = time.Now()
	_, err := r.pool.Exec(
		ctx,
		queryUpdateCard,
		card.AccountID,
		card.CardNumber,
		card.CVV,
		card.IsActive,
		card.UpdatedAt,
		card.ExpiryDate,
		card.ID,
	)
	if err != nil {
		return fmt.Errorf("repo:UpdateCard: %w", err)
	}
	return nil
}

func (r *cardRepo) UpdateCardActiveStatus(ctx context.Context, id int64, isActive bool) error {
	updatedAt := time.Now()
	_, err := r.pool.Exec(
		ctx, queryUpdateCardStatus,
		isActive,
		updatedAt,
		id,
	)
	if err != nil {
		return fmt.Errorf("repo:UpdateCardActiveStatus: %w", err)
	}
	return nil
}

func (r *cardRepo) DeleteCard(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, queryDeleteCard, id)
	if err != nil {
		return fmt.Errorf("repo:DeleteCard: %w", err)
	}
	return nil
}
