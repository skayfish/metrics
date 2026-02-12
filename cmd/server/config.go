package main

import (
	"fmt"

	"github.com/caarlos0/env/v6"
	"github.com/skayfish/metrics/internal/flags"
	"github.com/spf13/pflag"
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
func parseConfig() (*flags.NetAddress, error) {
	// Парсинг флагов
	addr := flags.NetAddress{Host: "localhost", Port: 8080}
	pflag.VarP(&addr, "address", "a", "Server listening address in host:port format")
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

	return &addr, nil
}
