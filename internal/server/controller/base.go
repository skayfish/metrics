package controller

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/skayfish/metrics/internal/logger"
	"github.com/skayfish/metrics/internal/server/storage"
)

// SF TODO
type BaseController struct {
	storage  *storage.MemStorage // Хранилище метрик
	database *sql.DB             // База данных
}

// SF TODO
func NewBaseController(storage *storage.MemStorage, database *sql.DB) BaseController {
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
