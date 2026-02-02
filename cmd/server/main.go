package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/skayfish/metrics/internal/server/handler"
	"github.com/skayfish/metrics/internal/server/storage"
)

// Возвращает маршрутизатор запросов
//
//	@param storage хранилище метрик
//	@returns маршрутизатор запросов
func getRouter(storage *storage.MemStorage) (chi.Router, error) {
	router := chi.NewRouter()

	router.Post("/update/{type}/{name}/{value}", handler.CreateUpdateHandler(storage))
	router.Get("/value/{type}/{name}", handler.CreateGetValueHandler(storage))
	getAllMetricsHandler, err := handler.CreateGetAllMetricsHandler(storage)
	if err != nil {
		return nil, err
	}

	router.Get("/", getAllMetricsHandler)
	return router, nil
}

// Запуск сервера
func main() {
	netAddress := parseFlags()
	storage := storage.NewMemStorage()
	router, err := getRouter(&storage)
	if err != nil {
		log.Fatal(err)
	}

	if err = http.ListenAndServe(netAddress.String(), router); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
