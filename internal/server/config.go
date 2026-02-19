package server

import (
	"time"

	"github.com/skayfish/metrics/internal/flags"
	"github.com/skayfish/metrics/internal/logger"
)

// Конфигурация сервера
type Config struct {
	// Адрес, по которому сервер ждёт запросы
	Address flags.NetAddress

	// Уровень логирования
	LogLevel logger.Level

	// SF TODO
	StoreInterval time.Duration

	// SF TODO
	FileStoragePath string

	// SF TODO
	ToRestore bool
}
