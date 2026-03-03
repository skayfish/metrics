package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/skayfish/metrics/internal/logger"
	"github.com/skayfish/metrics/internal/server/database"
)

// Контроллер обработки базовых запросов
type BaseController struct {
	database database.Database // База данных
}

// Создаёт новый контроллер обработки базовых запросов
//
//	@param database база данных
//	@returns BaseController новый контроллер обработки базовых запросов
func NewBaseController(database database.Database) BaseController {
	return BaseController{database: database}
}

// Проверяет соединение с базой данных
//
//	@param resp объект для записи ответа
//	@param req  объект запроса
func (c *BaseController) Ping(resp http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 1*time.Second)
	defer cancel()
	if err := c.database.PingContext(ctx); err != nil {
		logger.LogS.Error("controller: BaseController.Ping: ping failed: ", err.Error())
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
}
