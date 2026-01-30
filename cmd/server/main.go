package main

import (
	"log"
	"net/http"

	"github.com/skayfish/metrics/internal/handler"
	"github.com/skayfish/metrics/internal/server/storage"
)

// Настраивает и запускает сервер
//
//	@param storage хранилище метрик
//	@returns ошибку работы сервера
func run(storage *storage.MemStorage) error {
	mux := http.NewServeMux()
	mux.Handle("/update/", http.StripPrefix("/update", handler.CreateHandlerUpdate(storage)))
	return http.ListenAndServe(":8080", mux)
}

// Запуск сервера
func main() {
	storage := storage.NewMemStorage()
	log.Fatal(run(&storage))
}
