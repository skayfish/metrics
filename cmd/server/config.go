package main

import (
	"fmt"
	"os"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/skayfish/metrics/internal/flags"
	"github.com/skayfish/metrics/internal/logger"
	"github.com/skayfish/metrics/internal/server"
	"github.com/spf13/pflag"
)

// Переменные окружения
type environments struct {
	// Сетевой адрес
	Address *flags.NetAddress `env:"ADDRESS" example:"localhost:8080"`

	// SF TODO
	StoreInterval *uint `env:"STORE_INTERVAL" example:"5"`

	// SF TODO
	FileStoragePath *string `env:"FILE_STORAGE_PATH" example:"/home/user/server"`

	// SF TODO
	ToRestore *bool `env:"RESTORE" example:"true"`
}

// Парсит флаги, указанные при запуске программы и переменные окружения
//
//	@returns *server.Config конфигурацию сервера в случае успеха
//	@returns error ошибку, в противном случае
func parseConfig() (*server.Config, error) {
	binaryDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get binary directory: %v", err)
	}

	// Парсинг флагов
	addr := flags.NetAddress{Host: "localhost", Port: 8080}
	pflag.VarP(&addr, "address", "a", "Server listening address in host:port format")
	var logLevel logger.Level
	pflag.VarP(&logLevel, "log-level", "l", "Logging level")
	storeInterval := pflag.UintP("store-interval", "i", 300,
		"Number of seconds before current storage data is written to the \"--file-storage-path\" location")
	fileStoragePath := pflag.StringP("file-storage-path", "f", binaryDir+"\\storage.json",
		"File system path to which current storage data is persisted")
	toRestore := pflag.BoolP("restore", "r", false,
		"Read saved values from the \"--file-storage-path\" file when the server starts (default false)")

	pflag.Parse()

	// Парсинг переменных окружения
	var envs environments
	err = env.Parse(&envs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %v", err)
	}

	if envs.Address != nil {
		addr = *envs.Address
	}

	if envs.StoreInterval != nil {
		*storeInterval = *envs.StoreInterval
	}

	if envs.FileStoragePath != nil {
		*fileStoragePath = *envs.FileStoragePath
	}

	if envs.ToRestore != nil {
		*toRestore = *envs.ToRestore
	}

	result := server.Config{
		Address:         addr,
		LogLevel:        logLevel,
		StoreInterval:   time.Duration(*storeInterval) * time.Second,
		FileStoragePath: *fileStoragePath,
		ToRestore:       *toRestore,
	}

	return &result, nil
}
