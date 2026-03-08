package storage

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/skayfish/metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SF TODO
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

// SF TODO
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
		mock.ExpectExec(
			`INSERT INTO metrics_schema\.metrics \(id, "type", delta, value, hash\)
			VALUES \(\$1, \$2, \$3, \$4, \$5\)
			ON CONFLICT \(id\)
			DO UPDATE SET
				delta = metrics_schema\.metrics\.delta \+ EXCLUDED\.delta,
				value = EXCLUDED\.value,
				hash = EXCLUDED\.hash;`).
			WithArgs(
				expected.ID,
				model.Counter,
				sql.NullInt64{Valid: true, Int64: delta},
				sql.NullFloat64{Valid: false},
				"",
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		expectedRow := sqlmock.NewRows([]string{"id", "type", "delta", "value", "hash"}).
			AddRow(
				expected.ID,
				model.Counter,
				delta,
				nil,
				"")
		mock.ExpectQuery(
			`SELECT \* FROM metrics_schema\.metrics
		    WHERE id = \$1;`).
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
		mock.ExpectExec(
			`INSERT INTO metrics_schema\.metrics \(id, "type", delta, value, hash\)
			VALUES \(\$1, \$2, \$3, \$4, \$5\)
			ON CONFLICT \(id\)
			DO UPDATE SET
				delta = EXCLUDED\.delta,
				value = EXCLUDED\.value,
				hash = EXCLUDED\.hash;`).
			WithArgs(
				expected.ID,
				model.Gauge,
				sql.NullInt64{Valid: false},
				sql.NullFloat64{Valid: true, Float64: value},
				"",
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		expectedRow := sqlmock.NewRows([]string{"id", "type", "delta", "value", "hash"}).
			AddRow(
				expected.ID,
				model.Gauge,
				nil,
				value,
				"")
		mock.ExpectQuery(
			`SELECT \* FROM metrics_schema\.metrics
		    WHERE id = \$1;`).
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
}
