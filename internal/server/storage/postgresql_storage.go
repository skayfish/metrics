package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/skayfish/metrics/internal/model"
)

const (
	insertGaugeQueryName = "insert_gauge"
	insertGaugeQuery     = `INSERT INTO metrics_schema.metrics (id, "type", delta, value, hash)
							VALUES ($1, $2, $3, $4, $5)
							ON CONFLICT (id)
							DO UPDATE SET
								delta = EXCLUDED.delta,
								value = EXCLUDED.value,
								hash = EXCLUDED.hash;`

	insertCounterQueryName = "insert_counter"
	insertCounterQuery     = `INSERT INTO metrics_schema.metrics (id, "type", delta, value, hash)
							VALUES ($1, $2, $3, $4, $5)
							ON CONFLICT (id)
							DO UPDATE SET
								delta = metrics_schema.metrics.delta + EXCLUDED.delta,
								value = EXCLUDED.value,
								hash = EXCLUDED.hash;`

	selectMetricQueryName = "select_metric"
	selectMetricQuery     = `SELECT * FROM metrics_schema.metrics
							WHERE id = $1;`

	selectAllMetricsQueryName = "select_all_metrics"
	selectAllMetricsQuery     = `SELECT * FROM metrics_schema.metrics;`
)

// Выполняет переданную функцию с повторениями c linear возрастающей по времени задержкой.
//
//	@param execute функция, которую нужно будет повторять, если возникает ошибка
//	@returns error возможную ошибку или nil, при отсутствии
func ExecuteWithRetry(execute func() error) error {
	const maxRetries = 4
	var lastErr error

	retryDuration := time.Second
	classifier := NewPostgresErrorClassifier()
	for attempt := 0; attempt < maxRetries; attempt++ {
		lastErr = execute()
		if lastErr == nil {
			return nil
		}

		if classifier.Classify(lastErr) == NonRetriable {
			return lastErr
		}

		if attempt+1 == maxRetries {
			break
		}

		time.Sleep(retryDuration)
		retryDuration += 2 * time.Second
	}

	return fmt.Errorf("execution aborted after %d attempts: %w", maxRetries, lastErr)
}

// Интерфейс пула соединений к базе данных PostgreSQL
type iPgxPool interface {
	Begin(context.Context) (pgx.Tx, error)
	Ping(ctx context.Context) error
	Close()
}

// Хранилище, в виде базы данных PostgreSQL
type PostgreSQLStorage struct {
	conn iPgxPool // Пул соединений к базе данных PostgreSQL
}

// Создаёт новое хранилище, в виде базы данных PostgreSQL
//
//	@param conn пул соединений к базе данных PostgreSQL
//	@returns *PostgreSQLStorage хранилище, в виде базы данных PostgreSQL
//	@returns error              ошибку, если не удалось создать хранилище
func NewPostgreSQLStorage(conn iPgxPool) (*PostgreSQLStorage, error) {
	if conn == nil {
		return nil, fmt.Errorf("database connections pool is nil")
	}

	return &PostgreSQLStorage{conn: conn}, nil
}

// Интерфейс с запросами к базе данных PostgreSQL
type psqlExecutor interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error)
}

// Конфигурация для запроса в PostgreSQL
type config struct {
	ctx context.Context // Контекст для завершения работы
	db  psqlExecutor    // База данных PostgreSQL
}

// Конфигурация для обновления/добавления метрики
type updateConfig struct {
	config

	sGauge   *pgconn.StatementDescription // Описание подготовленного запроса для обновления/добавления метрики типа gauge
	sCounter *pgconn.StatementDescription // Описание подготовленного запроса для обновления/добавления метрики типа counter
	sGet     *pgconn.StatementDescription // Описание подготовленного запроса для получения конкретной метрики
}

// Добавляет/обновляет метрику в базе данных
//
// @param cfg    конфигурация метода
// @param metric метрика для добавления/обновления
// @returns *model.Metrics добавленную/обновленную метрику
// @returns error ошибку, если не удалось добавить/обновить метрику
func updateContext(cfg updateConfig, metric model.Metrics) (*model.Metrics, error) {
	const prefix = "storage.PostgreSQLStorage.updateContext"

	var stmt *pgconn.StatementDescription
	switch metric.MType {
	case model.Gauge:
		stmt = cfg.sGauge
	case model.Counter:
		stmt = cfg.sCounter
	default:
		return nil, fmt.Errorf("%s: unknown metric type", prefix)
	}

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

	err := ExecuteWithRetry(func() error {
		_, err := cfg.db.Exec(cfg.ctx, stmt.Name, metric.ID, metric.MType, delta, value, metric.Hash)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", prefix, err)
	}

	updatedMetric, err := getContext(
		getConfig{config: cfg.config, sGet: cfg.sGet},
		metric.ID)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", prefix, err)
	}

	return updatedMetric, nil
}

// Добавляет/обновляет метрику в хранилище
//
//	@param ctx    контекст для завершения работы
//	@param metric метрика, которую нужно добавить/обновить в хранилище
//	@returns *model.Metrics обновленная метрика
//	@returns error          ошибку, если не удалось обновить метрику
func (s PostgreSQLStorage) UpdateContext(ctx context.Context, metric model.Metrics) (*model.Metrics, error) {
	const prefix = "storage.PostgreSQLStorage.UpdateContext"

	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", prefix, err)
	}

	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	cfg := updateConfig{config: config{ctx: ctx, db: tx}}

	var stmt *pgconn.StatementDescription
	stmt, err = tx.Prepare(ctx, insertGaugeQueryName, insertGaugeQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", prefix, err)
	}
	cfg.sGauge = stmt

	stmt, err = tx.Prepare(ctx, insertCounterQueryName, insertCounterQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", prefix, err)
	}
	cfg.sCounter = stmt

	stmt, err = tx.Prepare(ctx, selectMetricQueryName, selectMetricQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", prefix, err)
	}
	cfg.sGet = stmt

	updatedMetric, err := updateContext(cfg, metric)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", prefix, err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", prefix, err)
	}

	return updatedMetric, nil
}

// Добавляет/обновляет метрику в хранилище
//
//	@param metric метрика, которую нужно добавить/обновить в хранилище
//	@returns *model.Metrics обновленная метрика
//	@returns error          ошибку, если не удалось обновить метрику
func (s PostgreSQLStorage) Update(metric model.Metrics) (*model.Metrics, error) {
	return s.UpdateContext(context.Background(), metric)
}

// Добавляет/обновляет метрики в базе данных
//
//	@param ctx контекст для завершения работы
//	@param m   метрики для добавления/обновления
//	@returns []model.Metrics обновленные/добавленные метрики
//	@returns error ошибку, если не удалось добавить/обновить метрики
func (s PostgreSQLStorage) UpdatesContext(ctx context.Context, m []model.Metrics) ([]model.Metrics, error) {
	const prefix = "storage.PostgreSQLStorage.UpdatesContext"

	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", prefix, err)
	}

	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	cfg := updateConfig{config: config{ctx: ctx, db: tx}}

	var stmt *pgconn.StatementDescription
	stmt, err = tx.Prepare(ctx, insertGaugeQueryName, insertGaugeQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", prefix, err)
	}
	cfg.sGauge = stmt

	stmt, err = tx.Prepare(ctx, insertCounterQueryName, insertCounterQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", prefix, err)
	}
	cfg.sCounter = stmt

	stmt, err = tx.Prepare(ctx, selectMetricQueryName, selectMetricQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", prefix, err)
	}
	cfg.sGet = stmt

	updatedMetrics := []model.Metrics{}
	for _, metric := range m {
		updatedMetric, err := updateContext(cfg, metric)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", prefix, err)
		}

		updatedMetrics = append(updatedMetrics, *updatedMetric)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", prefix, err)
	}

	return updatedMetrics, nil
}

// Добавляет/обновляет метрики в базе данных
//
//	@param m метрики для добавления/обновления
//	@returns []model.Metrics обновленные/добавленные метрики
//	@returns error ошибку, если не удалось добавить/обновить метрики
func (s PostgreSQLStorage) Updates(m []model.Metrics) ([]model.Metrics, error) {
	return s.UpdatesContext(context.Background(), m)
}

// Конфигурация для получения конкретной метрики
type getConfig struct {
	config

	sGet *pgconn.StatementDescription // Описание подготовленного запроса для получения конкретной метрики
}

// Возвращает конкретную метрику из хранилища
//
//	@param cfg конфигурация для получения конкретной метрики
//	@param id  идентификатор метрики
//	@returns *model.Metrics найденную метрику
//	@returns error          ошибку, если возникли проблемы при поиске метрики
func getContext(cfg getConfig, id string) (*model.Metrics, error) {
	const prefix = "storage.PostgreSQLStorage.getContext"

	var (
		mType     string
		deltaNull sql.NullInt64
		valueNull sql.NullFloat64
		hash      string
	)
	err := ExecuteWithRetry(func() error {
		return cfg.db.QueryRow(cfg.ctx, cfg.sGet.Name, id).
			Scan(&id, &mType, &deltaNull, &valueNull, &hash)
	})
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
	const prefix = "storage.PostgreSQLStorage.GetContext"

	// Создание транзакции
	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", prefix, err)
	}

	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// Подготовка конфигурации для запроса
	cfg := getConfig{config: config{ctx: ctx, db: tx}}

	stmt, err := tx.Prepare(ctx, selectMetricQueryName, selectMetricQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", prefix, err)
	}
	cfg.sGet = stmt

	metric, err := getContext(cfg, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", prefix, err)
	}

	// Подтверждение транзакции
	err = tx.Commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", prefix, err)
	}

	return metric, nil
}

// Возвращает конкретную метрику из хранилища
//
//	@param id идентификатор метрики
//	@returns *model.Metrics найденную метрику
//	@returns error          ошибку, если возникли проблемы при поиске метрики
func (s PostgreSQLStorage) Get(id string) (*model.Metrics, error) {
	return s.GetContext(context.Background(), id)
}

// Конфигурация для получения всех метрик
type getAllConfig struct {
	config

	sGetAll *pgconn.StatementDescription // Описание подготовленного запроса для получения всех метрик
}

// Возвращает все метрики из хранилища
//
//	@param cfg конфигурация для получения всех метрик
//	@returns []model.Metrics все метрики из хранилища
//	@returns error           ошибку, если возникли проблемы при получении всех метрик
func getAllContext(cfg getAllConfig) (result []model.Metrics, err error) {
	const prefix = "storage.PostgreSQLStorage.getAllContext"

	err = ExecuteWithRetry(func() error {
		result = make([]model.Metrics, 0)
		rows, err := cfg.db.Query(cfg.ctx, cfg.sGetAll.Name)
		if err != nil {
			return err
		}

		defer rows.Close()

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
				return err
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

		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %v", prefix, err)
	}

	return result, nil
}

// Возвращает все метрики из хранилища
//
//	@param ctx контекст для завершения работы
//	@returns []model.Metrics все метрики из хранилища
//	@returns error           ошибку, если возникли проблемы при получении всех метрик
func (s PostgreSQLStorage) GetAllContext(ctx context.Context) ([]model.Metrics, error) {
	const prefix = "storage.PostgreSQLStorage.GetAllContext"

	// Создание транзакции
	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", prefix, err)
	}

	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// Подготовка конфигурации для запроса
	cfg := getAllConfig{config: config{ctx: ctx, db: tx}}

	stmt, err := tx.Prepare(ctx, selectAllMetricsQueryName, selectAllMetricsQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", prefix, err)
	}
	cfg.sGetAll = stmt

	// Получение всех метрик из бд
	metrics, err := getAllContext(cfg)
	if err != nil {
		return []model.Metrics{}, fmt.Errorf("%s: %v", prefix, err)
	}

	// Подтверждение транзакции
	err = tx.Commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", prefix, err)
	}

	return metrics, nil
}

// Возвращает все метрики из хранилища
//
//	@returns []model.Metrics все метрики из хранилища
//	@returns error           ошибку, если возникли проблемы при получении всех метрик
func (s PostgreSQLStorage) GetAll() ([]model.Metrics, error) {
	return s.GetAllContext(context.Background())
}

// Завершает работу хранилища в виде базы данных PostgreSQL
//
//	@returns error nil
func (s *PostgreSQLStorage) Close() error {
	s.conn.Close()
	return nil
}

// Проверяет связь с базой данных PostgreSQL
//
//	@param ctx контекст для завершения работы
//	@returns error ошибку, в случае если связь нарушена
func (s PostgreSQLStorage) PingContext(ctx context.Context) error {
	return s.conn.Ping(ctx)
}

// Проверка, что [PostgreSQLStorage] удовлетворяет интерфейсу [DatabaseStorage]
var _ DatabaseStorage = (*PostgreSQLStorage)(nil)
