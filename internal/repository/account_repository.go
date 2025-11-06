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
	queryGetAllAccounts = `
        SELECT id, user_id, account_number, balance, created_at, updated_at
        FROM accounts
    `
	queryGetAccountByID = `
        SELECT id, user_id, account_number, balance, created_at, updated_at
        FROM accounts WHERE id=$1
    `
	queryGetAccountByIDForUpdate = `
		SELECT id, user_id, account_number, balance, created_at, updated_at
		FROM accounts WHERE id=$1
		FOR UPDATE
	`
	queryAddAccount = `
        INSERT INTO accounts (user_id, account_number, balance, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `
	queryUpdateAccount = `
        UPDATE accounts SET balance=$1, updated_at=$2 WHERE id=$3
    `
	queryDeleteAccount = `
        DELETE FROM accounts WHERE id=$1
    `
	queryGetAccountsByUser = `
        SELECT id, user_id, account_number, balance, created_at, updated_at
        FROM accounts WHERE user_id=$1
    `
)

type AccountRepository interface {
	GetAllAccounts(ctx context.Context) ([]model.Account, error)
	GetAccountByID(ctx context.Context, id int64) (model.Account, error)
	GetAccountByIDForUpdate(ctx context.Context, tx pgx.Tx, id int64) (model.Account, error)
	AddAccount(ctx context.Context, a *model.Account) error
	UpdateAccount(ctx context.Context, a *model.Account) error
	UpdateAccountTx(ctx context.Context, tx pgx.Tx, a *model.Account) error
	DeleteAccount(ctx context.Context, id int64) error
	GetAccountsByUser(ctx context.Context, userID int64) ([]model.Account, error)
	WithTransaction(ctx context.Context, fn func(pgx.Tx) error) error
}

type accountRepo struct {
	pool *pgxpool.Pool
}

func NewAccountRepository(pool *pgxpool.Pool) AccountRepository {
	return &accountRepo{pool: pool}
}

func (r *accountRepo) GetAllAccounts(ctx context.Context) ([]model.Account, error) {
	rows, err := r.pool.Query(ctx, queryGetAllAccounts)
	if err != nil {
		return nil, fmt.Errorf("repo:get all accounts: %w", err)
	}
	defer rows.Close()

	var accounts []model.Account
	for rows.Next() {
		var a model.Account
		if err := rows.Scan(
			&a.ID, &a.UserID, &a.AccountNumber, &a.Balance, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("repo:scan account: %w", err)
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

func (r *accountRepo) GetAccountByID(ctx context.Context, id int64) (model.Account, error) {
	var a model.Account
	err := r.pool.QueryRow(ctx, queryGetAccountByID, id).Scan(
		&a.ID, &a.UserID, &a.AccountNumber, &a.Balance, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return a, fmt.Errorf("repo:get account by id: %w", err)
	}
	return a, nil
}

func (r *accountRepo) GetAccountByIDForUpdate(ctx context.Context, tx pgx.Tx, id int64) (model.Account, error) {
	var a model.Account
	err := tx.QueryRow(ctx, queryGetAccountByIDForUpdate, id).Scan(
		&a.ID, &a.UserID, &a.AccountNumber, &a.Balance, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return a, fmt.Errorf("repo:get account by id for update: %w", err)
	}
	return a, nil
}

func (r *accountRepo) GetAccountsByUser(ctx context.Context, userID int64) ([]model.Account, error) {
	rows, err := r.pool.Query(ctx, queryGetAccountsByUser, userID)
	if err != nil {
		return nil, fmt.Errorf("repo:get accounts by user: %w", err)
	}
	defer rows.Close()

	var accounts []model.Account
	for rows.Next() {
		var a model.Account
		if err := rows.Scan(
			&a.ID, &a.UserID, &a.AccountNumber, &a.Balance, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("repo:scan account: %w", err)
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

func (r *accountRepo) AddAccount(ctx context.Context, a *model.Account) error {
	now := time.Now()
	err := r.pool.QueryRow(
		ctx,
		queryAddAccount,
		a.UserID, a.AccountNumber, a.Balance, now, now,
	).Scan(&a.ID)
	if err != nil {
		return fmt.Errorf("repo:add account: %w", err)
	}
	a.CreatedAt = now
	a.UpdatedAt = now
	return nil
}

func (r *accountRepo) UpdateAccount(ctx context.Context, a *model.Account) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx, queryUpdateAccount, a.Balance, now, a.ID)
	if err != nil {
		return fmt.Errorf("repo:update account: %w", err)
	}
	a.UpdatedAt = now
	return nil
}

func (r *accountRepo) UpdateAccountTx(ctx context.Context, tx pgx.Tx, a *model.Account) error {
	now := time.Now()
	if _, err := tx.Exec(ctx, queryUpdateAccount, a.Balance, now, a.ID); err != nil {
		return fmt.Errorf("repo:update account tx: %w", err)
	}
	a.UpdatedAt = now
	return nil
}

func (r *accountRepo) DeleteAccount(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, queryDeleteAccount, id)
	if err != nil {
		return fmt.Errorf("repo:delete account: %w", err)
	}
	return nil
}

func (r *accountRepo) WithTransaction(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("repo:begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
	}()
	return fn(tx)
}
