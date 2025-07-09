package app

import (
	"time"

	"github.com/yusufziyrek/bank-app/common/postgresql"
)

type ConfigurationManager struct {
	PostgreSqlConfig postgresql.Config
	AppPort          string
	JwtSecret        string
	JwtTTL           int // dakika cinsinden
}

func NewConfigurationManager() *ConfigurationManager {
	return &ConfigurationManager{
		PostgreSqlConfig: postgresql.Config{
			Host:                  "localhost",
			Port:                  "6432",
			UserName:              "postgres",
			Password:              "1234",
			DbName:                "bankapp",
			MaxConnections:        10,
			MaxConnectionIdleTime: 300 * time.Second,
		},
		AppPort:   "8080",
		JwtSecret: "change-me",
		JwtTTL:    60, // 60 dakika
	}
}
