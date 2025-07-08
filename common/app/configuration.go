package app

import (
	"os"
	"strconv"
	"time"

	"github.com/yusufziyrek/bank-app/common/postgresql"
)

type ConfigurationManager struct {
	PostgreSqlConfig postgresql.Config
	AppPort          string
	JwtSecret        string
	JwtTTL           int
}

func NewConfigurationManager() *ConfigurationManager {
	maxIdle, _ := strconv.Atoi(os.Getenv("PG_IDLE_TIME"))
	cfg := postgresql.Config{
		Host:                  os.Getenv("PG_HOST"),
		Port:                  os.Getenv("PG_PORT"),
		UserName:              os.Getenv("PG_USER"),
		Password:              os.Getenv("PG_PASS"),
		DbName:                os.Getenv("PG_DB"),
		MaxConnections:        10,
		MaxConnectionIdleTime: time.Duration(maxIdle) * time.Second,
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "change-me"
	}
	jwtTTL, err := strconv.Atoi(os.Getenv("JWT_TTL"))
	if err != nil || jwtTTL <= 0 {
		jwtTTL = 60
	}

	return &ConfigurationManager{
		PostgreSqlConfig: cfg,
		AppPort:          os.Getenv("APP_PORT"),
		JwtSecret:        jwtSecret,
		JwtTTL:           jwtTTL,
	}
}
