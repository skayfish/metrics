package main

import (
	"fmt"
	"time"

	"github.com/skayfish/metrics/internal/agent"
)

// SF TODO
const pollInterval time.Duration = 2 * time.Second

// SF TODO
const reportInterval time.Duration = 10 * time.Second

// SF TODO
const serverAddress string = "http://localhost:8080"

// SF TODO
func main() {
	sender := agent.NewSender(serverAddress, pollInterval, reportInterval)
	if err := sender.Run(); err != nil {
		fmt.Println("Во время работы приложения произошла ошибка:\n", err)
	}
}
