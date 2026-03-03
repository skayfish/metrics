package main

import (
	"fmt"
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

	// Интервал времени в секундах, по истечении которого текущие данные хранилища метрик сохраняются на диск
	StoreInterval *uint `env:"STORE_INTERVAL" example:"5"`

	// Путь до файла, куда сохраняются данные хранилища метрик. По умолчанию во временный файл
	FileStoragePath *string `env:"FILE_STORAGE_PATH" example:"/home/user/server"`

	// Булево значение (true/false), определяющее, следует ли загружать ранее сохранённые значения из указанного файла при старте сервера
	ToRestore *bool `env:"RESTORE" example:"true"`

	// SF TODO
	DatabaseDSN *string `env:"DATABASE_DSN" example:"host=localhost port=5432 user=username password=XXXX dbname=databasename"`
}

// Парсит флаги, указанные при запуске программы и переменные окружения
//
//	@returns *server.Config конфигурацию сервера в случае успеха
//	@returns error ошибку, в противном случае
func parseConfig() (*server.Config, error) {
	// Парсинг флагов
	addr := flags.NetAddress{Host: "localhost", Port: 8080}
	pflag.VarP(&addr, "address", "a", "Server listening address in host:port format")
	var logLevel logger.Level
	pflag.VarP(&logLevel, "log-level", "l", "Logging level")
	storeInterval := pflag.UintP("store-interval", "i", 300,
		"Number of seconds before current storage data is written to the \"--file-storage-path\" location")
	fileStoragePath := pflag.StringP("file-storage-path", "f", "",
		"File system path to which current storage data is persisted")
	toRestore := pflag.BoolP("restore", "r", false,
		"Read saved values from the \"--file-storage-path\" file when the server starts (default false)")
	databaseDSN := pflag.StringP("database-dsn", "d", "",
		`Connection string for database access, structured as: "host=<host> port=<port> user=<username> password=<pass> dbname=<database name>"`)

	pflag.Parse()

	// Парсинг переменных окружения
	var envs environments
	err := env.Parse(&envs)
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

	if envs.DatabaseDSN != nil {
		*databaseDSN = *envs.DatabaseDSN
	}

	result := server.Config{
		Address:         addr,
		LogLevel:        logLevel,
		StoreInterval:   time.Duration(*storeInterval) * time.Second,
		FileStoragePath: *fileStoragePath,
		ToRestore:       *toRestore,
		DatabaseDSN:     *databaseDSN,
	}

	return &result, nil
}
