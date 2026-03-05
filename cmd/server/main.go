package main

import (
	"database/sql"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/skayfish/metrics/internal/logger"
	"github.com/skayfish/metrics/internal/server"
	"github.com/skayfish/metrics/internal/server/database"
)

// Запуск сервера
func main() {
	// Парсинг конфигурации из флагов запуска приложения и переменных окружения
	config, err := parseConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Подключение к базе данных
	// SF LOGIC перенести в NewServer
	var database database.Database
	if config.DatabaseDSN != nil {
		db, err := sql.Open("pgx", *config.DatabaseDSN)
		if err != nil {
			log.Fatal(err)
		}

		defer db.Close()

		database = db
	}

	// Инициализация логгера
	if err = logger.Init(config.LogLevel); err != nil {
		log.Fatal(err)
	}

	defer logger.Log.Sync()
	logger.LogS.Debugw("Server configuration", "config", config)

	// Запуск сервера
	server, err := server.NewServer(config, database)
	if err != nil {
		logger.LogS.Fatal(err)
	}

	if err = server.Listen(); err != nil {
		logger.LogS.Fatal(err)
	}
}
