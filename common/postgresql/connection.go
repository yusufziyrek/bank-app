package postgresql

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func GetConnectionPool(ctx context.Context, config Config) (*pgxpool.Pool, error) {
	connString := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable pool_max_conns=%d pool_max_conn_idle_time=%s",
		config.Host,
		config.Port,
		config.UserName,
		config.Password,
		config.DbName,
		config.MaxConnections,
		config.MaxConnectionIdleTime.String(),
	)

	connConfig, parseConfigErr := pgxpool.ParseConfig(connString)
	if parseConfigErr != nil {
		return nil, fmt.Errorf("postgresql: failed to parse config: %w", parseConfigErr)
	}

	conn, err := pgxpool.NewWithConfig(ctx, connConfig)
	if err != nil {
		return nil, fmt.Errorf("postgresql: unable to connect to database: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err = conn.Ping(pingCtx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("postgresql: failed to ping database: %w", err)
	}

	log.Println("Successfully connected to PostgreSQL database!")
	return conn, nil
}
