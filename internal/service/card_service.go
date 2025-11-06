package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/yusufziyrek/bank-app/internal/model"
	"github.com/yusufziyrek/bank-app/internal/repository"
	"golang.org/x/crypto/bcrypt"
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
	return s.sanitizeCards(cards)
}

type cardService struct {
	repo   repository.CardRepository
	encKey []byte
}

func NewCardService(r repository.CardRepository, encryptionSecret string) CardService {
	return &cardService{
		repo:   r,
		encKey: deriveCardKey(encryptionSecret),
	}
}

func deriveCardKey(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

func hashCVV(cvv string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(cvv), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func (s *cardService) encryptCardNumber(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	block, err := aes.NewCipher(s.encKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *cardService) decryptCardNumber(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(s.encKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (s *cardService) sanitizeCard(card *model.Card) error {
	if card == nil {
		return nil
	}
	if card.CardNumber != "" {
		plain, err := s.decryptCardNumber(card.CardNumber)
		if err != nil {
			return err
		}
		card.CardNumber = plain
	}
	card.CVV = ""
	return nil
}

func (s *cardService) sanitizeCards(cards []model.Card) ([]model.Card, error) {
	for i := range cards {
		if err := s.sanitizeCard(&cards[i]); err != nil {
			return nil, err
		}
	}
	return cards, nil
}

func (s *cardService) GetAllCards(ctx context.Context) ([]model.Card, error) {
	cards, err := s.repo.GetAllCards(ctx)
	if err != nil {
		return nil, fmt.Errorf("service:GetAllCards: %w", err)
	}
	return s.sanitizeCards(cards)
}

func (s *cardService) GetCardByID(ctx context.Context, id int64) (model.Card, error) {
	c, err := s.repo.GetCardById(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Card{}, ErrCardNotFound
		}
		return model.Card{}, fmt.Errorf("service:GetCardByID: %w", err)
	}
	if err := s.sanitizeCard(&c); err != nil {
		return model.Card{}, fmt.Errorf("service:GetCardByID:sanitize: %w", err)
	}
	return c, nil
}

func (s *cardService) GetCardsByUser(ctx context.Context, userID int64) ([]model.Card, error) {
	cards, err := s.repo.GetCardsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("service:GetCardsByUser: %w", err)
	}
	return s.sanitizeCards(cards)
}

func (s *cardService) CreateCard(ctx context.Context, card *model.Card) error {
	// Limit cards per account to 3
	cards, err := s.repo.GetCardsByAccount(ctx, card.AccountID)
	if err != nil {
		return fmt.Errorf("service:CreateCard:getCardsByAccount: %w", err)
	}
	if len(cards) >= 3 {
		return ErrMaxCardsPerAccount
	}
	plainPAN := card.CardNumber
	encryptedPAN, err := s.encryptCardNumber(plainPAN)
	if err != nil {
		return fmt.Errorf("service:CreateCard:encryptPAN: %w", err)
	}
	hashedCVV, err := hashCVV(card.CVV)
	if err != nil {
		return fmt.Errorf("service:CreateCard:hashCVV: %w", err)
	}
	card.CardNumber = encryptedPAN
	card.CVV = hashedCVV
	if !card.IsActive {
		card.IsActive = true
	}
	err = s.repo.AddCard(ctx, card)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrCardAlreadyExists
		}
		return fmt.Errorf("service:CreateCard: %w", err)
	}
	card.CVV = ""
	card.CardNumber = plainPAN
	return nil
}

func (s *cardService) UpdateCard(ctx context.Context, card *model.Card) error {
	storedCard, err := s.repo.GetCardById(ctx, card.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCardNotFound
		}
		return fmt.Errorf("service:UpdateCard:get: %w", err)
	}
	updated := storedCard
	if card.CardNumber != "" {
		encrypted, err := s.encryptCardNumber(card.CardNumber)
		if err != nil {
			return fmt.Errorf("service:UpdateCard:encryptPAN: %w", err)
		}
		updated.CardNumber = encrypted
	}
	if card.CVV != "" {
		hashed, err := hashCVV(card.CVV)
		if err != nil {
			return fmt.Errorf("service:UpdateCard:hashCVV: %w", err)
		}
		updated.CVV = hashed
	}
	if !card.ExpiryDate.IsZero() {
		updated.ExpiryDate = card.ExpiryDate
	}
	err = s.repo.UpdateCard(ctx, &updated)
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
