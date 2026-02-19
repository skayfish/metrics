package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
}

// Возвращает маршрутизатор запросов
//
//	@param storage хранилище метрик
//	@returns маршрутизатор запросов в случае успеха
//
// SF TODO
func getRouter(storage *storage.MemStorage) (chi.Router, error) {
	router := chi.NewRouter()
	router.Use(middleware.CompressingMiddleware, middleware.LoggingMiddleware)

	metricsController, err := controller.NewMetricsController(storage)
	if err != nil {
		return nil, fmt.Errorf("failed create metric controller: %w", err)
	}

	router.Post("/update/{type}/{name}/{value}", metricsController.UpdateFromURL)
	router.Post("/update", metricsController.UpdateFromJSON)
	router.Post("/update/", metricsController.UpdateFromJSON)
	router.Get("/value/{type}/{name}", metricsController.GetValueFromURL)
	router.Post("/value", metricsController.GetValueFromJSON)
	router.Post("/value/", metricsController.GetValueFromJSON)
	router.Get("/", metricsController.GetAllMetrics)

	return router, nil
}

// SF TODO
func NewServer(config *Config) (*Server, error) {
	storage := storage.NewMemStorage()
	router, err := getRouter(&storage)
	if err != nil {
		return nil, fmt.Errorf("server: NewServer: failed create router: %v", err)
	}

	return &Server{
		config:  config,
		storage: &storage,
		router:  &router,
	}, nil
}

// SF TODO
func (s *Server) saveStorageToFile() error {
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		if s.config.StoreInterval == 0 {
			return
		}

		saveStorageTicker := time.NewTicker(s.config.StoreInterval)
		defer saveStorageTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.LogS.Debug("Data‑saving goroutine (file output) has successfully terminated")
				return
			case <-saveStorageTicker.C:
				if err := s.saveStorageToFile(); err != nil {
					logger.LogS.Errorf("Failed save storage to file: %v", err)
					return
				}
			}
		}
	}()

	logger.LogS.Info(fmt.Sprint("Server launch successful on ", s.config.Address))
	if err := http.ListenAndServe(s.config.Address.String(), *s.router); err != http.ErrServerClosed {
		return fmt.Errorf("server: server.Listen: %v", err)
	}

	return nil
}
