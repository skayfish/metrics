package server

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/skayfish/metrics/internal/logger"
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
func getRouter(controller *controller.MetricsController) chi.Router {
	router := chi.NewRouter()
	router.Use(middleware.CompressingMiddleware, middleware.LoggingMiddleware)
	router.Post("/update/{type}/{name}/{value}", controller.UpdateFromURL)
	router.Post("/update", controller.UpdateFromJSON)
	router.Post("/update/", controller.UpdateFromJSON)
	router.Get("/value/{type}/{name}", controller.GetValueFromURL)
	router.Post("/value", controller.GetValueFromJSON)
	router.Post("/value/", controller.GetValueFromJSON)
	router.Get("/", controller.GetAllMetrics)

	return router
}

// SF TODO
func NewServer(config *Config) (*Server, error) {
	storage := storage.NewMemStorage()
	metricsController, err := controller.NewMetricsController(&storage)
	if err != nil {
		return nil, fmt.Errorf("server: NewServer: %v", err)
	}

	// Настройка маршрутизации запросов
	router := getRouter(metricsController)

	return &Server{
		config:  config,
		storage: &storage,
		router:  &router,
	}, nil
}

// SF TODO
func (s *Server) Listen() error {
	logger.LogS.Info(fmt.Sprint("Server launch successful on ", s.config.Address))
	if err := http.ListenAndServe(s.config.Address.String(), *s.router); err != http.ErrServerClosed {
		return fmt.Errorf("server: server.Listen: %v", err)
	}

	return nil
}
