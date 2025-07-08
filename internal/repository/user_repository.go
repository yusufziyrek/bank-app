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
	rows, err := r.pool.Query(ctx, `
        SELECT id, full_name, email, password_hash, role, is_active, created_at, updated_at
        FROM users
    `)
	if err != nil {
		return nil, fmt.Errorf("repo:GetAllUsers: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(
			&u.ID, &u.FullName, &u.Email, &u.PasswordHash,
			&u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("repo:scan user: %w", err)
		}
		users = append(users, u)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("repo:rows err: %w", rows.Err())
	}
	return users, nil
}

func (r *userRepo) GetUserByID(ctx context.Context, id int64) (model.User, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, full_name, email, password_hash, role, is_active, created_at, updated_at
        FROM users WHERE id=$1
    `, id)
	var u model.User
	err := row.Scan(
		&u.ID, &u.FullName, &u.Email, &u.PasswordHash,
		&u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, pgx.ErrNoRows
	} else if err != nil {
		return u, fmt.Errorf("repo:GetUserByID: %w", err)
	}
	return u, nil
}

func (r *userRepo) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, full_name, email, password_hash, role, is_active, created_at, updated_at
        FROM users WHERE email=$1
    `, email)
	var u model.User
	err := row.Scan(
		&u.ID, &u.FullName, &u.Email, &u.PasswordHash,
		&u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, pgx.ErrNoRows
	} else if err != nil {
		return u, fmt.Errorf("repo:GetUserByEmail: %w", err)
	}
	return u, nil
}

func (r *userRepo) AddUser(ctx context.Context, u *model.User) error {
	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now

	err := r.pool.QueryRow(ctx, `
        INSERT INTO users
            (full_name, email, password_hash, role, is_active, created_at, updated_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7)
        RETURNING id
    `, u.FullName, u.Email, u.PasswordHash, u.Role, u.IsActive, u.CreatedAt, u.UpdatedAt).
		Scan(&u.ID)
	if err != nil {
		return fmt.Errorf("repo:AddUser: %w", err)
	}
	return nil
}

func (r *userRepo) UpdateUserEmail(ctx context.Context, id int64, email string) error {
	cmd, err := r.pool.Exec(ctx, `
        UPDATE users SET email=$1, updated_at=$2 WHERE id=$3
    `, email, time.Now(), id)
	if err != nil {
		return fmt.Errorf("repo:UpdateEmail: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *userRepo) UpdateUserPassword(ctx context.Context, id int64, hash string) error {
	cmd, err := r.pool.Exec(ctx, `
        UPDATE users SET password_hash=$1, updated_at=$2 WHERE id=$3
    `, hash, time.Now(), id)
	if err != nil {
		return fmt.Errorf("repo:UpdatePassword: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *userRepo) UpdateUserActiveStatus(ctx context.Context, id int64, isActive bool) error {
	cmd, err := r.pool.Exec(ctx, `
        UPDATE users SET is_active=$1, updated_at=$2 WHERE id=$3
    `, isActive, time.Now(), id)
	if err != nil {
		return fmt.Errorf("repo:UpdateStatus: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *userRepo) DeleteUserByID(ctx context.Context, id int64) error {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("repo:DeleteUser: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
