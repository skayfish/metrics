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
	config, err := parseConfig()
	if err != nil {
		log.Fatal(err)
	}

	if err = logger.Init(config.LogLevel); err != nil {
		log.Fatal(err)
	}

	defer logger.Log.Sync()

	sender := agent.NewSender(*config)
	if err := sender.Run(context.TODO()); err != nil {
		if !errors.Is(err, context.DeadlineExceeded) {
			logger.Log.Fatal(err.Error())
		}
	}
}
