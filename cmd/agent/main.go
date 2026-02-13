package main

import (
	"context"
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

	if err = logger.Init(config.LogLevel); err != nil || logger.LogS == nil {
		log.Fatal(err)
	}

	sender := agent.NewSender(*config)
	if err := sender.Run(context.TODO()); err != nil {
		logger.Log.Error(err.Error())
	}
}
