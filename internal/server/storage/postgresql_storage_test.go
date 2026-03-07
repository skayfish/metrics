package storage

import (
	"database/sql"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/skayfish/metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SF TODO
func TestNewPostgreSQLStorage(t *testing.T) {
	db, err := sql.Open("pgx", "")
	require.NoError(t, err)

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
				assert.Equal(t, "sql.DB is nil", err.Error())
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
		storage := NewMemStorage()
		delta := int64(5)
		expected := model.Metrics{
			ID:    "counterID",
			MType: model.Counter,
			Delta: &delta,
		}

		metric, err := storage.Update(expected)
		require.NoError(t, err)

		storagesEqual(t, MemStorage{"counterID": expected}, storage)
		metricsEqual(t, expected, *metric)

		metric, err = storage.Update(expected)
		require.NoError(t, err)

		*expected.Delta += delta
		storagesEqual(t, MemStorage{"counterID": expected}, storage)
		metricsEqual(t, expected, *metric)
	})
	//t.Run("add gauge", func(t *testing.T) {
	//	storage := NewMemStorage()
	//	value := 0.15
	//	expected := model.Metrics{
	//		ID:    "gaugeID",
	//		MType: model.Gauge,
	//		Value: &value,
	//	}

	//	metric, err := storage.Update(expected)
	//	require.NoError(t, err)

	//	storagesEqual(t, MemStorage{"gaugeID": expected}, storage)
	//	metricsEqual(t, expected, *metric)

	//	value = 4.0023
	//	metric, err = storage.Update(expected)
	//	require.NoError(t, err)

	//	*expected.Value = value
	//	storagesEqual(t, MemStorage{"gaugeID": expected}, storage)
	//	metricsEqual(t, expected, *metric)
	//})
	//t.Run("update old counter", func(t *testing.T) {
	//	delta := int64(5)
	//	expectedCounter := model.Metrics{
	//		ID:    "counterID",
	//		MType: model.Counter,
	//		Delta: &delta,
	//	}

	//	value := 0.15
	//	expectedGauge := model.Metrics{
	//		ID:    "gaugeID",
	//		MType: model.Gauge,
	//		Value: &value,
	//	}

	//	storage := NewMemStorage()
	//	storage[expectedCounter.ID] = expectedCounter
	//	storage[expectedGauge.ID] = expectedGauge

	//	metric, err := storage.Update(expectedCounter)
	//	require.NoError(t, err)

	//	*expectedCounter.Delta += delta
	//	storagesEqual(t, MemStorage{expectedGauge.ID: expectedGauge, expectedCounter.ID: expectedCounter}, storage)
	//	metricsEqual(t, expectedCounter, *metric)

	//	delta = int64(10)
	//	metric, err = storage.Update(expectedCounter)
	//	require.NoError(t, err)

	//	*expectedCounter.Delta += delta
	//	storagesEqual(t, MemStorage{expectedGauge.ID: expectedGauge, expectedCounter.ID: expectedCounter}, storage)
	//	metricsEqual(t, expectedCounter, *metric)
	//})
	//t.Run("update old gauge", func(t *testing.T) {
	//	delta := int64(5)
	//	expectedCounter := model.Metrics{
	//		ID:    "counterID",
	//		MType: model.Counter,
	//		Delta: &delta,
	//	}

	//	value := 0.15
	//	expectedGauge := model.Metrics{
	//		ID:    "gaugeID",
	//		MType: model.Gauge,
	//		Value: &value,
	//	}

	//	storage := NewMemStorage()
	//	storage[expectedCounter.ID] = expectedCounter
	//	storage[expectedGauge.ID] = expectedGauge

	//	metric, err := storage.Update(expectedGauge)
	//	require.NoError(t, err)

	//	storagesEqual(t, MemStorage{expectedGauge.ID: expectedGauge, expectedCounter.ID: expectedCounter}, storage)
	//	metricsEqual(t, expectedGauge, *metric)

	//	value = -0.43441
	//	metric, err = storage.Update(expectedGauge)
	//	require.NoError(t, err)

	//	*expectedGauge.Value = value
	//	storagesEqual(t, MemStorage{expectedGauge.ID: expectedGauge, expectedCounter.ID: expectedCounter}, storage)
	//	metricsEqual(t, expectedGauge, *metric)
	//})
}
