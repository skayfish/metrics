package storage

import (
	"context"
	"errors"

	"github.com/skayfish/metrics/internal/model"
)

// SF TODO
type Storage interface {
	// SF TODO
	Update(m model.Metrics) (*model.Metrics, error)

	// SF TODO
	UpdateContext(ctx context.Context, m model.Metrics) (*model.Metrics, error)

	// SF TODO
	Get(id string) (*model.Metrics, error)

	// SF TODO
	GetContext(ctx context.Context, id string) (*model.Metrics, error)

	// SF TODO
	GetAll() ([]model.Metrics, error)

	// SF TODO
	GetAllContext(ctx context.Context) ([]model.Metrics, error)

	// SF TODO
	Close() error
}

// SF TODO
type Database interface {
	// Проверяет, что соединение с базой данных всё ещё активно
	//
	//	@param ctx контекст для завершения
	//	@returns error ошибку, если соединение не активно
	PingContext(ctx context.Context) error
}

//go:generate mockgen --destination=mock_database_storage.go --package=storage github.com/skayfish/metrics/internal/server/storage DatabaseStorage

// SF TODO
type DatabaseStorage interface {
	Storage
	Database
}

// Ошибка: метрика не найдена в хранилище
var ErrMetricNotFound = errors.New(`metric not found`)
