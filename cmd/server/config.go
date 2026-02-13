package main

import (
	"fmt"

	"github.com/caarlos0/env/v6"
	"github.com/skayfish/metrics/internal/flags"
	"github.com/skayfish/metrics/internal/logger"
	"github.com/skayfish/metrics/internal/server"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

// Переменные окружения
type environments struct {
	// Сетевой адрес
	Address *flags.NetAddress `env:"ADDRESS" example:"localhost:8080"`
}

// Парсит флаги, указанные при запуске программы и переменные окружения
//
//	@returns *flags.NetAddress данные о хосте и порте, в случае успеха
//	@returns error ошибку, в противном случае
//
// SF TODO
func parseConfig() (*server.Config, error) {
	// Парсинг флагов
	addr := flags.NetAddress{Host: "localhost", Port: 8080}
	pflag.VarP(&addr, "address", "a", "Server listening address in host:port format")
	logLevel := logger.Level{}
	pflag.VarP(&logLevel, "log-level", "l", "Logging level")

	pflag.Parse()

	// Парсинг переменных окружения
	var envs environments
	err := env.Parse(&envs)
	if err != nil {
		return nil, fmt.Errorf("main: failed to parse environment variables: %v", err)
	}

	if envs.Address != nil {
		addr = *envs.Address
	}

	result := server.Config{
		Address:  addr,
		LogLevel: zap.NewAtomicLevelAt(logLevel.Lvl),
	}

	return &result, nil
}
