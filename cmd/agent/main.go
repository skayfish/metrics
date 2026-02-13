package main

import (
	"context"
	"errors"
	"log"

	"github.com/skayfish/metrics/internal/agent"
	"github.com/skayfish/metrics/internal/logger"
)

// Запуск агента
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

	logger.LogS.Debugw("Agent configuration", "config", config)

	// Инициализация и запуск менеджера отправки данных серверу
	sender := agent.NewSender(*config)
	if err := sender.Run(context.TODO()); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return
		}

		logger.Log.Fatal(err.Error())
	}
}
