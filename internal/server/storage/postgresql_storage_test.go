package storage

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/skayfish/metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	mockInsertGaugeQuery = `INSERT INTO metrics_schema\.metrics \(id, "type", delta, value, hash\)
							VALUES \(\$1, \$2, \$3, \$4, \$5\)
							ON CONFLICT \(id\)
							DO UPDATE SET
								delta = EXCLUDED\.delta,
								value = EXCLUDED\.value,
								hash = EXCLUDED\.hash;`

	mockInsertCounterQuery = `INSERT INTO metrics_schema\.metrics \(id, "type", delta, value, hash\)
							VALUES \(\$1, \$2, \$3, \$4, \$5\)
							ON CONFLICT \(id\)
							DO UPDATE SET
								delta = metrics_schema\.metrics\.delta \+ EXCLUDED\.delta,
								value = EXCLUDED\.value,
								hash = EXCLUDED\.hash;`

	mockSelectMetricQuery = `SELECT \* FROM metrics_schema\.metrics
							WHERE id = \$1;`

	mockSelectAllMetricsQuery = `SELECT \* FROM metrics_schema\.metrics;`
)

// Проверяет создание хранилища в виде базы данных PostgreSQL
func TestNewPostgreSQLStorage(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tests := []struct {
		test    string
		db      *sql.DB
		want    *PostgreSQLStorage
		wantErr bool
	}{
		{
			test:    "failed",
			db:      nil,
			want:    nil,
			wantErr: true,
		},
		{
			test:    "success",
			db:      db,
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.test, func(t *testing.T) {
			database, err := NewPostgreSQLStorage(tt.db)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, "database is nil", err.Error())
				require.Nil(t, database)
			} else {
				require.NoError(t, err)
				require.NotNil(t, database)
			}
		})
	}
}

// Проверяет добавление/обновление новых метрик в хранилище PostgreSQL
func TestPostgreSQLStorage_Update(t *testing.T) {
	t.Run("add counter", func(t *testing.T) {
		delta := int64(5)
		expected := model.Metrics{
			ID:    "counterID",
			MType: model.Counter,
			Delta: &delta,
		}

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectPrepare(mockInsertGaugeQuery)
		mock.ExpectPrepare(mockInsertCounterQuery)
		mock.ExpectPrepare(mockSelectMetricQuery)
		mock.ExpectExec(mockInsertCounterQuery).
			WithArgs(
				expected.ID,
				expected.MType,
				sql.NullInt64{Valid: true, Int64: *expected.Delta},
				sql.NullFloat64{Valid: false},
				expected.Hash,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		expectedRow := sqlmock.NewRows([]string{"id", "type", "delta", "value", "hash"}).
			AddRow(
				expected.ID,
				expected.MType,
				*expected.Delta,
				expected.Value,
				expected.Hash)
		mock.ExpectQuery(mockSelectMetricQuery).
			WithArgs(expected.ID).
			WillReturnRows(expectedRow)
		mock.ExpectCommit()

		storage, err := NewPostgreSQLStorage(db)
		require.NoError(t, err)

		// Обновление метрики в бд
		metric, err := storage.Update(expected)
		require.NoError(t, err)

		metricsEqual(t, expected, *metric)

		// Проверка мок вызовов
		err = mock.ExpectationsWereMet()
		require.NoError(t, err)
	})
	t.Run("add gauge", func(t *testing.T) {
		value := 0.15
		expected := model.Metrics{
			ID:    "gaugeID",
			MType: model.Gauge,
			Value: &value,
		}

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectPrepare(mockInsertGaugeQuery)
		mock.ExpectPrepare(mockInsertCounterQuery)
		mock.ExpectPrepare(mockSelectMetricQuery)
		mock.ExpectExec(mockInsertGaugeQuery).
			WithArgs(
				expected.ID,
				expected.MType,
				sql.NullInt64{Valid: false},
				sql.NullFloat64{Valid: true, Float64: *expected.Value},
				expected.Hash,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		expectedRow := sqlmock.NewRows([]string{"id", "type", "delta", "value", "hash"}).
			AddRow(
				expected.ID,
				expected.MType,
				expected.Delta,
				*expected.Value,
				expected.Hash)
		mock.ExpectQuery(mockSelectMetricQuery).
			WithArgs(expected.ID).
			WillReturnRows(expectedRow)
		mock.ExpectCommit()

		storage, err := NewPostgreSQLStorage(db)
		require.NoError(t, err)

		// Обновление метрики в бд
		metric, err := storage.Update(expected)
		require.NoError(t, err)

		metricsEqual(t, expected, *metric)

		// Проверка мок вызовов
		err = mock.ExpectationsWereMet()
		require.NoError(t, err)
	})
	t.Run("rollback", func(t *testing.T) {
		delta := int64(5)
		expected := model.Metrics{
			ID:    "counterID",
			MType: model.Counter,
			Delta: &delta,
		}

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		errorMessage := "some error"

		mock.ExpectBegin()
		mock.ExpectPrepare(mockInsertGaugeQuery)
		mock.ExpectPrepare(mockInsertCounterQuery)
		mock.ExpectPrepare(mockSelectMetricQuery)
		mock.ExpectExec(mockInsertCounterQuery).
			WithArgs(
				expected.ID,
				expected.MType,
				sql.NullInt64{Valid: true, Int64: *expected.Delta},
				sql.NullFloat64{Valid: false},
				expected.Hash,
			).
			WillReturnError(errors.New(errorMessage))
		mock.ExpectRollback()

		storage, err := NewPostgreSQLStorage(db)
		require.NoError(t, err)

		// Обновление метрики в бд
		metric, err := storage.Update(expected)
		require.Error(t, err)
		assert.Contains(t, err.Error(), errorMessage)
		require.Nil(t, metric)

		// Проверка мок вызовов
		err = mock.ExpectationsWereMet()
		require.NoError(t, err)
	})
}

// SF TODO
func TestPostgreSQLStorage_Updates(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		delta1 := int64(5)
		delta2 := int64(-774)
		value1 := 0.5531
		value2 := -0.00041
		expected := []model.Metrics{
			{ // #0
				ID:    "counterID1",
				MType: model.Counter,
				Delta: &delta1,
			},
			{ // #1
				ID:    "gaugeID1",
				MType: model.Gauge,
				Value: &value1,
			},
			{ // #2
				ID:    "gaugeID2",
				MType: model.Gauge,
				Value: &value2,
			},
			{ // #3
				ID:    "counterID2",
				MType: model.Counter,
				Delta: &delta2,
			},
		}

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectPrepare(mockInsertGaugeQuery)
		mock.ExpectPrepare(mockInsertCounterQuery)
		mock.ExpectPrepare(mockSelectMetricQuery)
		for _, metric := range expected {
			delta := sql.NullInt64{Valid: false}
			value := sql.NullFloat64{Valid: false}
			switch metric.MType {
			case model.Gauge:
				value = sql.NullFloat64{Valid: true, Float64: *metric.Value}
				mock.ExpectExec(mockInsertGaugeQuery).
					WithArgs(
						metric.ID,
						metric.MType,
						delta,
						value,
						metric.Hash,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			case model.Counter:
				delta = sql.NullInt64{Valid: true, Int64: *metric.Delta}
				mock.ExpectExec(mockInsertCounterQuery).
					WithArgs(
						metric.ID,
						metric.MType,
						delta,
						value,
						metric.Hash,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			}

			expectedRow := sqlmock.NewRows([]string{"id", "type", "delta", "value", "hash"})
			switch metric.MType {
			case model.Gauge:
				expectedRow.AddRow(metric.ID, metric.MType, metric.Delta, *metric.Value, metric.Hash)
			case model.Counter:
				expectedRow.AddRow(metric.ID, metric.MType, *metric.Delta, metric.Value, metric.Hash)
			}

			mock.ExpectQuery(mockSelectMetricQuery).
				WithArgs(metric.ID).
				WillReturnRows(expectedRow)
		}

		mock.ExpectCommit()

		storage, err := NewPostgreSQLStorage(db)
		require.NoError(t, err)

		// Обновление метрики в бд
		updatedMetric, err := storage.Updates(expected)
		require.NoError(t, err)

		storagesEqual(t, storageByMetricsArray(expected), storageByMetricsArray(updatedMetric))

		// Проверка мок вызовов
		err = mock.ExpectationsWereMet()
		require.NoError(t, err)
	})
}

// Проверяет получение метрики из хранилища PostgreSQL
func TestPostgreSQLStorage_Get(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectPrepare(mockSelectMetricQuery)

		expectedRow := sqlmock.NewRows([]string{"id", "type", "delta", "value", "hash"})
		id := "MetricName"
		mock.ExpectQuery(mockSelectMetricQuery).
			WithArgs(id).
			WillReturnRows(expectedRow)

		storage, err := NewPostgreSQLStorage(db)
		require.NoError(t, err)

		// Получение метрики из бд
		metric, err := storage.Get(id)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrMetricNotFound)
		require.Nil(t, metric)

		// Проверка мок вызовов
		err = mock.ExpectationsWereMet()
		require.NoError(t, err)
	})
	t.Run("failed scan", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectPrepare(mockSelectMetricQuery)

		expectedRow := sqlmock.NewRows([]string{"id"}).
			AddRow("error id")
		id := "MetricName"
		mock.ExpectQuery(mockSelectMetricQuery).
			WithArgs(id).
			WillReturnRows(expectedRow)

		storage, err := NewPostgreSQLStorage(db)
		require.NoError(t, err)

		// Получение метрики из бд
		metric, err := storage.Get(id)
		require.Error(t, err)
		require.Nil(t, metric)

		// Проверка мок вызовов
		err = mock.ExpectationsWereMet()
		require.NoError(t, err)
	})
	t.Run("found gauge", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectPrepare(mockSelectMetricQuery)

		value := 0.15
		expectedMetric := model.Metrics{
			ID:    "gaugeID",
			MType: model.Gauge,
			Value: &value,
		}
		expectedRow := sqlmock.NewRows([]string{"id", "type", "delta", "value", "hash"}).
			AddRow(
				expectedMetric.ID,
				expectedMetric.MType,
				expectedMetric.Delta,
				*expectedMetric.Value,
				expectedMetric.Hash)
		mock.ExpectQuery(mockSelectMetricQuery).
			WithArgs(expectedMetric.ID).
			WillReturnRows(expectedRow)

		storage, err := NewPostgreSQLStorage(db)
		require.NoError(t, err)

		// Получение метрики из бд
		metric, err := storage.Get(expectedMetric.ID)
		require.NoError(t, err)
		require.NotNil(t, metric)

		metricsEqual(t, expectedMetric, *metric)

		// Проверка мок вызовов
		err = mock.ExpectationsWereMet()
		require.NoError(t, err)
	})
	t.Run("found counter", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectPrepare(mockSelectMetricQuery)

		delta := int64(5)
		expectedMetric := model.Metrics{
			ID:    "counterID",
			MType: model.Counter,
			Delta: &delta,
		}
		expectedRow := sqlmock.NewRows([]string{"id", "type", "delta", "value", "hash"}).
			AddRow(
				expectedMetric.ID,
				expectedMetric.MType,
				*expectedMetric.Delta,
				expectedMetric.Value,
				expectedMetric.Hash)
		mock.ExpectQuery(mockSelectMetricQuery).
			WithArgs(expectedMetric.ID).
			WillReturnRows(expectedRow)

		storage, err := NewPostgreSQLStorage(db)
		require.NoError(t, err)

		// Получение метрики из бд
		metric, err := storage.Get(expectedMetric.ID)
		require.NoError(t, err)
		require.NotNil(t, metric)

		metricsEqual(t, expectedMetric, *metric)

		// Проверка мок вызовов
		err = mock.ExpectationsWereMet()
		require.NoError(t, err)
	})
}

// Проверяет получение всех метрик из хранилища PostgreSQL
func TestPostgreSQLStorage_GetAll(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectPrepare(mockSelectAllMetricsQuery)

		expectedRows := sqlmock.NewRows([]string{"id", "type", "delta", "value", "hash"})
		mock.ExpectQuery(mockSelectAllMetricsQuery).
			WillReturnRows(expectedRows)

		storage, err := NewPostgreSQLStorage(db)
		require.NoError(t, err)

		// Получение метрик из бд
		metrics, err := storage.GetAll()
		require.NoError(t, err)
		require.NotNil(t, metrics)

		storagesEqual(t, NewMemStorage(), storageByMetricsArray(metrics))

		// Проверка мок вызовов
		err = mock.ExpectationsWereMet()
		require.NoError(t, err)
	})
	t.Run("get all", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectPrepare(mockSelectAllMetricsQuery)

		gaugeValue1 := -34.4441
		gaugeValue2 := 0.1
		counterDelta1 := int64(-36)
		counterDelta2 := int64(5)
		gaugeID1 := "MetricNameGauge1"
		gaugeID2 := "MetricNameGauge2"
		counterID1 := "MetricNameCounter1"
		counterID2 := "MetricNameCounter2"
		expectedS := MemStorage{
			gaugeID1: {
				ID:    gaugeID1,
				MType: model.Gauge,
				Value: &gaugeValue1,
			},
			gaugeID2: {
				ID:    gaugeID2,
				MType: model.Gauge,
				Value: &gaugeValue2,
			},
			counterID1: {
				ID:    counterID1,
				MType: model.Counter,
				Delta: &counterDelta1,
			},
			counterID2: {
				ID:    counterID2,
				MType: model.Counter,
				Delta: &counterDelta2,
			},
		}
		expectedRows := sqlmock.NewRows([]string{"id", "type", "delta", "value", "hash"}).
			AddRows([][]driver.Value{
				{expectedS[gaugeID1].ID, expectedS[gaugeID1].MType, expectedS[gaugeID1].Delta, *expectedS[gaugeID1].Value, expectedS[gaugeID1].Hash},
				{expectedS[gaugeID2].ID, expectedS[gaugeID2].MType, expectedS[gaugeID2].Delta, *expectedS[gaugeID2].Value, expectedS[gaugeID2].Hash},
				{expectedS[counterID1].ID, expectedS[counterID1].MType, *expectedS[counterID1].Delta, expectedS[counterID1].Value, expectedS[counterID1].Hash},
				{expectedS[counterID2].ID, expectedS[counterID2].MType, *expectedS[counterID2].Delta, expectedS[counterID2].Value, expectedS[counterID2].Hash},
			}...)
		mock.ExpectQuery(mockSelectAllMetricsQuery).
			WillReturnRows(expectedRows)

		// Получение метрик из бд
		storage, err := NewPostgreSQLStorage(db)
		require.NoError(t, err)

		metrics, err := storage.GetAll()
		require.NoError(t, err)

		storagesEqual(t, expectedS, storageByMetricsArray(metrics))

		// Проверка мок вызовов
		err = mock.ExpectationsWereMet()
		require.NoError(t, err)
	})
}
