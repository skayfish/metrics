package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/skayfish/metrics/internal/model"
)

// SF TODO
type PostgreSQLDatabase struct {
	*sql.DB
}

// SF TODO
func (db PostgreSQLDatabase) Update(metric model.Metrics) (*model.Metrics, error) {
	return db.UpdateContext(context.Background(), metric)
}

// SF TODO
func (db PostgreSQLDatabase) UpdateContext(ctx context.Context, metric model.Metrics) (*model.Metrics, error) {
	const prefix = "storage.PostgreSQLDatabase.UpdateContext"

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

	tx, err := db.BeginTx(ctx, nil)
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
		_, err = tx.ExecContext(ctx, `
			INSERT INTO metrics_schema.metrics (id, "type", delta, value, hash)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (id)
			DO UPDATE SET
				delta = EXCLUDED.delta,
				value = EXCLUDED.value,
				hash = EXCLUDED.hash;`,
			metric.ID, metric.MType, delta, value, metric.Hash)

	case model.Counter:
		_, err = tx.ExecContext(ctx, `
			INSERT INTO metrics_schema.metrics (id, "type", delta, value, hash)
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

// SF TODO
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// SF TODO
func getContext(ctx context.Context, db SQLExecutor, id string) (*model.Metrics, error) {
	const prefix = "storage.PostgreSQLDatabase.getContext"

	row := db.QueryRowContext(ctx, `
		SELECT * FROM metrics_schema.metrics
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

// SF TODO
func (db PostgreSQLDatabase) GetContext(ctx context.Context, id string) (*model.Metrics, error) {
	return getContext(ctx, db.DB, id)
}

// SF TODO
func (db PostgreSQLDatabase) Get(id string) (*model.Metrics, error) {
	return db.GetContext(context.Background(), id)
}

// SF TODO
func (db PostgreSQLDatabase) GetAll() ([]model.Metrics, error) {
	return db.GetAllContext(context.Background())
}

// SF TODO
func (db PostgreSQLDatabase) GetAllContext(ctx context.Context) ([]model.Metrics, error) {
	const prefix = "storage.PostgreSQLDatabase.GetAllContext"

	rows, err := db.QueryContext(ctx, `SELECT * FROM metrics_schema.metrics;`)
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
			delta = &deltaNull.Int64
		}

		var value *float64
		if valueNull.Valid {
			value = &valueNull.Float64
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

// SF TODO
func (db PostgreSQLDatabase) Close() error {
	return db.DB.Close()
}

// SF TODO
var _ Database = (*PostgreSQLDatabase)(nil)
