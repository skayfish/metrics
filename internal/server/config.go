package server

import (
	"github.com/skayfish/metrics/internal/flags"
	"go.uber.org/zap"
)

// Конфигурация сервера
type Config struct {
	// Адрес, по которому сервер ждёт запросы
	Address flags.NetAddress

	// Уровень логирования // SF LOGIC использовать собственный
	LogLevel zap.AtomicLevel
}
