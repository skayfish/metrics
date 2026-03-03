package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/skayfish/metrics/internal/logger"
	"github.com/skayfish/metrics/internal/server/database"
	"github.com/skayfish/metrics/internal/server/storage"
)

// SF TODO
type BaseController struct {
	storage  *storage.MemStorage // Хранилище метрик
	database database.Database   // База данных
}

// SF TODO
func NewBaseController(storage *storage.MemStorage, database database.Database) BaseController {
	return BaseController{storage: storage, database: database}
}

// SF TODO
func (c *BaseController) Ping(resp http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 1*time.Second)
	defer cancel()
	if err := c.database.PingContext(ctx); err != nil {
		logger.LogS.Error("controller: BaseController.Ping: ping failed: ", err.Error())
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
}
