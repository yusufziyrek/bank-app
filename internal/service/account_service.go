package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"github.com/jackc/pgx/v5"
	"github.com/yusufziyrek/bank-app/internal/model"
	"github.com/yusufziyrek/bank-app/internal/repository"
)

type AccountService interface {
	GetAllAccounts(ctx context.Context) ([]model.Account, error)
	GetAccountByID(ctx context.Context, id int64) (model.Account, error)
	GetAccountsByUser(ctx context.Context, userID int64) ([]model.Account, error)
	CreateAccount(ctx context.Context, a *model.Account) error
	UpdateAccount(ctx context.Context, a *model.Account) error
	DeleteAccount(ctx context.Context, id int64) error
}

type accountService struct {
	repo repository.AccountRepository
}

func NewAccountService(r repository.AccountRepository) AccountService {
	return &accountService{repo: r}
}

func (s *accountService) GetAllAccounts(ctx context.Context) ([]model.Account, error) {
	return s.repo.GetAllAccounts(ctx)
}

func (s *accountService) GetAccountByID(ctx context.Context, id int64) (model.Account, error) {
	a, err := s.repo.GetAccountByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return a, ErrAccountNotFound
		}
		return a, fmt.Errorf("service:GetAccountByID: %w", err)
	}
	return a, nil
}

func (s *accountService) GetAccountsByUser(ctx context.Context, userID int64) ([]model.Account, error) {
	return s.repo.GetAccountsByUser(ctx, userID)
}

func (s *accountService) CreateAccount(ctx context.Context, a *model.Account) error {
	existing, err := s.repo.GetAccountsByUser(ctx, a.UserID)
	if err != nil {
		return fmt.Errorf("service:getAccountsByUser: %w", err)
	}
	if len(existing) >= 3 {
		return ErrMaxAccountsExceeded
	}

	num, err := generateAccountNumber()
	if err != nil {
		return fmt.Errorf("service:generateAccountNumber: %w", err)
	}
	a.AccountNumber = num

	if err := s.repo.AddAccount(ctx, a); err != nil {
		return fmt.Errorf("service:AddAccount: %w", err)
	}
	return nil
}

func (s *accountService) UpdateAccount(ctx context.Context, a *model.Account) error {
	acc, err := s.repo.GetAccountByID(ctx, a.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAccountNotFound
		}
		return fmt.Errorf("service:GetAccountByID: %w", err)
	}
	acc.Balance = a.Balance
	if err := s.repo.UpdateAccount(ctx, &acc); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAccountNotFound
		}
		return fmt.Errorf("service:UpdateAccount: %w", err)
	}
	return nil
}

func (s *accountService) DeleteAccount(ctx context.Context, id int64) error {
	if err := s.repo.DeleteAccount(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAccountNotFound
		}
		return fmt.Errorf("service:DeleteAccount: %w", err)
	}
	return nil
}

func generateAccountNumber() (string, error) {
	b := make([]byte, 10)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("TR%020d", new(big.Int).SetBytes(b)), nil
}
