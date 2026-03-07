package main

import (
	"context"
	"log"

	"github.com/skayfish/metrics/internal/logger"
	"github.com/skayfish/metrics/internal/server"
)

// Запуск сервера
func main() {
	// Парсинг конфигурации из флагов запуска приложения и переменных окружения
	config, err := parseConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Инициализация логгера
	if err = logger.Init(config.LogLevel); err != nil {
		log.Fatal(err)
	}

	defer logger.Log.Sync()
	logger.LogS.Debugw("Server configuration", "config", config)

	// Запуск сервера
	server, err := server.NewServer(config)
	if err != nil {
		logger.LogS.Fatal(err)
	}

	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err = server.Listen(ctx); err != nil {
		logger.LogS.Fatal(err)
	}
}
