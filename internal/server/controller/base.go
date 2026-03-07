package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/skayfish/metrics/internal/logger"
	"github.com/skayfish/metrics/internal/server/storage"
)

// Контроллер обработки базовых запросов
type BaseController struct {
	storage storage.Storage // Хранилище метрик
}

// Создаёт новый контроллер обработки базовых запросов
//
//	@param database база данных
//	@returns BaseController новый контроллер обработки базовых запросов
//
// SF TODO
func NewBaseController(storage storage.Storage) BaseController {
	return BaseController{storage: storage}
}

// Проверяет соединение с базой данных
//
//	@param resp объект для записи ответа
//	@param req  объект запроса
func (c *BaseController) Ping(resp http.ResponseWriter, req *http.Request) {
	const prefix = "controller: BaseController.Ping"

	switch storage := c.storage.(type) {
	case storage.Database:
		ctx, cancel := context.WithTimeout(req.Context(), 1*time.Second)
		defer cancel()
		if err := storage.PingContext(ctx); err != nil {
			logger.LogS.Errorf("%s: ping failed: %v", prefix, err)
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}

	case *storage.MemStorage:
		return

	default:
		logger.LogS.Warnf("%s: unknown storage type", prefix)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
}
