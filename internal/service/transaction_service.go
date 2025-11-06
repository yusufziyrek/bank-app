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
	GetTransactionsByAccountIDs(ctx context.Context, accountIDs []int64) ([]model.Transaction, error)
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

func (s *transactionService) GetTransactionsByAccountIDs(ctx context.Context, accountIDs []int64) ([]model.Transaction, error) {
	return s.repo.GetTransactionsByAccountIDs(ctx, accountIDs)
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
		acc, err := s.accRepo.GetAccountByIDForUpdate(ctx, tx, t.AccountID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrAccountNotFound
			}
			return fmt.Errorf("service:GetAccountByID: %w", err)
		}
		if acc.UserID != userID {
			return errors.New("yetkisiz işlem: hesap size ait değil")
		}
		if t.Type == model.TransactionTypeTransfer && t.Description == "" {
			return errors.New("açıklama alanı zorunludur")
		}
		if t.Amount <= 0 || t.Amount > maxTransactionAmount {
			return errors.New("işlem tutarı 0'dan büyük ve 10.000'den küçük olmalı")
		}

		var recvAcc *model.Account
		switch t.Type {
		case model.TransactionTypeDeposit:
			acc.Balance += t.Amount
		case model.TransactionTypeWithdraw:
			if acc.Balance < t.Amount {
				return ErrInsufficientFunds
			}
			acc.Balance -= t.Amount
		case model.TransactionTypeTransfer:
			if t.ToAccountID == nil || *t.ToAccountID == t.AccountID {
				return ErrInvalidTransactionType
			}
			if acc.Balance < t.Amount {
				return ErrInsufficientFunds
			}
			toAcc, err := s.accRepo.GetAccountByIDForUpdate(ctx, tx, *t.ToAccountID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return ErrAccountNotFound
				}
				return fmt.Errorf("service:GetToAccountByID: %w", err)
			}
			if toAcc.ID == acc.ID {
				return ErrInvalidTransactionType
			}
			acc.Balance -= t.Amount
			toAcc.Balance += t.Amount
			recvAcc = &toAcc
		default:
			return ErrInvalidTransactionType
		}

		if err := s.accRepo.UpdateAccountTx(ctx, tx, &acc); err != nil {
			return fmt.Errorf("service:UpdateAccount: %w", err)
		}
		updatedAccounts[acc.ID] = acc
		if recvAcc != nil {
			if err := s.accRepo.UpdateAccountTx(ctx, tx, recvAcc); err != nil {
				return fmt.Errorf("service:UpdateReceiverAccount: %w", err)
			}
			updatedAccounts[recvAcc.ID] = *recvAcc
		}

		if err := s.repo.AddTransactionTx(ctx, tx, t); err != nil {
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
	const maxTransactionAmount = 10000.0
	updatedAccounts := make(map[int64]model.Account)

	err := s.repo.WithTransaction(ctx, func(tx pgx.Tx) error {
		existing, err := s.repo.GetTransactionByIDForUpdate(ctx, tx, t.ID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrTransactionNotFound
			}
			return fmt.Errorf("service:GetTransactionByID: %w", err)
		}

		sourceAcc, err := s.accRepo.GetAccountByIDForUpdate(ctx, tx, existing.AccountID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrAccountNotFound
			}
			return fmt.Errorf("service:GetAccountByID: %w", err)
		}

		var oldDestAcc *model.Account
		if existing.Type == model.TransactionTypeTransfer && existing.ToAccountID != nil {
			acc, err := s.accRepo.GetAccountByIDForUpdate(ctx, tx, *existing.ToAccountID)
			if err != nil {
				return fmt.Errorf("service:GetExistingToAccount: %w", err)
			}
			oldDestAcc = &acc
		}

		// Revert previous transaction effect
		switch existing.Type {
		case model.TransactionTypeDeposit:
			sourceAcc.Balance -= existing.Amount
		case model.TransactionTypeWithdraw:
			sourceAcc.Balance += existing.Amount
		case model.TransactionTypeTransfer:
			sourceAcc.Balance += existing.Amount
			if oldDestAcc != nil {
				oldDestAcc.Balance -= existing.Amount
			}
		}

		// Apply new transaction
		if t.Type == model.TransactionTypeTransfer && t.Description == "" {
			return errors.New("açıklama alanı zorunludur")
		}
		if t.Amount <= 0 || t.Amount > maxTransactionAmount {
			return errors.New("işlem tutarı 0'dan büyük ve 10.000'den küçük olmalı")
		}

		var newDestAcc *model.Account
		switch t.Type {
		case model.TransactionTypeDeposit:
			sourceAcc.Balance += t.Amount
			t.ToAccountID = nil
		case model.TransactionTypeWithdraw:
			if sourceAcc.Balance < t.Amount {
				return ErrInsufficientFunds
			}
			sourceAcc.Balance -= t.Amount
			t.ToAccountID = nil
		case model.TransactionTypeTransfer:
			if t.ToAccountID == nil || *t.ToAccountID == existing.AccountID {
				return ErrInvalidTransactionType
			}
			if sourceAcc.Balance < t.Amount {
				return ErrInsufficientFunds
			}
			destAcc, err := s.accRepo.GetAccountByIDForUpdate(ctx, tx, *t.ToAccountID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return ErrAccountNotFound
				}
				return fmt.Errorf("service:GetNewToAccount: %w", err)
			}
			if destAcc.ID == existing.AccountID {
				return ErrInvalidTransactionType
			}
			sourceAcc.Balance -= t.Amount
			destAcc.Balance += t.Amount
			newDestAcc = &destAcc
		default:
			return ErrInvalidTransactionType
		}

		if err := s.accRepo.UpdateAccountTx(ctx, tx, &sourceAcc); err != nil {
			return fmt.Errorf("service:UpdateSourceAccount: %w", err)
		}
		updatedAccounts[sourceAcc.ID] = sourceAcc

		if oldDestAcc != nil {
			if err := s.accRepo.UpdateAccountTx(ctx, tx, oldDestAcc); err != nil {
				return fmt.Errorf("service:UpdateOldDestAccount: %w", err)
			}
			updatedAccounts[oldDestAcc.ID] = *oldDestAcc
		}

		if newDestAcc != nil {
			if err := s.accRepo.UpdateAccountTx(ctx, tx, newDestAcc); err != nil {
				return fmt.Errorf("service:UpdateNewDestAccount: %w", err)
			}
			updatedAccounts[newDestAcc.ID] = *newDestAcc
		}

		t.AccountID = existing.AccountID
		if err := s.repo.UpdateTransactionTx(ctx, tx, t); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrTransactionNotFound
			}
			return fmt.Errorf("service:UpdateTransaction: %w", err)
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

func (s *transactionService) DeleteTransaction(ctx context.Context, id int64) error {
	updatedAccounts := make(map[int64]model.Account)

	err := s.repo.WithTransaction(ctx, func(tx pgx.Tx) error {
		existing, err := s.repo.GetTransactionByIDForUpdate(ctx, tx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrTransactionNotFound
			}
			return fmt.Errorf("service:GetTransactionByID: %w", err)
		}

		sourceAcc, err := s.accRepo.GetAccountByIDForUpdate(ctx, tx, existing.AccountID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrAccountNotFound
			}
			return fmt.Errorf("service:GetAccountByID: %w", err)
		}

		var destAcc *model.Account
		if existing.Type == model.TransactionTypeTransfer && existing.ToAccountID != nil {
			acc, err := s.accRepo.GetAccountByIDForUpdate(ctx, tx, *existing.ToAccountID)
			if err != nil {
				return fmt.Errorf("service:GetToAccountByID: %w", err)
			}
			destAcc = &acc
		}

		switch existing.Type {
		case model.TransactionTypeDeposit:
			sourceAcc.Balance -= existing.Amount
		case model.TransactionTypeWithdraw:
			sourceAcc.Balance += existing.Amount
		case model.TransactionTypeTransfer:
			sourceAcc.Balance += existing.Amount
			if destAcc != nil {
				destAcc.Balance -= existing.Amount
			}
		}

		if err := s.accRepo.UpdateAccountTx(ctx, tx, &sourceAcc); err != nil {
			return fmt.Errorf("service:UpdateSourceAccount: %w", err)
		}
		updatedAccounts[sourceAcc.ID] = sourceAcc
		if destAcc != nil {
			if err := s.accRepo.UpdateAccountTx(ctx, tx, destAcc); err != nil {
				return fmt.Errorf("service:UpdateDestAccount: %w", err)
			}
			updatedAccounts[destAcc.ID] = *destAcc
		}

		if err := s.repo.DeleteTransactionTx(ctx, tx, id); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrTransactionNotFound
			}
			return fmt.Errorf("service:DeleteTransaction: %w", err)
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
