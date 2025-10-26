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
	AppEnv           string
	JwtSecret        string
	JwtTTL           int
	AllowedOrigins   string
	RedisAddr        string
	RedisPassword    string
	RedisDB          int
}

func NewConfigurationManager() *ConfigurationManager {
	host := os.Getenv("PG_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("PG_PORT")
	if port == "" {
		port = "6432"
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

	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
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

	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:5173"
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")

	redisDBStr := os.Getenv("REDIS_DB")
	redisDB := 0
	if redisDBStr != "" {
		parsed, err := strconv.Atoi(redisDBStr)
		if err != nil {
			log.Fatalf("Geçersiz REDIS_DB: %v", err)
		}
		redisDB = parsed
	}

	log.Printf("PostgreSQL Config - Host: %s, Port: %s, User: %s, DB: %s", host, port, user, db)

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
		AppPort:        appPort,
		AppEnv:         appEnv,
		JwtSecret:      jwtSecret,
		JwtTTL:         jwtTTL,
		AllowedOrigins: allowedOrigins,
		RedisAddr:      redisAddr,
		RedisPassword:  redisPassword,
		RedisDB:        redisDB,
	}
}
