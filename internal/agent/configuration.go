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

	// Частота обновления метрик (например, раз в 2 секунды)
	PollInterval time.Duration

	// Частота отправки метрик серверу (например, раз в 2 секунды)
	ReportInterval time.Duration
}

// todo добавить метод получения типа соединения
