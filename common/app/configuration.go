package app

import (
	"log"
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
	host := os.Getenv("PG_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("PG_PORT")
	if port == "" {
		port = "5432"
	}

	user := os.Getenv("PG_USER")
	if user == "" {
		user = "postgres"
	}

	pass := os.Getenv("PG_PASS")
	if pass == "" {
		pass = "1234"
	}

	db := os.Getenv("PG_DB")
	if db == "" {
		db = "bankapp"
	}

	idleStr := os.Getenv("PG_IDLE_TIME")
	if idleStr == "" {
		idleStr = "300"
	}
	idleSec, err := strconv.Atoi(idleStr)
	if err != nil {
		log.Fatalf("Geçersiz PG_IDLE_TIME: %v", err)
	}

	maxConnStr := os.Getenv("PG_MAX_CONNS")
	if maxConnStr == "" {
		maxConnStr = "10"
	}
	maxConns, err := strconv.Atoi(maxConnStr)
	if err != nil {
		log.Fatalf("Geçersiz PG_MAX_CONNS: %v", err)
	}

	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "8080"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "change-me"
	}

	jwtTTLStr := os.Getenv("JWT_TTL")
	jwtTTL, err := strconv.Atoi(jwtTTLStr)
	if err != nil || jwtTTL <= 0 {
		jwtTTL = 60
	}

	return &ConfigurationManager{
		PostgreSqlConfig: postgresql.Config{
			Host:                  host,
			Port:                  port,
			UserName:              user,
			Password:              pass,
			DbName:                db,
			MaxConnections:        int32(maxConns),
			MaxConnectionIdleTime: time.Duration(idleSec) * time.Second,
		},
		AppPort:   appPort,
		JwtSecret: jwtSecret,
		JwtTTL:    jwtTTL,
	}
}
