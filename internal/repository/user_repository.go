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
}

type userRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepo{pool: pool}
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

	if errors.Is(err, pgx.ErrNoRows) {
		return user, pgx.ErrNoRows
	} else if err != nil {
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
