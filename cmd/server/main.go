package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/skayfish/metrics/internal/logger"
	"github.com/skayfish/metrics/internal/server/controller"
	"github.com/skayfish/metrics/internal/server/middleware"
	"github.com/skayfish/metrics/internal/server/storage"
)

// Возвращает маршрутизатор запросов
//
//	@param storage хранилище метрик
//	@returns маршрутизатор запросов в случае успеха
func getRouter(controller *controller.MetricsController) chi.Router {
	router := chi.NewRouter()
	router.Use(middleware.LoggingMiddleware, middleware.CompressingMiddleware)
	router.Post("/update/{type}/{name}/{value}", controller.UpdateFromURL)
	router.Post("/update", controller.UpdateFromJSON)
	router.Post("/update/", controller.UpdateFromJSON)
	router.Get("/value/{type}/{name}", controller.GetValueFromURL)
	router.Post("/value", controller.GetValueFromJSON)
	router.Post("/value/", controller.GetValueFromJSON)
	router.Get("/", controller.GetAllMetrics)

	return router
}

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

	// Инициализация хранилища, контроллеров
	storage := storage.NewMemStorage()

	metricsController, err := controller.NewMetricsController(&storage)
	if err != nil {
		log.Fatal(err)
	}

	// Настройка маршрутизации запросов
	router := getRouter(metricsController)

	// Запуск сервера
	logger.LogS.Info("Server launch successful")
	if err = http.ListenAndServe(config.Address.String(), router); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
