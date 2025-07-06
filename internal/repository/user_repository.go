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

type UserRepository interface {
	GetAllUsers(ctx context.Context) ([]model.User, error)
	GetUserByID(ctx context.Context, userID int64) (model.User, error)
	GetUserByEmail(ctx context.Context, email string) (model.User, error)
	AddUser(ctx context.Context, user *model.User) error
	UpdateUserEmail(ctx context.Context, userID int64, newEmail string) error
	UpdateUserPassword(ctx context.Context, userID int64, newPasswordHash string) error
	UpdateUserActiveStatus(ctx context.Context, userID int64, isActive bool) error
	DeleteUserByID(ctx context.Context, userID int64) error
}

type userRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepo{pool: pool}
}

func (r *userRepo) GetAllUsers(ctx context.Context) ([]model.User, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, full_name, email, password_hash, role, is_active, created_at, updated_at FROM users`,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: query all users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(
			&u.ID,
			&u.FullName,
			&u.Email,
			&u.PasswordHash,
			&u.Role,
			&u.IsActive,
			&u.CreatedAt,
			&u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("repository: scan user row: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: rows iteration error: %w", err)
	}
	return users, nil
}

func (r *userRepo) GetUserByID(ctx context.Context, userID int64) (model.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, full_name, email, password_hash, role, is_active, created_at, updated_at FROM users WHERE id=$1`, userID)
	var u model.User
	err := row.Scan(
		&u.ID,
		&u.FullName,
		&u.Email,
		&u.PasswordHash,
		&u.Role,
		&u.IsActive,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, fmt.Errorf("repository: user with ID %d not found", userID)
	}
	if err != nil {
		return model.User{}, fmt.Errorf("repository: scan user by ID: %w", err)
	}
	return u, nil
}

func (r *userRepo) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, full_name, email, password_hash, role, is_active, created_at, updated_at FROM users WHERE email=$1`, email)
	var u model.User
	err := row.Scan(
		&u.ID,
		&u.FullName,
		&u.Email,
		&u.PasswordHash,
		&u.Role,
		&u.IsActive,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, fmt.Errorf("repository: user with email %s not found", email)
	}
	if err != nil {
		return model.User{}, fmt.Errorf("repository: scan user by email: %w", err)
	}
	return u, nil
}

func (r *userRepo) AddUser(ctx context.Context, user *model.User) error {
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	query := `
		INSERT INTO users (full_name, email, password_hash, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	err := r.pool.QueryRow(ctx, query,
		user.FullName, user.Email, user.PasswordHash, user.Role, user.IsActive, user.CreatedAt, user.UpdatedAt,
	).Scan(&user.ID)
	if err != nil {
		return fmt.Errorf("repository: AddUser: insert user: %w", err)
	}
	return nil
}

func (r *userRepo) UpdateUserEmail(ctx context.Context, userID int64, newEmail string) error {
	cmdTag, err := r.pool.Exec(ctx,
		`UPDATE users SET email=$1, updated_at=NOW() WHERE id=$2`, newEmail, userID,
	)
	if err != nil {
		return fmt.Errorf("repository: update user email: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("repository: user with ID %d not found for update", userID)
	}
	return nil
}

func (r *userRepo) UpdateUserPassword(ctx context.Context, userID int64, newPasswordHash string) error {
	cmdTag, err := r.pool.Exec(ctx,
		`UPDATE users SET password_hash=$1, updated_at=NOW() WHERE id=$2`, newPasswordHash, userID,
	)
	if err != nil {
		return fmt.Errorf("repository: update user password: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("repository: user with ID %d not found for password update", userID)
	}
	return nil
}

func (r *userRepo) UpdateUserActiveStatus(ctx context.Context, userID int64, isActive bool) error {
	cmdTag, err := r.pool.Exec(ctx,
		`UPDATE users SET is_active=$1, updated_at=NOW() WHERE id=$2`, isActive, userID,
	)
	if err != nil {
		return fmt.Errorf("repository: update user active status: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("repository: user with ID %d not found for active status update", userID)
	}
	return nil
}

func (r *userRepo) DeleteUserByID(ctx context.Context, userID int64) error {
	cmdTag, err := r.pool.Exec(ctx, "DELETE FROM users WHERE id=$1", userID)
	if err != nil {
		return fmt.Errorf("repository: delete user: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("repository: user with ID %d not found for deletion", userID)
	}
	return nil
}
