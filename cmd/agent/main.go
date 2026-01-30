package main

import (
	"fmt"
	"net"
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
const timeout time.Duration = 30 * time.Second

// Частота попыток подключения к серверу (например, раз в 2 секунды)
const waitInterval time.Duration = 2 * time.Second

// Частота обновления метрик (например, раз в 2 секунды)
const pollInterval time.Duration = 2 * time.Second

// Частота отправки метрик серверу (например, раз в 2 секунды)
const reportInterval time.Duration = 10 * time.Second

// Проверяет, можно ли подключиться к хосту:порту за указанное время
//
//	@param host    хост сервера
//	@param port    порт сервера
//	@param timeout таймаут ожидания попытки подключения к серверу
//	@returns ошибку попытки подключения к серверу
func PingTCP(host string, port string, timeout time.Duration) error {
	address := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return err
	}
	_ = conn.Close() // закрываем соединение
	return nil
}

// Проверяет можно ли подключиться к серверу
//
//	@returns true - подключение успешно, false - ошибка подключения
func checkConnection() bool {
	totalTime := time.Duration(0)
	totalAttempts := timeout.Nanoseconds() / waitInterval.Nanoseconds()
	attemptsCounter := 1
	for {
		if totalTime >= timeout {
			fmt.Printf("Не удалось подключиться к %s:%s ;(\n", host, port)
			return false
		}

		err := PingTCP(host, port, timeout)
		if err != nil {
			fmt.Printf("Не удалось подключиться к %s:%s: %v\n", host, port, err)
			fmt.Printf("Попытка соединения с сервером... [%d/%d, ожидание %v]\n", attemptsCounter, totalAttempts, waitInterval)
			attemptsCounter++
		} else {
			fmt.Printf("Успешно подключились к %s:%s\n\n", host, port)
			return true
		}

		time.Sleep(waitInterval)
		totalTime += waitInterval
	}
}

// Запуск агента
func main() {
	if !checkConnection() {
		return
	}

	config := agent.Configuration{
		SecureConnection: secureConnection,
		Host:             host,
		Port:             port,
		PollInterval:     pollInterval,
		ReportInterval:   reportInterval,
	}
	sender := agent.NewSender(config)
	if err := sender.Run(); err != nil {
		fmt.Println("Во время работы приложения произошла ошибка:\n", err)
	}
}
