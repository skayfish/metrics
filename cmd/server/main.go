package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/skayfish/metrics/internal/server/controller"
	"github.com/skayfish/metrics/internal/server/storage"
)

// Возвращает маршрутизатор запросов
//
//	@param storage хранилище метрик
//	@returns маршрутизатор запросов в случае успеха
//	@returns ошибку в ином случае
//
// SF TODO
func getRouter(controller *controller.MetricsController) chi.Router {
	router := chi.NewRouter()
	router.Post("/update/{type}/{name}/{value}", controller.Update)
	router.Get("/value/{type}/{name}", controller.GetValue)
	router.Get("/", controller.GetAllMetrics)

	return router
}

// Запуск сервера
func main() {
	netAddress := parseFlags()
	storage := storage.NewMemStorage()
	metricsController, err := controller.NewMetricsController(&storage)
	if err != nil {
		log.Fatal(err)
	}

	router := getRouter(metricsController)

	if err = http.ListenAndServe(netAddress.String(), router); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
