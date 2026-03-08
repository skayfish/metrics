package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/skayfish/metrics/internal/model"
)

// Хранилище, в виде базы данных PostgreSQL
type PostgreSQLStorage struct {
	*sql.DB
}

// Создаёт новое хранилище, в виде базы данных PostgreSQL
//
//	@param db база данных PostgreSQL
//	@returns *PostgreSQLStorage хранилище, в виде базы данных PostgreSQL
//	@returns error              ошибку, если не удалось создать хранилище
func NewPostgreSQLStorage(db *sql.DB) (*PostgreSQLStorage, error) {
	if db == nil {
		return nil, fmt.Errorf("database is nil")
	}

	return &PostgreSQLStorage{DB: db}, nil
}

// Добавляет/обновляет метрику в хранилище
//
//	@param metric метрика, которую нужно добавить/обновить в хранилище
//	@returns *model.Metrics обновленная метрика
//	@returns error          ошибку, если не удалось обновить метрику
func (s PostgreSQLStorage) Update(metric model.Metrics) (*model.Metrics, error) {
	return s.UpdateContext(context.Background(), metric)
}

// Добавляет/обновляет метрику в хранилище
//
//	@param ctx    контекст для завершения работы
//	@param metric метрика, которую нужно добавить/обновить в хранилище
//	@returns *model.Metrics обновленная метрика
//	@returns error          ошибку, если не удалось обновить метрику
func (s PostgreSQLStorage) UpdateContext(ctx context.Context, metric model.Metrics) (*model.Metrics, error) {
	const prefix = "storage.PostgreSQLStorage.UpdateContext"

	var delta sql.NullInt64
	if metric.Delta != nil {
		delta.Int64 = *metric.Delta
		delta.Valid = true
	}

	var value sql.NullFloat64
	if metric.Value != nil {
		value.Float64 = *metric.Value
		value.Valid = true
	}

	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", prefix, err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	switch metric.MType {
	case model.Gauge:
		_, err = tx.ExecContext(ctx,
			`INSERT INTO metrics_schema.metrics (id, "type", delta, value, hash)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (id)
			DO UPDATE SET
				delta = EXCLUDED.delta,
				value = EXCLUDED.value,
				hash = EXCLUDED.hash;`,
			metric.ID, metric.MType, delta, value, metric.Hash)

	case model.Counter:
		_, err = tx.ExecContext(ctx,
			`INSERT INTO metrics_schema.metrics (id, "type", delta, value, hash)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (id)
			DO UPDATE SET
				delta = metrics_schema.metrics.delta + EXCLUDED.delta,
				value = EXCLUDED.value,
				hash = EXCLUDED.hash;`,
			metric.ID, metric.MType, delta, value, metric.Hash)
	}

	if err != nil {
		return nil, fmt.Errorf("%s: %w", prefix, err)
	}

	updatedMetric, err := getContext(ctx, tx, metric.ID)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", prefix, err)
	}

	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("%s: %v", prefix, err)
	}

	return updatedMetric, nil
}

// Возвращает конкретную метрику из хранилища
//
//	@param ctx контекст для завершения работы
//	@param db  база данных
//	@param id  идентификатор метрики
//	@returns *model.Metrics найденную метрику
//	@returns error          ошибку, если возникли проблемы при поиске метрики
func getContext(ctx context.Context, db SQLExecutor, id string) (*model.Metrics, error) {
	const prefix = "storage.PostgreSQLStorage.getContext"

	row := db.QueryRowContext(ctx,
		`SELECT * FROM metrics_schema.metrics
		WHERE id = $1;`, id)
	var (
		mType     string
		deltaNull sql.NullInt64
		valueNull sql.NullFloat64
		hash      string
	)

	err := row.Scan(&id, &mType, &deltaNull, &valueNull, &hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", prefix, ErrMetricNotFound)
		}

		return nil, fmt.Errorf("%s: %w", prefix, err)
	}

	var delta *int64
	if deltaNull.Valid {
		delta = &deltaNull.Int64
	}

	var value *float64
	if valueNull.Valid {
		value = &valueNull.Float64
	}

	return &model.Metrics{
		ID:    id,
		MType: mType,
		Delta: delta,
		Value: value,
		Hash:  hash,
	}, nil
}

// Возвращает конкретную метрику из хранилища
//
//	@param ctx контекст для завершения работы
//	@param id  идентификатор метрики
//	@returns *model.Metrics найденную метрику
//	@returns error          ошибку, если возникли проблемы при получении метрики
func (s PostgreSQLStorage) GetContext(ctx context.Context, id string) (*model.Metrics, error) {
	return getContext(ctx, s.DB, id)
}

// Возвращает конкретную метрику из хранилища
//
//	@param id идентификатор метрики
//	@returns *model.Metrics найденную метрику
//	@returns error          ошибку, если возникли проблемы при поиске метрики
func (s PostgreSQLStorage) Get(id string) (*model.Metrics, error) {
	return s.GetContext(context.Background(), id)
}

// Возвращает все метрики из хранилища
//
//	@returns []model.Metrics все метрики из хранилища
//	@returns error           ошибку, если возникли проблемы при получении всех метрик
func (s PostgreSQLStorage) GetAll() ([]model.Metrics, error) {
	return s.GetAllContext(context.Background())
}

// Возвращает все метрики из хранилища
//
//	@param ctx контекст для завершения работы
//	@returns []model.Metrics все метрики из хранилища
//	@returns error           ошибку, если возникли проблемы при получении всех метрик
func (s PostgreSQLStorage) GetAllContext(ctx context.Context) ([]model.Metrics, error) {
	const prefix = "storage.PostgreSQLStorage.GetAllContext"

	rows, err := s.DB.QueryContext(ctx, `SELECT * FROM metrics_schema.metrics;`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", prefix, err)
	}

	defer rows.Close()

	result := make([]model.Metrics, 0)
	var (
		id        string
		mType     string
		deltaNull sql.NullInt64
		valueNull sql.NullFloat64
		hash      string
	)
	for rows.Next() {
		err := rows.Scan(&id, &mType, &deltaNull, &valueNull, &hash)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", prefix, err)
		}

		var delta *int64
		if deltaNull.Valid {
			tmp := deltaNull.Int64
			delta = &tmp
		}

		var value *float64
		if valueNull.Valid {
			tmp := valueNull.Float64
			value = &tmp
		}

		result = append(result, model.Metrics{
			ID:    id,
			MType: mType,
			Delta: delta,
			Value: value,
			Hash:  hash,
		})
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", prefix, err)
	}

	return result, nil
}

// Проверка, что [PostgreSQLStorage] удовлетворяет интерфейсу [DatabaseStorage]
var _ DatabaseStorage = (*PostgreSQLStorage)(nil)
