package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yusufziyrek/bank-app/internal/model"
)

const (
	queryGetAllUsers = `
        SELECT id, full_name, email, password_hash, role, is_active, created_at, updated_at 
        FROM users
    `
	queryGetUserByID = `
        SELECT id, full_name, email, password_hash, role, is_active, created_at, updated_at
        FROM users WHERE id=$1
    `
	queryGetUserByEmail = `
        SELECT id, full_name, email, password_hash, role, is_active, created_at, updated_at
        FROM users WHERE email=$1
    `
	queryAddUser = `
        INSERT INTO users
            (full_name, email, password_hash, role, is_active, created_at, updated_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7)
        RETURNING id
    `
	queryUpdateUserEmail = `
        UPDATE users SET email=$1, updated_at=$2 WHERE id=$3
    `
	queryUpdateUserPassword = `
        UPDATE users SET password_hash=$1, updated_at=$2 WHERE id=$3
    `
	queryUpdateUserActiveStatus = `
        UPDATE users SET is_active=$1, updated_at=$2 WHERE id=$3
    `
	queryDeleteUserByID = `
        DELETE FROM users WHERE id=$1
    `
	queryInsertRefreshToken = `
		INSERT INTO refresh_tokens (user_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	queryGetRefreshToken = `
		SELECT id, user_id, token, expires_at, created_at FROM refresh_tokens WHERE token=$1
	`
	queryDeleteRefreshToken = `
		DELETE FROM refresh_tokens WHERE token=$1
	`
	queryDeleteUserRefreshTokens = `
		DELETE FROM refresh_tokens WHERE user_id=$1
	`
)

type UserRepository interface {
	GetAllUsers(ctx context.Context) ([]model.User, error)
	GetUserByID(ctx context.Context, id int64) (model.User, error)
	GetUserByEmail(ctx context.Context, email string) (model.User, error)
	AddUser(ctx context.Context, u *model.User) error
	UpdateUserEmail(ctx context.Context, id int64, email string) error
	UpdateUserPassword(ctx context.Context, id int64, hash string) error
	UpdateUserActiveStatus(ctx context.Context, id int64, isActive bool) error
	DeleteUserByID(ctx context.Context, id int64) error

	// Transaction support for future complex operations
	WithTransaction(ctx context.Context, fn func(pgx.Tx) error) error
	InsertRefreshToken(ctx context.Context, rt *model.RefreshToken) error
	GetRefreshToken(ctx context.Context, token string) (model.RefreshToken, error)
	DeleteRefreshToken(ctx context.Context, token string) error
	DeleteUserRefreshTokens(ctx context.Context, userID int64) error
}

type userRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepo{pool: pool}
}

// WithTransaction executes a function within a database transaction
func (r *userRepo) WithTransaction(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("repo:begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			// A panic occurred, rollback and re-panic
			tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			// Something went wrong, rollback
			tx.Rollback(ctx)
		} else {
			// All good, commit
			err = tx.Commit(ctx)
		}
	}()

	err = fn(tx)
	return err
}

func (r *userRepo) GetAllUsers(ctx context.Context) ([]model.User, error) {
	rows, err := r.pool.Query(ctx, queryGetAllUsers)
	if err != nil {
		return nil, fmt.Errorf("repo:GetAllUsers:query: %w", err)
	}
	defer rows.Close()
	users, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.User])
	if err != nil {
		return nil, fmt.Errorf("repo:GetAllUsers:scan: %w", err)
	}
	return users, nil
}

func (r *userRepo) GetUserByID(ctx context.Context, id int64) (model.User, error) {
	var user model.User
	err := r.pool.QueryRow(ctx, queryGetUserByID, id).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return user, pgx.ErrNoRows
	} else if err != nil {
		return user, fmt.Errorf("repo:GetUserByID: %w", err)
	}
	return user, nil
}

func (r *userRepo) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	var user model.User
	err := r.pool.QueryRow(ctx, queryGetUserByEmail, email).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return user, pgx.ErrNoRows
		}
		return user, fmt.Errorf("repo:GetUserByEmail: %w", err)
	}
	return user, nil
}

func (r *userRepo) AddUser(ctx context.Context, u *model.User) error {
	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now

	err := r.pool.QueryRow(ctx, queryAddUser, u.FullName, u.Email, u.PasswordHash, u.Role, u.IsActive, u.CreatedAt, u.UpdatedAt).
		Scan(&u.ID)
	if err != nil {
		return fmt.Errorf("repo:AddUser: %w", err)
	}
	return nil
}

func (r *userRepo) UpdateUserEmail(ctx context.Context, id int64, email string) error {
	cmd, err := r.pool.Exec(ctx, queryUpdateUserEmail, email, time.Now(), id)
	if err != nil {
		return fmt.Errorf("repo:UpdateEmail: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *userRepo) UpdateUserPassword(ctx context.Context, id int64, hash string) error {
	cmd, err := r.pool.Exec(ctx, queryUpdateUserPassword, hash, time.Now(), id)
	if err != nil {
		return fmt.Errorf("repo:UpdatePassword: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *userRepo) UpdateUserActiveStatus(ctx context.Context, id int64, isActive bool) error {
	cmd, err := r.pool.Exec(ctx, queryUpdateUserActiveStatus, isActive, time.Now(), id)
	if err != nil {
		return fmt.Errorf("repo:UpdateStatus: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *userRepo) DeleteUserByID(ctx context.Context, id int64) error {
	cmd, err := r.pool.Exec(ctx, queryDeleteUserByID, id)
	if err != nil {
		return fmt.Errorf("repo:DeleteUser: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *userRepo) InsertRefreshToken(ctx context.Context, rt *model.RefreshToken) error {
	err := r.pool.QueryRow(ctx, queryInsertRefreshToken, rt.UserID, rt.Token, rt.ExpiresAt, rt.CreatedAt).Scan(&rt.ID)
	if err != nil {
		return fmt.Errorf("repo:InsertRefreshToken: %w", err)
	}
	return nil
}

func (r *userRepo) GetRefreshToken(ctx context.Context, token string) (model.RefreshToken, error) {
	var rt model.RefreshToken
	err := r.pool.QueryRow(ctx, queryGetRefreshToken, token).Scan(
		&rt.ID, &rt.UserID, &rt.Token, &rt.ExpiresAt, &rt.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return rt, pgx.ErrNoRows
	} else if err != nil {
		return rt, fmt.Errorf("repo:GetRefreshToken: %w", err)
	}
	return rt, nil
}

func (r *userRepo) DeleteRefreshToken(ctx context.Context, token string) error {
	_, err := r.pool.Exec(ctx, queryDeleteRefreshToken, token)
	if err != nil {
		return fmt.Errorf("repo:DeleteRefreshToken: %w", err)
	}
	return nil
}

func (r *userRepo) DeleteUserRefreshTokens(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx, queryDeleteUserRefreshTokens, userID)
	if err != nil {
		return fmt.Errorf("repo:DeleteUserRefreshTokens: %w", err)
	}
	return nil
}
