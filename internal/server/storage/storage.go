package storage

import (
	"context"
	"errors"

	"github.com/skayfish/metrics/internal/model"
)

// Интерфейс хранилища данных
type Storage interface {
	// Добавляет/обновляет метрику в хранилище
	//
	//	@param m метрика для добавления/обновления
	//	@returns *model.Metrics добавленную/обновленную метрику
	//	@returns error          ошибку, если возникли проблемы при добавлении/обновлении метрики
	Update(m model.Metrics) (*model.Metrics, error)

	// Добавляет/обновляет метрику в хранилище
	//
	//	@param ctx контекст для завершения работы
	//	@param m   метрика для добавления/обновления
	//	@returns *model.Metrics добавленную/обновленную метрику
	//	@returns error          ошибку, если возникли проблемы при добавлении/обновлении метрики
	UpdateContext(ctx context.Context, m model.Metrics) (*model.Metrics, error)

	// Добавляет/обновляет набор метрик в хранилище
	//
	//	@param m набор метрик для добавления/обновления
	//	@returns []model.Metrics добавленный/обновленный набор метрик
	//	@returns error           ошибку, если возникли проблемы при добавлении/обновлении набора метрик
	Updates(m []model.Metrics) ([]model.Metrics, error)

	// Добавляет/обновляет набор метрик в хранилище
	//
	//	@param ctx контекст для завершения работы
	//	@param m набор метрик для добавления/обновления
	//	@returns []model.Metrics добавленный/обновленный набор метрик
	//	@returns error           ошибку, если возникли проблемы при добавлении/обновлении набора метрик
	UpdatesContext(ctx context.Context, m []model.Metrics) ([]model.Metrics, error)

	// Возвращает конкретную метрику из хранилища
	//
	//	@param id идентификатор метрики
	//	@returns *model.Metrics найденную метрику
	//	@returns error          ошибку, если возникли проблемы при поиске метрики
	Get(id string) (*model.Metrics, error)

	// Возвращает конкретную метрику из хранилища
	//
	//	@param ctx контекст для завершения работы
	//	@param id  идентификатор метрики
	//	@returns *model.Metrics найденную метрику
	//	@returns error          ошибку, если возникли проблемы при получении метрики
	GetContext(ctx context.Context, id string) (*model.Metrics, error)

	// Возвращает все метрики из хранилища
	//
	//	@returns []model.Metrics все метрики из хранилища
	//	@returns error           ошибку, если возникли проблемы при получении всех метрик
	GetAll() ([]model.Metrics, error)

	// Возвращает все метрики из хранилища
	//
	//	@param ctx контекст для завершения работы
	//	@returns []model.Metrics все метрики из хранилища
	//	@returns error           ошибку, если возникли проблемы при получении всех метрик
	GetAllContext(ctx context.Context) ([]model.Metrics, error)

	// Завершает работу хранилища
	//
	//	@returns error ошибку, если возникли проблемы при завершении работы хранилища
	Close() error
}

// Интерфейс базы данных
type Database interface {
	// Проверяет, что соединение с базой данных всё ещё активно
	//
	//	@param ctx контекст для завершения
	//	@returns error ошибку, если соединение не активно
	PingContext(ctx context.Context) error
}

//go:generate mockgen --destination=mock_database_storage.go --package=storage github.com/skayfish/metrics/internal/server/storage DatabaseStorage

// Интерфейс хранилища, в виде базы данных
type DatabaseStorage interface {
	Storage
	Database
}

// Ошибка: метрика не найдена в хранилище
var ErrMetricNotFound = errors.New(`metric not found`)
