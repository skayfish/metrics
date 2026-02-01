package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/skayfish/metrics/internal/agent"
	"github.com/spf13/pflag"
)

// SF LOGIC

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

// SF LOGIC
// SF LOGIC
// SF LOGIC

// Структура для хранения хоста и порта сервера
type NetAddress struct {
	Host string // Хост сервера
	Port int    // Порт сервера
}

// SF TODO возвращает строковое представление адреса
func (a NetAddress) String() string {
	return fmt.Sprintf("%s:%d", a.Host, a.Port)
}

// SF TODO обрабатывает входную строку и заполняет структуру
func (a *NetAddress) Set(s string) error {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid format: expected 'host:port', got %s", s)
	}

	host := parts[0]
	portStr := parts[1]

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port: %s", portStr)
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be in range 1–65535, got %d", port)
	}

	a.Host = host
	a.Port = port
	return nil
}

// SF TODO возвращает тип значения для документации
func (a *NetAddress) Type() string {
	return "host:port"
}

// SF TODO
func parseFlags() (config agent.Config) {
	addr := NetAddress{Host: "localhost", Port: 8080}
	pflag.VarP(&addr, "netaddress", "a", "Server address in format host:port (e.g., localhost:8080)")
	// SF LOGIC

	config.Host = addr.Host
	config.Port = addr.Port

	return
}
