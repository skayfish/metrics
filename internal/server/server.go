package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose"

	"github.com/go-chi/chi/v5"
	"github.com/skayfish/metrics/internal/logger"
	"github.com/skayfish/metrics/internal/model"
	"github.com/skayfish/metrics/internal/server/controller"
	"github.com/skayfish/metrics/internal/server/middleware"
	"github.com/skayfish/metrics/internal/server/storage"
)

// Сервер для принятия запросов по HTTP
type Server struct {
	// Конфигурация сервере
	config *Config

	// SF TODO
	storage storage.Storage

	// Маршрутизатор запросов
	router *chi.Router

	// Канал для отправки сигнала на сохранение данных хранилища в файл
	// @warning использовать только когда doSaveStorage = true
	saveStorageChan chan struct{}

	// Определяет, нужно ли сохранять данные хранилища метрик в файл.
	// Используется для корректной работы канала.
	doSaveStorage atomic.Bool
}

// Возвращает middleware-обёртку.
// Обёртка корректно отправляет сигнал на сохранение данных хранилища в файл после работы handler
//
//	@returns middleware-обёртку
func (s *Server) getSaveMiddleware() func(handler http.HandlerFunc) http.HandlerFunc {
	return func(handler http.HandlerFunc) http.HandlerFunc {
		return func(resp http.ResponseWriter, req *http.Request) {
			handler(resp, req)
			if s.doSaveStorage.Load() {
				s.saveStorageChan <- struct{}{}
			}
		}
	}
}

// Возвращает маршрутизатор запросов
//
//	@param storage        хранилище метрик
//	@param database       база данных
//	@param saveMiddleware middleware-обёртка для отправки сигнала на сохранение данных хранилища метрик в файл
//	@returns chi.Router маршрутизатор запросов в случае успеха
//	@returns error ошибку в ином случае
//
// SF TODO
func getRouter(
	storage storage.Storage,
	saveMiddleware func(http.HandlerFunc) http.HandlerFunc,
) (chi.Router, error) {
	router := chi.NewRouter()
	router.Use(middleware.CompressingMiddleware, middleware.LoggingMiddleware)

	// Base

	baseController := controller.NewBaseController(storage)

	router.Get("/ping", baseController.Ping)

	// Metrics

	metricsController, err := controller.NewMetricsController(storage)
	if err != nil {
		return nil, fmt.Errorf("failed create metric controller: %w", err)
	}

	router.Post("/update/{type}/{name}/{value}", saveMiddleware(metricsController.UpdateFromURL))
	router.Post("/update", saveMiddleware(metricsController.UpdateFromJSON))
	router.Post("/update/", saveMiddleware(metricsController.UpdateFromJSON))
	router.Get("/value/{type}/{name}", metricsController.GetValueFromURL)
	router.Post("/value", metricsController.GetMetricFromJSON)
	router.Post("/value/", metricsController.GetMetricFromJSON)
	router.Get("/", metricsController.GetAllMetrics)

	return router, nil
}

// Считывает данные метрик из json файла и создаёт из них хранилище метрик
//
//	@param filePath путь к json файлу с данными метрик
//	@returns *storage.MemStorage хранилище метрик в случае успеха
//	@returns error ошибку в ином случае
func createStorageFromJSON(filePath string) (*storage.MemStorage, error) {
	const prefix = "server.createStorageFromJSON"

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("%s: failed read from file %q: %w", prefix, filePath, err)
	}

	var metrics []model.Metrics
	if err = json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("%s: failed unmarshal metrics from file %q: %w", prefix, filePath, err)
	}

	storage := make(storage.MemStorage, len(metrics))
	for _, metric := range metrics {
		storage[metric.ID] = metric
	}

	return &storage, nil
}

// SF TODO
func createStorage(config *Config) (storage.Storage, error) {
	const prefix = "server.createStorage"

	// Подключение к серверу базы данных, если есть данные для соединения
	if config.DatabaseDSN != nil {
		db, err := sql.Open("pgx", *config.DatabaseDSN)
		if err != nil {
			return nil, fmt.Errorf("%s: failed open database: %w", prefix, err)
		}

		err = goose.Up(db, config.MigrationsPath)
		if err != nil {
			return nil, fmt.Errorf("%s: migrations failed: %w", prefix, err)
		}

		storage, err := storage.NewPostgreSQLStorage(db)
		if err != nil {
			return nil, fmt.Errorf("%s: failed create PostgreSQL instance: %w", prefix, err)
		}

		return storage, nil
	}

	// Создание временного файла хранилища, т.к. путь не передан
	if config.FileStoragePath == "" {
		file, err := os.CreateTemp(os.TempDir(), "storage*.json")
		if err != nil {
			return nil, fmt.Errorf("%s: failed create temporary file for storage: %v", prefix, err)
		}

		logger.LogS.Warnf("File storage path: %q", file.Name())
		config.FileStoragePath = file.Name()

		storage := storage.NewMemStorage()
		return &storage, nil
	}

	// Восстановление данных хранилища из json файла, при необходимости
	if config.ToRestore {
		storageFromFile, err := createStorageFromJSON(config.FileStoragePath)
		if err != nil {
			tmp := storage.NewMemStorage()
			storageFromFile = &tmp
			logger.LogS.Warnf("Failed fill storage from file: %s", err)
		}

		return storageFromFile, nil
	}

	storage := storage.NewMemStorage()
	return &storage, nil

}

// Создаёт новый сервер по переданной конфигурации
//
//	@param config   конфигурация сервера
//	@param database база данных
//	@returns *Server сервер в случае успеха
//	@returns error ошибку в ином случае
func NewServer(config *Config) (*Server, error) {
	const prefix = "server.NewServer"

	storage, err := createStorage(config)
	if err != nil {
		return nil, fmt.Errorf("%s: failed create storage: %v", prefix, err)
	}

	result := Server{
		config:  config,
		storage: storage,
	}

	if config.DatabaseDSN == nil && config.StoreInterval == 0 {
		result.doSaveStorage.Store(true)
		result.saveStorageChan = make(chan struct{})
	}

	router, err := getRouter(storage, result.getSaveMiddleware())
	if err != nil {
		return nil, fmt.Errorf("%s: failed create router: %v", prefix, err)
	}

	result.router = &router
	return &result, nil
}

// Запускает сервер на ожидание запросов. Блокирует дальнейшую работу программы
//
//	@returns error ошибку в случае неудачи
//
// SF TODO
func (s *Server) Listen(ctx context.Context) error {
	go func() {
		if s.config.StoreInterval == 0 {
			defer close(s.saveStorageChan)
			defer s.doSaveStorage.Swap(false)
			// Сохранение данных хранилища метрик в файл синхронно (по получению сигнала)
			for {
				select {
				case <-ctx.Done():
					logger.LogS.Debug("Data-saving goroutine (file output) has successfully terminated")
					return
				case <-s.saveStorageChan:
					ms, ok := s.storage.(*storage.MemStorage)
					if !ok {
						logger.Log.DPanic("Storage type is not in-memory")
						return
					}

					if err := ms.SaveStorageToFile(s.config.FileStoragePath); err != nil {
						logger.LogS.Errorf("Failed save storage to file: %v", err)
						return
					}
				}
			}
		}

		_, ok := s.storage.(*storage.MemStorage)
		if !ok {
			return
		}

		// Сохранение данных хранилища метрик в файл асинхронно (каждые N секунд, задаётся в конфигурации сервера)
		saveStorageTicker := time.NewTicker(s.config.StoreInterval)
		defer saveStorageTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.LogS.Debug("Data-saving goroutine (file output) has successfully terminated")
				return
			case <-saveStorageTicker.C:
				// Проверено перед началом сохранения по таймеру
				ms, _ := s.storage.(*storage.MemStorage)
				if err := ms.SaveStorageToFile(s.config.FileStoragePath); err != nil {
					logger.LogS.Errorf("Failed save storage to file: %v", err)
					return
				}
			}
		}
	}()

	// Запуск сервера
	logger.LogS.Info(fmt.Sprint("Server launch successful on http://", s.config.Address))
	if err := http.ListenAndServe(s.config.Address.String(), *s.router); err != http.ErrServerClosed {
		return fmt.Errorf("server: server.Listen: %v", err)
	}

	return nil
}

// SF TODO
func (s *Server) Close() error {
	return s.storage.Close()
}
