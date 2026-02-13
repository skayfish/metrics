package server

import (
	"github.com/skayfish/metrics/internal/flags"
	"github.com/skayfish/metrics/internal/logger"
)

// Конфигурация сервера
type Config struct {
	// Адрес, по которому сервер ждёт запросы
	Address flags.NetAddress

	// Уровень логирования
	LogLevel logger.Level
}
