package agent

import "time"

// Конфигурация работы менеджера отправки метрик серверу
type Configuration struct {
	// Безопасное соединение с сервером
	SecureConnection bool

	// Хост сервера
	Host string

	// Порт сервера
	Port string

	// Таймаут ожидания подключения к серверу
	RetryTimeout time.Duration

	// Частота попыток подключения к серверу (например, раз в 2 секунды)
	RetryWaitTime time.Duration

	// Частота обновления метрик (например, раз в 2 секунды)
	PollInterval time.Duration

	// Частота отправки метрик серверу (например, раз в 2 секунды)
	ReportInterval time.Duration
}

// Возвращает тип соединения
//
//	@returns тип соединения [https, http]
func (obj *Configuration) getConnectionType() string {
	if obj.SecureConnection {
		return "https"
	}

	return "http"
}
