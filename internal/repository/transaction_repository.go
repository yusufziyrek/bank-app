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
	queryGetAllTransactions = `
		SELECT id, account_id, to_account_id, amount, type, description, created_at
		FROM transactions
	`
	queryGetTransactionByID = `
		SELECT id, account_id, to_account_id, amount, type, description, created_at
		FROM transactions WHERE id=$1
	`
	queryGetTransactionByIDForUpdate = `
		SELECT id, account_id, to_account_id, amount, type, description, created_at
		FROM transactions WHERE id=$1
		FOR UPDATE
	`
	queryAddTransaction = `
		INSERT INTO transactions (account_id, to_account_id, amount, type, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	queryUpdateTransaction = `
		UPDATE transactions SET to_account_id=$1, amount=$2, type=$3, description=$4 WHERE id=$5
	`
	queryDeleteTransaction = `
		DELETE FROM transactions WHERE id=$1
	`
	queryGetTransactionsByAccountIDs = `
		SELECT id, account_id, to_account_id, amount, type, description, created_at
		FROM transactions
		WHERE account_id = ANY($1)
	`
)

type TransactionRepository interface {
	GetAllTransactions(ctx context.Context) ([]model.Transaction, error)
	GetTransactionsByAccountIDs(ctx context.Context, accountIDs []int64) ([]model.Transaction, error)
	GetTransactionByID(ctx context.Context, id int64) (model.Transaction, error)
	GetTransactionByIDForUpdate(ctx context.Context, tx pgx.Tx, id int64) (model.Transaction, error)
	AddTransaction(ctx context.Context, t *model.Transaction) error
	AddTransactionTx(ctx context.Context, tx pgx.Tx, t *model.Transaction) error
	UpdateTransaction(ctx context.Context, t *model.Transaction) error
	UpdateTransactionTx(ctx context.Context, tx pgx.Tx, t *model.Transaction) error
	DeleteTransaction(ctx context.Context, id int64) error
	DeleteTransactionTx(ctx context.Context, tx pgx.Tx, id int64) error
	WithTransaction(ctx context.Context, fn func(pgx.Tx) error) error
}

type transactionRepo struct {
	pool *pgxpool.Pool
}

func NewTransactionRepository(pool *pgxpool.Pool) TransactionRepository {
	return &transactionRepo{pool: pool}
}

func (r *transactionRepo) GetAllTransactions(ctx context.Context) ([]model.Transaction, error) {
	rows, err := r.pool.Query(ctx, queryGetAllTransactions)
	if err != nil {
		return nil, fmt.Errorf("repo:get all transactions: %w", err)
	}
	defer rows.Close()

	var transactions []model.Transaction
	for rows.Next() {
		var t model.Transaction
		if err := rows.Scan(&t.ID, &t.AccountID, &t.ToAccountID, &t.Amount, &t.Type, &t.Description, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("repo:scan transaction: %w", err)
		}
		transactions = append(transactions, t)
	}
	return transactions, nil
}

func (r *transactionRepo) GetTransactionsByAccountIDs(ctx context.Context, accountIDs []int64) ([]model.Transaction, error) {
	if len(accountIDs) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, queryGetTransactionsByAccountIDs, accountIDs)
	if err != nil {
		return nil, fmt.Errorf("repo:get transactions by accounts: %w", err)
	}
	defer rows.Close()

	var transactions []model.Transaction
	for rows.Next() {
		var t model.Transaction
		if err := rows.Scan(&t.ID, &t.AccountID, &t.ToAccountID, &t.Amount, &t.Type, &t.Description, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("repo:scan transaction: %w", err)
		}
		transactions = append(transactions, t)
	}
	return transactions, nil
}

func (r *transactionRepo) GetTransactionByID(ctx context.Context, id int64) (model.Transaction, error) {
	var t model.Transaction
	err := r.pool.QueryRow(ctx, queryGetTransactionByID, id).Scan(
		&t.ID, &t.AccountID, &t.ToAccountID, &t.Amount, &t.Type, &t.Description, &t.CreatedAt,
	)
	if err != nil {
		return t, fmt.Errorf("repo:get transaction by id: %w", err)
	}
	return t, nil
}

func (r *transactionRepo) GetTransactionByIDForUpdate(ctx context.Context, tx pgx.Tx, id int64) (model.Transaction, error) {
	var t model.Transaction
	err := tx.QueryRow(ctx, queryGetTransactionByIDForUpdate, id).Scan(
		&t.ID, &t.AccountID, &t.ToAccountID, &t.Amount, &t.Type, &t.Description, &t.CreatedAt,
	)
	if err != nil {
		return t, fmt.Errorf("repo:get transaction by id for update: %w", err)
	}
	return t, nil
}

func (r *transactionRepo) AddTransaction(ctx context.Context, t *model.Transaction) error {
	now := time.Now()
	var toAccountID interface{} = nil
	if t.ToAccountID != nil {
		toAccountID = *t.ToAccountID
	}
	err := r.pool.QueryRow(ctx, queryAddTransaction, t.AccountID, toAccountID, t.Amount, t.Type, t.Description, now).Scan(&t.ID)
	if err != nil {
		return fmt.Errorf("repo:add transaction: %w", err)
	}
	t.CreatedAt = now
	return nil
}

func (r *transactionRepo) AddTransactionTx(ctx context.Context, tx pgx.Tx, t *model.Transaction) error {
	now := time.Now()
	var toAccountID interface{} = nil
	if t.ToAccountID != nil {
		toAccountID = *t.ToAccountID
	}
	err := tx.QueryRow(ctx, queryAddTransaction, t.AccountID, toAccountID, t.Amount, t.Type, t.Description, now).Scan(&t.ID)
	if err != nil {
		return fmt.Errorf("repo:add transaction tx: %w", err)
	}
	t.CreatedAt = now
	return nil
}

func (r *transactionRepo) UpdateTransaction(ctx context.Context, t *model.Transaction) error {
	_, err := r.pool.Exec(ctx, queryUpdateTransaction, t.ToAccountID, t.Amount, t.Type, t.Description, t.ID)
	if err != nil {
		return fmt.Errorf("repo:update transaction: %w", err)
	}
	return nil
}

func (r *transactionRepo) UpdateTransactionTx(ctx context.Context, tx pgx.Tx, t *model.Transaction) error {
	_, err := tx.Exec(ctx, queryUpdateTransaction, t.ToAccountID, t.Amount, t.Type, t.Description, t.ID)
	if err != nil {
		return fmt.Errorf("repo:update transaction tx: %w", err)
	}
	return nil
}

func (r *transactionRepo) DeleteTransaction(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, queryDeleteTransaction, id)
	if err != nil {
		return fmt.Errorf("repo:delete transaction: %w", err)
	}
	return nil
}

func (r *transactionRepo) DeleteTransactionTx(ctx context.Context, tx pgx.Tx, id int64) error {
	_, err := tx.Exec(ctx, queryDeleteTransaction, id)
	if err != nil {
		return fmt.Errorf("repo:delete transaction tx: %w", err)
	}
	return nil
}

func (r *transactionRepo) WithTransaction(ctx context.Context, fn func(pgx.Tx) error) (err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("repo:begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			err = fmt.Errorf("repo:commit transaction: %w", commitErr)
		}
	}()

	err = fn(tx)
	return err
}
