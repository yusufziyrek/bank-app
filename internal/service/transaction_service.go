package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/yusufziyrek/bank-app/internal/model"
	"github.com/yusufziyrek/bank-app/internal/repository"
)

type TransactionService interface {
	GetAllTransactions(ctx context.Context) ([]model.Transaction, error)
	GetTransactionByID(ctx context.Context, id int64) (model.Transaction, error)
	CreateTransaction(ctx context.Context, t *model.Transaction, userID int64) error
	UpdateTransaction(ctx context.Context, t *model.Transaction) error
	DeleteTransaction(ctx context.Context, id int64) error
}

type transactionService struct {
	repo    repository.TransactionRepository
	accRepo repository.AccountRepository
	accSvc  AccountService
}

func NewTransactionService(r repository.TransactionRepository, accRepo repository.AccountRepository, accSvc AccountService) TransactionService {
	return &transactionService{repo: r, accRepo: accRepo, accSvc: accSvc}
}

func (s *transactionService) GetAllTransactions(ctx context.Context) ([]model.Transaction, error) {
	return s.repo.GetAllTransactions(ctx)
}

func (s *transactionService) GetTransactionByID(ctx context.Context, id int64) (model.Transaction, error) {
	t, err := s.repo.GetTransactionByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return t, ErrTransactionNotFound
		}
		return t, fmt.Errorf("service:GetTransactionByID: %w", err)
	}
	return t, nil
}

func (s *transactionService) CreateTransaction(ctx context.Context, t *model.Transaction, userID int64) error {
	const maxTransactionAmount = 10000.0
	updatedAccounts := make(map[int64]model.Account)
	err := s.repo.WithTransaction(ctx, func(tx pgx.Tx) error {
		acc, err := s.accRepo.GetAccountByID(ctx, t.AccountID)
		if err != nil {
			return fmt.Errorf("service:GetAccountByID: %w", err)
		}

		if acc.UserID != userID {
			return errors.New("yetkisiz işlem: hesap size ait değil")
		}

		if t.Type == "transfer" && t.Description == "" {
			return errors.New("açıklama alanı zorunludur")
		}

		if t.Amount <= 0 || t.Amount > maxTransactionAmount {
			return errors.New("işlem tutarı 0'dan büyük ve 10.000'den küçük olmalı")
		}

		switch t.Type {
		case "deposit":
			acc.Balance += t.Amount
			if err := s.accRepo.UpdateAccount(ctx, &acc); err != nil {
				return fmt.Errorf("service:UpdateAccount: %w", err)
			}
			updatedAccounts[acc.ID] = acc
		case "withdraw":
			if acc.Balance < t.Amount {
				return ErrInsufficientFunds
			}
			acc.Balance -= t.Amount
			if err := s.accRepo.UpdateAccount(ctx, &acc); err != nil {
				return fmt.Errorf("service:UpdateAccount: %w", err)
			}
			updatedAccounts[acc.ID] = acc
		case "transfer":
			if acc.Balance < t.Amount {
				return ErrInsufficientFunds
			}
			if t.ToAccountID == nil || *t.ToAccountID == t.AccountID {
				return ErrInvalidTransactionType
			}
			recvAcc, err := s.accRepo.GetAccountByID(ctx, *t.ToAccountID)
			if err != nil {
				return fmt.Errorf("service:GetToAccountByID: %w", err)
			}
			if recvAcc.ID == 0 {
				return ErrAccountNotFound
			}
			if recvAcc.ID == acc.ID {
				return ErrInvalidTransactionType
			}

			acc.Balance -= t.Amount
			recvAcc.Balance += t.Amount
			if acc.Balance < 0 {
				return ErrInsufficientFunds
			}
			if err := s.accRepo.UpdateAccount(ctx, &acc); err != nil {
				return fmt.Errorf("service:UpdateSenderAccount: %w", err)
			}
			if err := s.accRepo.UpdateAccount(ctx, &recvAcc); err != nil {
				return fmt.Errorf("service:UpdateReceiverAccount: %w", err)
			}
			updatedAccounts[acc.ID] = acc
			updatedAccounts[recvAcc.ID] = recvAcc
		default:
			return ErrInvalidTransactionType
		}

		if err := s.repo.AddTransaction(ctx, t); err != nil {
			return fmt.Errorf("service:AddTransaction: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if s.accSvc != nil {
		for _, account := range updatedAccounts {
			s.accSvc.NotifyAccountBalanceChanged(ctx, account)
		}
	}

	return nil
}

func (s *transactionService) UpdateTransaction(ctx context.Context, t *model.Transaction) error {
	if err := s.repo.UpdateTransaction(ctx, t); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTransactionNotFound
		}
		return fmt.Errorf("service:UpdateTransaction: %w", err)
	}
	return nil
}

func (s *transactionService) DeleteTransaction(ctx context.Context, id int64) error {
	if err := s.repo.DeleteTransaction(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTransactionNotFound
		}
		return fmt.Errorf("service:DeleteTransaction: %w", err)
	}
	return nil
}
