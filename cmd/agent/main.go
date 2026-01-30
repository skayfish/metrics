package main

import (
	"fmt"
	"time"

	"github.com/skayfish/metrics/internal/agent"
)

// Хост сервера
const host string = "localhost"

// Порт сервера
const port string = "8080"

// Безопасное соединение с сервером
const secureConnection bool = false

// Таймаут ожидания подключения к серверу
const retryTimeout time.Duration = 30 * time.Second

// Частота попыток подключения к серверу (например, раз в 2 секунды)
const retryWaitTime time.Duration = 2 * time.Second

// Частота обновления метрик (например, раз в 2 секунды)
const pollInterval time.Duration = 2 * time.Second

// Частота отправки метрик серверу (например, раз в 2 секунды)
const reportInterval time.Duration = 10 * time.Second

// Запуск агента
func main() {
	config := agent.Configuration{
		SecureConnection: secureConnection,
		Host:             host,
		Port:             port,
		RetryTimeout:     retryTimeout,
		RetryWaitTime:    retryWaitTime,
		PollInterval:     pollInterval,
		ReportInterval:   reportInterval,
	}
	sender := agent.NewSender(config)
	if err := sender.Run(); err != nil {
		fmt.Println("Во время работы приложения произошла ошибка:\n", err)
	}
}
