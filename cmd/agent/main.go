package main

import (
	"fmt"

	"github.com/skayfish/metrics/internal/agent"
)

// Запуск агента
func main() {
	sender := agent.NewSender(parseFlags())
	if err := sender.Run(nil); err != nil {
		fmt.Println("Во время работы приложения произошла ошибка:\n", err)
	}
}
