package main

import (
	"fmt"

	"github.com/caarlos0/env/v6"
	"github.com/skayfish/metrics/internal/flags"
	"github.com/spf13/pflag"
)

// Парсит флаги, указанные при запуске программы и переменные окружения
//
//	@returns данные о хосте и порте
//
// SF TODO
func parseFlags() (*flags.NetAddress, error) {
	// Парсинг флагов
	addr := flags.NetAddress{Host: "localhost", Port: 8080}
	pflag.VarP(&addr, "address", "a", "Server listening address in host:port format")
	pflag.Parse()

	// Парсинг переменных окружения
	var config flags.Config
	err := env.Parse(&config)
	if err != nil {
		return nil, fmt.Errorf("main: failed to parse environment variables: %v", err)
	}

	if config.Address != nil {
		addr = *config.Address
	}

	return &addr, nil
}
