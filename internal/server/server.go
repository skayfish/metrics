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

// SF TODO
type Server struct {
	// SF TODO
	config *Config

	// SF TODO
	storage *storage.MemStorage

	// SF TODO
	router *chi.Router

	// SF TODO
	saveStorageChan chan struct{}

	// SF TODO
	doSaveStorage atomic.Bool
}

// SF TODO
func (s *Server) getSaveMiddleware() func(http.HandlerFunc) http.HandlerFunc {
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
//	@param storage хранилище метрик
//	@returns маршрутизатор запросов в случае успеха
//
// SF TODO
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

// SF TODO
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

// SF TODO
func NewServer(config *Config) (*Server, error) {
	var metricsStorage storage.MemStorage
	if config.ToRestore {
		if config.FileStoragePath == "" {
			file, err := os.CreateTemp(os.TempDir(), "storage*.json")
			if err != nil {
				return nil, fmt.Errorf("server: NewServer: failed create temporary file for storage: %v", err)
			}

			logger.LogS.Warnf("File storage path: %q", file.Name())
			config.FileStoragePath = file.Name()

			metricsStorage = storage.NewMemStorage()
		} else {
			storageFromFile, err := createStorageFromJSON(config.FileStoragePath)
			if err != nil {
				tmp := storage.NewMemStorage()
				storageFromFile = &tmp
				logger.LogS.Warnf("Failed fill storage from file: %s", err)
			}

			metricsStorage = *storageFromFile
		}
	} else {
		metricsStorage = storage.NewMemStorage()
	}

	result := Server{
		config:          config,
		storage:         &metricsStorage,
		saveStorageChan: make(chan struct{}),
	}
	router, err := getRouter(&metricsStorage, result.getSaveMiddleware())
	if err != nil {
		return nil, fmt.Errorf("server: NewServer: failed create router: %v", err)
	}

	result.router = &router

	return &result, nil
}

// SF TODO
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

// SF TODO
func (s *Server) Listen() error {
	defer close(s.saveStorageChan)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		if s.config.StoreInterval == 0 {
			for {
				select {
				case <-ctx.Done():
					logger.LogS.Debug("Data-saving goroutine (file output) has successfully terminated")
					return
				case <-s.saveStorageChan:
					if err := s.saveStorageToFile(); err != nil {
						logger.LogS.Errorf("Failed save storage to file: %v", err)
						close(s.saveStorageChan)
						s.doSaveStorage.Swap(false)
						return
					}
				}
			}
		}

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
			case <-s.saveStorageChan:
			}
		}
	}()

	logger.LogS.Info(fmt.Sprint("Server launch successful on http://", s.config.Address))
	if err := http.ListenAndServe(s.config.Address.String(), *s.router); err != http.ErrServerClosed {
		return fmt.Errorf("server: server.Listen: %v", err)
	}

	return nil
}
