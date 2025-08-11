package postgresql

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Host                  string
	Port                  string
	UserName              string
	Password              string
	DbName                string
	MaxConnections        int32
	MaxConnectionIdleTime time.Duration
}

func GetConnectionPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	connString := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable pool_max_conns=%d pool_max_conn_idle_time=%s connect_timeout=10",
		cfg.Host, cfg.Port, cfg.UserName, cfg.Password, cfg.DbName,
		cfg.MaxConnections, cfg.MaxConnectionIdleTime,
	)
	pc, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("postgresql: parse config: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, fmt.Errorf("postgresql: new pool: %w", err)
	}
	return pool, nil
}
