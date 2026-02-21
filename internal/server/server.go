package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
	"time"

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

	// Хранилище метрик
	storage *storage.MemStorage

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
//	@param saveMiddleware middleware-обёртка для отправки сигнала на сохранение данных хранилища метрик в файл
//	@returns chi.Router маршрутизатор запросов в случае успеха
//	@returns error ошибку в ином случае
func getRouter(
	storage *storage.MemStorage,
	saveMiddleware func(http.HandlerFunc) http.HandlerFunc,
) (chi.Router, error) {
	router := chi.NewRouter()
	router.Use(middleware.CompressingMiddleware, middleware.LoggingMiddleware)

	metricsController, err := controller.NewMetricsController(storage)
	if err != nil {
		return nil, fmt.Errorf("failed create metric controller: %w", err)
	}

	router.Post("/update/{type}/{name}/{value}", saveMiddleware(metricsController.UpdateFromURL))
	router.Post("/update", saveMiddleware(metricsController.UpdateFromJSON))
	router.Post("/update/", saveMiddleware(metricsController.UpdateFromJSON))
	router.Get("/value/{type}/{name}", metricsController.GetValueFromURL)
	router.Post("/value", metricsController.GetValueFromJSON)
	router.Post("/value/", metricsController.GetValueFromJSON)
	router.Get("/", metricsController.GetAllMetrics)

	return router, nil
}

// Считывает данные метрик из json файла и создаёт из них хранилище метрик
//
//	@param filePath путь к json файлу с данными метрик
//	@returns *storage.MemStorage хранилище метрик в случае успеха
//	@returns error ошибку в ином случае
func createStorageFromJSON(filePath string) (*storage.MemStorage, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed read from file %q: %w", filePath, err)
	}

	var metrics []model.Metrics
	if err = json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("failed unmarshal metrics from file %q: %w", filePath, err)
	}

	storage := make(storage.MemStorage, len(metrics))
	for _, metric := range metrics {
		storage[metric.ID] = metric
	}

	return &storage, nil
}

// Создаёт новый сервер по переданной конфигурации
//
//	@param config конфигурация сервера
//	@returns *Server сервер в случае успеха
//	@returns error ошибку в ином случае
func NewServer(config *Config) (*Server, error) {
	var metricsStorage storage.MemStorage

	if config.FileStoragePath == "" {
		file, err := os.CreateTemp(os.TempDir(), "storage*.json")
		if err != nil {
			return nil, fmt.Errorf("server: NewServer: failed create temporary file for storage: %v", err)
		}

		logger.LogS.Warnf("File storage path: %q", file.Name())
		config.FileStoragePath = file.Name()

		metricsStorage = storage.NewMemStorage()
	} else {
		if config.ToRestore {
			storageFromFile, err := createStorageFromJSON(config.FileStoragePath)
			if err != nil {
				tmp := storage.NewMemStorage()
				storageFromFile = &tmp
				logger.LogS.Warnf("Failed fill storage from file: %s", err)
			}

			metricsStorage = *storageFromFile
		} else {
			metricsStorage = storage.NewMemStorage()
		}
	}

	result := Server{
		config:  config,
		storage: &metricsStorage,
	}

	if config.StoreInterval == 0 {
		result.doSaveStorage.Store(true)
		result.saveStorageChan = make(chan struct{})
	}

	router, err := getRouter(&metricsStorage, result.getSaveMiddleware())
	if err != nil {
		return nil, fmt.Errorf("server: NewServer: failed create router: %v", err)
	}

	result.router = &router

	return &result, nil
}

// Сохраняет данные хранилища метрик в json файл, который указан в конфигурации сервера
//
//	@returns error ошибку в случае неудачи
func (s *Server) saveStorageToFile() error {
	logger.LogS.Debugw("Save metrics storage to file", "file", s.config.FileStoragePath, "metrics storage", s.storage)

	metrics := make([]model.Metrics, 0, len(*s.storage))
	for _, metric := range *s.storage {
		metrics = append(metrics, metric)
	}

	metricsJSON, err := json.MarshalIndent(metrics, "", "    ")
	if err != nil {
		return fmt.Errorf("failed marshal metrics: %w", err)
	}

	if err = os.WriteFile(s.config.FileStoragePath, metricsJSON, 0644); err != nil {
		return fmt.Errorf("failed write to file %q: %w", s.config.FileStoragePath, err)
	}

	return nil
}

// Запускает сервер на ожидание запросов. Блокирует дальнейшую работу программы
//
//	@returns error ошибку в случае неудачи
func (s *Server) Listen() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
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
					if err := s.saveStorageToFile(); err != nil {
						logger.LogS.Errorf("Failed save storage to file: %v", err)
						return
					}
				}
			}
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
				if err := s.saveStorageToFile(); err != nil {
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
