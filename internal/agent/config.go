package agent

import (
	"time"

	"github.com/skayfish/metrics/internal/logger"
)

// Конфигурация работы менеджера отправки метрик серверу
type Config struct {
	// Безопасное соединение с сервером
	SecureConnection bool

	// Хост сервера
	Host string

	// Порт сервера
	Port int

	// Максимальное время ожидания между попытками подключения к серверу
	RetryMaxWaitTime time.Duration

	// Частота попыток подключения к серверу (например, раз в 2 секунды)
	RetryWaitTime time.Duration

	// Частота обновления метрик (например, раз в 2 секунды)
	PollInterval time.Duration

	// Частота отправки метрик серверу (например, раз в 2 секунды)
	ReportInterval time.Duration

	// Уровень логирования
	LogLevel logger.Level

	// Ключ для подписи запросов и ответов
	KeyEncryption *string

	// SF TODO
	RateLimit uint
}

// Возвращает тип соединения
//
//	@returns тип соединения [https, http]
func (obj *Config) getConnectionType() string {
	if obj.SecureConnection {
		return "https"
	}

	return "http"
}
