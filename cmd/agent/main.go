package main

import (
	"fmt"
	"net"
	"time"

	"github.com/skayfish/metrics/internal/agent"
)

// SF TODO
const pollInterval time.Duration = 2 * time.Second

// SF TODO
const reportInterval time.Duration = 10 * time.Second

// SF LOGIC удалить
const serverAddress string = "http://localhost:8080"

// SF TODO
const host string = "localhost"

// SF TODO
const port string = "8080"

// SF TODO
const timeout time.Duration = 30 * time.Second

// SF TODO
const waitInterval time.Duration = 2 * time.Second

// PingTCP проверяет, можно ли подключиться к хосту:порту за указанное время
func PingTCP(host string, port string, timeout time.Duration) error {
	address := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return err
	}
	_ = conn.Close() // закрываем соединение
	return nil
}

// SF TODO
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

// SF TODO
func main() {
	if !checkConnection() {
		return
	}

	// Запуск отправки метрик
	sender := agent.NewSender(serverAddress, pollInterval, reportInterval)
	if err := sender.Run(); err != nil {
		fmt.Println("Во время работы приложения произошла ошибка:\n", err)
	}
}
