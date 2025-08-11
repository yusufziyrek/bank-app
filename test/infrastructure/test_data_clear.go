package infrastructure

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ClearTestDatabase test veritabanını temizler
func ClearTestDatabase(ctx context.Context, pool *pgxpool.Pool) error {
	// Users tablosunu temizle
	_, err := pool.Exec(ctx, "DELETE FROM users")
	if err != nil {
		return fmt.Errorf("failed to clear users table: %w", err)
	}

	// Sequence'leri sıfırla
	_, err = pool.Exec(ctx, "ALTER SEQUENCE users_id_seq RESTART WITH 1")
	if err != nil {
		return fmt.Errorf("failed to reset sequence: %w", err)
	}

	log.Println("Test database cleared successfully")
	return nil
}

// ClearSpecificTestData belirli test verilerini temizler
func ClearSpecificTestData(ctx context.Context, pool *pgxpool.Pool, emails []string) error {
	for _, email := range emails {
		_, err := pool.Exec(ctx, "DELETE FROM users WHERE email = $1", email)
		if err != nil {
			return fmt.Errorf("failed to delete user with email %s: %w", email, err)
		}
	}
	return nil
}
