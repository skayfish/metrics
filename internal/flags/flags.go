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
func (a NetAddress) String() string {
	return fmt.Sprintf("%s:%d", a.Host, a.Port)
}

// Обрабатывает входную строку флага и заполняет структуру
//
//	@param input входная строка флага
//	@returns ошибка в случае передачи некорректных данных
func (a *NetAddress) Set(input string) error {
	parts := strings.Split(input, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid format: expected 'host:port', got '%s'", input)
	}

	host := parts[0]
	portStr := parts[1]

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port: '%s'", portStr)
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be in range 1–65535, got %d", port)
	}

	a.Host = host
	a.Port = port
	return nil
}

// Возвращает тип значения для документации
//
//	@returns тип значения для документации
func (a *NetAddress) Type() string {
	return "host:port"
}
