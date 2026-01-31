package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/skayfish/metrics/internal/handler"
	"github.com/skayfish/metrics/internal/server/storage"
)

// SF TODO
func getRouter(storage *storage.MemStorage) chi.Router {
	router := chi.NewRouter()

	router.Post("/update/{type}/{name}/{value}", handler.CreateUpdateHandler(storage))
	router.Get("/value/{type}/{name}", handler.CreateValueHandler(storage))
	return router
}

// Запуск сервера
func main() {
	storage := storage.NewMemStorage()
	log.Fatal(http.ListenAndServe(":8080", getRouter(&storage)))
}
