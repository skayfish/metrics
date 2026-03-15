package main

import (
	"context"
	"log"
	"time"

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
	createServerContext, createServerCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer createServerCancel()
	server, err := server.NewServer(createServerContext, config)
	if err != nil {
		logger.LogS.Fatal(err)
	}

	defer server.Close()

	startListenContext, startListenCancel := context.WithCancel(context.Background())
	defer startListenCancel()
	if err = server.Listen(startListenContext); err != nil {
		logger.LogS.Fatal(err)
	}
}
