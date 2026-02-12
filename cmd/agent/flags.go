package main

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/skayfish/metrics/internal/agent"
	"github.com/skayfish/metrics/internal/flags"
	"github.com/spf13/pflag"
)

// Парсит флаги, указанные при запуске программы и переменные окружения
//
//	@returns конфигурацию работы менеджера отправки метрик серверу
func parseFlags() (*agent.Config, error) {
	// Парсинг флагов
	addr := flags.NetAddress{Host: "localhost", Port: 8080}
	pflag.VarP(&addr, "address", "a", "Server address in format host:port")
	pollInterval := pflag.UintP("poll-interval", "p", 2,
		"Metrics collection frequency, in seconds")
	reportInterval := pflag.UintP("report-interval", "r", 10,
		"Metrics sending frequency to the server, in seconds")
	isSecure := pflag.Bool("secure-connection", false,
		"Secure server connection [HTTP — disabled, HTTPS — enabled] (default false)")
	retryMaxWaitTime := pflag.Uint("retry-max-wait-time", 10,
		"Maximum wait time between connection attempts, in seconds")
	retryWaitTime := pflag.Uint("retry-wait-time", 2,
		"Connection retry interval, in seconds")

	pflag.Parse()

	// Парсинг переменных окружения
	var config flags.Config
	err := env.Parse(&config)
	if err != nil {
		return nil, fmt.Errorf("main: failed to parse environment variables: %v", err)
	}

	host := addr.Host
	port := addr.Port
	if config.Address != nil {
		host = config.Address.Host
		port = config.Address.Port
	}

	if config.PollInterval != nil {
		*pollInterval = *config.PollInterval
	}

	if config.ReportInterval != nil {
		*reportInterval = *config.ReportInterval
	}

	return &agent.Config{
		SecureConnection: *isSecure,
		Host:             host,
		Port:             port,
		RetryMaxWaitTime: time.Duration(*retryMaxWaitTime) * time.Second,
		RetryWaitTime:    time.Duration(*retryWaitTime) * time.Second,
		PollInterval:     time.Duration(*pollInterval) * time.Second,
		ReportInterval:   time.Duration(*reportInterval) * time.Second,
	}, nil
}
