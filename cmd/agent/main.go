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
func main() {
	// Проверка соединения с сервером
	timeout := 30 * time.Second
	err := PingTCP(host, port, timeout)
	if err != nil {
		fmt.Printf("Не удалось подключиться к %s:%s: %v\n\n\n", host, port, err)
	} else {
		fmt.Printf("Успешно подключились к %s:%s\n\n\n", host, port)
	}

	// Запуск отправки метрик
	sender := agent.NewSender(serverAddress, pollInterval, reportInterval)
	if err := sender.Run(); err != nil {
		fmt.Println("Во время работы приложения произошла ошибка:\n", err)
	}
}
