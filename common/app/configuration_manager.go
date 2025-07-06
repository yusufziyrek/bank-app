package app

import (
	"github.com/yusufziyrek/bank-app/common/postgresql"
	"time"
)

type ConfigurationManager struct {
	PostgreSqlConfig postgresql.Config
	AppPort          string
}

func NewConfigurationManager() *ConfigurationManager {
	postgreSqlConfig := postgresql.Config{
		Host:                  "localhost",
		Port:                  "6432",
		UserName:              "postgres",
		Password:              "1234",
		DbName:                "bankapp",
		MaxConnections:        10,
		MaxConnectionIdleTime: 30 * time.Second,
	}

	appPort := "8080"

	return &ConfigurationManager{
		PostgreSqlConfig: postgreSqlConfig,
		AppPort:          appPort,
	}
}
