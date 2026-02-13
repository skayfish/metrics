package main

import (
	"fmt"
	"log"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/skayfish/metrics/internal/agent"
	"github.com/skayfish/metrics/internal/flags"
	"github.com/skayfish/metrics/internal/logger"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

// Переменные окружения
type environments struct {
	// Сетевой адрес
	Address *flags.NetAddress `env:"ADDRESS" example:"localhost:8080"`

	// Частота отправки метрик серверу (например, раз в 2 секунды)
	ReportInterval *uint `env:"REPORT_INTERVAL" example:"2"`

	// Частота обновления метрик (например, раз в 2 секунды)
	PollInterval *uint `env:"POLL_INTERVAL" example:"2"`
}

// Парсит флаги, указанные при запуске программы и переменные окружения
//
// @returns *agent.Config конфигурацию работы менеджера отправки метрик серверу, в случае успеха
// @returns error ошибку, в противном случае
func parseConfig() (*agent.Config, error) {
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

	if envs.PollInterval != nil {
		*pollInterval = *envs.PollInterval
	}

	if envs.ReportInterval != nil {
		*reportInterval = *envs.ReportInterval
	}

	log.Printf("Debug data:\n")
	log.Printf("\tHost: %s", addr.Host)
	log.Printf("\tPort: %d", addr.Port)
	log.Printf("\tSecure: %t", *isSecure)
	log.Printf("\tRetryMaxWaitTime: %ds", *retryMaxWaitTime)
	log.Printf("\tRetryWaitTime: %ds", *retryWaitTime)
	log.Printf("\tPollInterval: %ds", *pollInterval)
	log.Printf("\tReportInterval: %ds", *reportInterval)

	return &agent.Config{
		SecureConnection: *isSecure,
		Host:             addr.Host,
		Port:             addr.Port,
		RetryMaxWaitTime: time.Duration(*retryMaxWaitTime) * time.Second,
		RetryWaitTime:    time.Duration(*retryWaitTime) * time.Second,
		PollInterval:     time.Duration(*pollInterval) * time.Second,
		ReportInterval:   time.Duration(*reportInterval) * time.Second,
		LogLevel:         zap.NewAtomicLevelAt(logLevel.Lvl),
	}, nil
}
