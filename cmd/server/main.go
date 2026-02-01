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
func getRouter(storage *storage.MemStorage) chi.Router {
	router := chi.NewRouter()

	router.Post("/update/{type}/{name}/{value}", handler.CreateUpdateHandler(storage))
	router.Get("/value/{type}/{name}", handler.CreateGetValueHandler(storage))
	router.Get("/", handler.CreateGetAllValuesHandler(storage))
	return router
}

// Запуск сервера
func main() {
	netAddress := parseFlags()
	storage := storage.NewMemStorage()
	log.Fatal(http.ListenAndServe(netAddress.String(), getRouter(&storage)))
}
