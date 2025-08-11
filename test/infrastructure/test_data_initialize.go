package infrastructure

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yusufziyrek/bank-app/internal/model"
	"golang.org/x/crypto/bcrypt"
)

// TestDatabaseConfig test veritabanı konfigürasyonu
type TestDatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

// InitializeTestDatabase test veritabanını başlatır
func InitializeTestDatabase(ctx context.Context, pool *pgxpool.Pool) error {
	// Users tablosunu oluştur
	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id BIGSERIAL PRIMARY KEY,
		full_name VARCHAR(100) NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		role VARCHAR(50) DEFAULT 'user',
		is_active BOOLEAN DEFAULT true,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := pool.Exec(ctx, createUsersTable)
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	// Test verilerini ekle
	return insertTestData(ctx, pool)
}

// insertTestData test verilerini ekler
func insertTestData(ctx context.Context, pool *pgxpool.Pool) error {
	// Test kullanıcıları
	testUsers := []struct {
		fullName  string
		email     string
		password  string
		role      string
		isActive  bool
		createdAt time.Time
		updatedAt time.Time
	}{
		{
			fullName:  "Test User 1",
			email:     "test1@example.com",
			password:  "password123",
			role:      "user",
			isActive:  true,
			createdAt: time.Now(),
			updatedAt: time.Now(),
		},
		{
			fullName:  "Test User 2",
			email:     "test2@example.com",
			password:  "password456",
			role:      "admin",
			isActive:  true,
			createdAt: time.Now(),
			updatedAt: time.Now(),
		},
		{
			fullName:  "Inactive User",
			email:     "inactive@example.com",
			password:  "password789",
			role:      "user",
			isActive:  false,
			createdAt: time.Now(),
			updatedAt: time.Now(),
		},
	}

	for _, user := range testUsers {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}

		_, err = pool.Exec(ctx, `
			INSERT INTO users (full_name, email, password_hash, role, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (email) DO NOTHING
		`, user.fullName, user.email, string(hashedPassword), user.role, user.isActive, user.createdAt, user.updatedAt)

		if err != nil {
			return fmt.Errorf("failed to insert test user: %w", err)
		}
	}

	log.Println("Test data initialized successfully")
	return nil
}

// GetTestUserByEmail test kullanıcısını email ile getirir
func GetTestUserByEmail(ctx context.Context, pool *pgxpool.Pool, email string) (*model.User, error) {
	var user model.User
	err := pool.QueryRow(ctx, `
		SELECT id, full_name, email, password_hash, role, is_active, created_at, updated_at
		FROM users WHERE email = $1
	`, email).Scan(
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
		return nil, err
	}

	return &user, nil
}
