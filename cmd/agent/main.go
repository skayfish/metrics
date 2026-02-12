package main

import (
	"context"
	"fmt"
	"log"

	"github.com/skayfish/metrics/internal/agent"
)

// Запуск агента
func main() {
	config, err := parseConfig()
	if err != nil {
		log.Fatal(err)
	}

	sender := agent.NewSender(*config)
	if err := sender.Run(context.TODO()); err != nil {
		fmt.Println("Во время работы приложения произошла ошибка:\n", err)
	}
}
