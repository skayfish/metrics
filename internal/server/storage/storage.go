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

// Ошибка: метрика не найдена в хранилище
var ErrMetricNotFound = errors.New(`metric not found`)
