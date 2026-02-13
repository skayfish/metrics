package flags

import (
	"fmt"
	"strconv"
	"strings"
)

// Структура для хранения хоста и порта сервера
type NetAddress struct {
	Host string // Хост сервера
	Port int    // Порт сервера
}

// Возвращает строковое представление адреса
//
//	@returns строковое представление адреса
func (n NetAddress) String() string {
	return fmt.Sprintf("%s:%d", n.Host, n.Port)
}

// Обрабатывает входную строку флага и заполняет структуру
//
//	@param input входная строка флага
//	@returns ошибку в случае передачи некорректных данных
func (n *NetAddress) Set(input string) error {
	parts := strings.Split(input, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid format: expected 'host:port', got %q", input)
	}

	host := parts[0]
	portStr := parts[1]

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port: %q", portStr)
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be in range 1–65535, got %d", port)
	}

	n.Host = host
	n.Port = port
	return nil
}

// Возвращает тип значения для документации
//
//	@returns тип значения для документации
func (NetAddress) Type() string {
	return `"host:port"`
}

// Обрабатывает входную строку переменной окружения и заполняет структуру
//
//	@param text входная строка переменной окружения
//	@returns ошибку в случае передачи некорректных данных
func (n *NetAddress) UnmarshalText(text []byte) error {
	return n.Set(string(text))
}
