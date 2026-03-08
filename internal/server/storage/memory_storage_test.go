package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/skayfish/metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SF TODO
func storageByMetricsArray(metrics []model.Metrics) MemStorage {
	storage := make(MemStorage)
	for _, metric := range metrics {
		storage[metric.ID] = metric
	}

	return storage
}

// Проверяет создание нового хранилища метрик
func TestNewMemStorage(t *testing.T) {
	want := make(MemStorage, 0)
	storage := NewMemStorage()
	assert.Equal(t, want, storage)
}

// Сравнивает два вещественных числа на равенство
//
//	@param t        экземпляр теста
//	@param expected ожидаемое значение
//	@param num      реальное значение
func float64Equal(t *testing.T, expected, num float64) {
	assert.Truef(t, math.Abs(expected-num) <= 1e-10, "expected(%f) != num(%f)", expected, num)
}

func metricsEqual(t *testing.T, expected, target model.Metrics) {
	require.Equal(t, reflect.TypeFor[model.Metrics]().NumField(), 5)

	assert.Equal(t, expected.ID, target.ID)
	assert.Equal(t, expected.Hash, target.Hash)
	require.Equal(t, expected.MType, target.MType)
	switch expected.MType {
	case model.Gauge:
		require.NotNil(t, expected.Value)
		require.NotNil(t, target.Value)
		float64Equal(t, *expected.Value, *target.Value)
	case model.Counter:
		require.NotNil(t, expected.Delta)
		require.NotNil(t, target.Delta)
		assert.Equal(t, *expected.Delta, *target.Delta)
	default:
		t.Error("Unknown target metric type")
	}
}

// Сравнивает в глубину два хранилища метрик
//
//	@param t        экземпляр теста
//	@param expected ожидаемое значение
//	@param storage  реальное значение
func storagesEqual(t *testing.T, expected, storage MemStorage) {
	if len(expected) != len(storage) {
		t.Errorf("Mismatched storage sizes: len(expected)=%d vs len(RHS)=%d", len(expected), len(storage))
	}

	for id, metric := range expected {
		metricRHS, found := storage[id]
		if !found {
			t.Errorf("RHS storage does not contain ID: %s", id)
		}

		metricsEqual(t, metric, metricRHS)
	}
}

// Проверяет обновление/добавление метрик в хранилище
func TestMemStorage_Update(t *testing.T) {
	t.Run("found not gauge metric type", func(t *testing.T) {
		storage := NewMemStorage()
		delta := int64(5)
		storage["ID"] = model.Metrics{
			ID:    "ID",
			MType: model.Counter,
			Delta: &delta,
		}

		value := 0.15
		metric, err := storage.Update(model.Metrics{
			ID:    "ID",
			MType: model.Gauge,
			Value: &value,
		})
		require.ErrorIs(t, err, ErrFoundNotGaugeMetricType)
		assert.Nil(t, metric)
	})
	t.Run("found not counter metric type", func(t *testing.T) {
		storage := NewMemStorage()
		value := 0.15
		storage["ID"] = model.Metrics{
			ID:    "ID",
			MType: model.Gauge,
			Value: &value,
		}

		delta := int64(5)
		metric, err := storage.Update(model.Metrics{
			ID:    "ID",
			MType: model.Counter,
			Delta: &delta,
		})
		require.ErrorIs(t, err, ErrFoundNotCounterMetricType)
		assert.Nil(t, metric)
	})
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
	t.Run("add gauge", func(t *testing.T) {
		storage := NewMemStorage()
		value := 0.15
		expected := model.Metrics{
			ID:    "gaugeID",
			MType: model.Gauge,
			Value: &value,
		}

		metric, err := storage.Update(expected)
		require.NoError(t, err)

		storagesEqual(t, MemStorage{"gaugeID": expected}, storage)
		metricsEqual(t, expected, *metric)

		value = 4.0023
		metric, err = storage.Update(expected)
		require.NoError(t, err)

		*expected.Value = value
		storagesEqual(t, MemStorage{"gaugeID": expected}, storage)
		metricsEqual(t, expected, *metric)
	})
	t.Run("update old counter", func(t *testing.T) {
		delta := int64(5)
		expectedCounter := model.Metrics{
			ID:    "counterID",
			MType: model.Counter,
			Delta: &delta,
		}

		value := 0.15
		expectedGauge := model.Metrics{
			ID:    "gaugeID",
			MType: model.Gauge,
			Value: &value,
		}

		storage := NewMemStorage()
		storage[expectedCounter.ID] = expectedCounter
		storage[expectedGauge.ID] = expectedGauge

		metric, err := storage.Update(expectedCounter)
		require.NoError(t, err)

		*expectedCounter.Delta += delta
		storagesEqual(t, MemStorage{expectedGauge.ID: expectedGauge, expectedCounter.ID: expectedCounter}, storage)
		metricsEqual(t, expectedCounter, *metric)

		delta = int64(10)
		metric, err = storage.Update(expectedCounter)
		require.NoError(t, err)

		*expectedCounter.Delta += delta
		storagesEqual(t, MemStorage{expectedGauge.ID: expectedGauge, expectedCounter.ID: expectedCounter}, storage)
		metricsEqual(t, expectedCounter, *metric)
	})
	t.Run("update old gauge", func(t *testing.T) {
		delta := int64(5)
		expectedCounter := model.Metrics{
			ID:    "counterID",
			MType: model.Counter,
			Delta: &delta,
		}

		value := 0.15
		expectedGauge := model.Metrics{
			ID:    "gaugeID",
			MType: model.Gauge,
			Value: &value,
		}

		storage := NewMemStorage()
		storage[expectedCounter.ID] = expectedCounter
		storage[expectedGauge.ID] = expectedGauge

		metric, err := storage.Update(expectedGauge)
		require.NoError(t, err)

		storagesEqual(t, MemStorage{expectedGauge.ID: expectedGauge, expectedCounter.ID: expectedCounter}, storage)
		metricsEqual(t, expectedGauge, *metric)

		value = -0.43441
		metric, err = storage.Update(expectedGauge)
		require.NoError(t, err)

		*expectedGauge.Value = value
		storagesEqual(t, MemStorage{expectedGauge.ID: expectedGauge, expectedCounter.ID: expectedCounter}, storage)
		metricsEqual(t, expectedGauge, *metric)
	})
}

// Проверяет получение значения метрики из хранилища
func TestMemStorage_Get(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		storage := NewMemStorage()
		_, err := storage.Get("MetricName")

		require.ErrorIs(t, err, ErrMetricNotFound)
	})
	t.Run("not found", func(t *testing.T) {
		gaugeValue := -34.4441
		counterValue := int64(-36)
		storage := MemStorage{
			"MetricNameGauge": {
				ID:    "MetricNameGauge",
				MType: model.Gauge,
				Value: &gaugeValue,
			},
			"MetricNameCounter": {
				ID:    "MetricNameCounter",
				MType: model.Counter,
				Delta: &counterValue,
			},
		}
		_, err := storage.Get("UnknownMetricName")

		require.ErrorIs(t, err, ErrMetricNotFound)
	})
	t.Run("found gauge", func(t *testing.T) {
		gaugeValue := -34.4441
		counterValue := int64(-36)
		storage := MemStorage{
			"MetricNameGauge": {
				ID:    "MetricNameGauge",
				MType: model.Gauge,
				Value: &gaugeValue,
			},
			"MetricNameCounter": {
				ID:    "MetricNameCounter",
				MType: model.Counter,
				Delta: &counterValue,
			},
		}

		metric, err := storage.Get("MetricNameGauge")
		require.NoError(t, err)
		metricsEqual(t, storage["MetricNameGauge"], *metric)
	})
	t.Run("found counter", func(t *testing.T) {
		gaugeValue := -34.4441
		counterValue := int64(-36)
		storage := MemStorage{
			"MetricNameGauge": {
				ID:    "MetricNameGauge",
				MType: model.Gauge,
				Value: &gaugeValue,
			},
			"MetricNameCounter": {
				ID:    "MetricNameCounter",
				MType: model.Counter,
				Delta: &counterValue,
			},
		}
		metric, err := storage.Get("MetricNameCounter")
		require.NoError(t, err)
		metricsEqual(t, storage["MetricNameCounter"], *metric)
	})
}

// Проверяет получение значений всех метрик из хранилища
func TestMemStorage_GetAll(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		storage := NewMemStorage()
		metrics, err := storage.GetAll()
		require.NoError(t, err)

		storagesEqual(t, NewMemStorage(), storageByMetricsArray(metrics))
	})
	t.Run("get all", func(t *testing.T) {
		oldGaugeValue := -34.4441
		oldGaugeValue1 := 0.1
		oldCounterValue := int64(-36)
		storage := MemStorage{
			"MetricNameGauge": {
				ID:    "MetricNameGauge",
				MType: model.Gauge,
				Value: &oldGaugeValue,
			},
			"MetricNameGauge1": {
				ID:    "MetricNameGauge1",
				MType: model.Gauge,
				Value: &oldGaugeValue1,
			},
			"MetricNameCounter": {
				ID:    "MetricNameCounter",
				MType: model.Counter,
				Delta: &oldCounterValue,
			},
		}

		metrics, err := storage.GetAll()
		require.NoError(t, err)

		storagesEqual(t, storage, storageByMetricsArray(metrics))
	})
}

// Проверяет получение значений всех метрик из хранилища
func TestMemStorage_GetAllContext(t *testing.T) {
	oldGaugeValue := -34.4441
	oldGaugeValue1 := 0.1
	oldCounterValue := int64(-36)

	t.Run("empty", func(t *testing.T) {
		storage := NewMemStorage()
		metrics, err := storage.GetAllContext(context.Background())
		require.NoError(t, err)

		storagesEqual(t, NewMemStorage(), storageByMetricsArray(metrics))
	})
	t.Run("get all", func(t *testing.T) {
		storage := MemStorage{
			"MetricNameGauge": {
				ID:    "MetricNameGauge",
				MType: model.Gauge,
				Value: &oldGaugeValue,
			},
			"MetricNameGauge1": {
				ID:    "MetricNameGauge1",
				MType: model.Gauge,
				Value: &oldGaugeValue1,
			},
			"MetricNameCounter": {
				ID:    "MetricNameCounter",
				MType: model.Counter,
				Delta: &oldCounterValue,
			},
		}

		metrics, err := storage.GetAllContext(context.Background())
		require.NoError(t, err)

		metricsMap := make(map[string]model.Metrics, 0)
		for _, metric := range metrics {
			metricsMap[metric.ID] = metric
		}

		storagesEqual(t, storage, MemStorage(metricsMap))
	})
	t.Run("canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		storage := MemStorage{
			"MetricNameGauge": {
				ID:    "MetricNameGauge",
				MType: model.Gauge,
				Value: &oldGaugeValue,
			},
			"MetricNameGauge1": {
				ID:    "MetricNameGauge1",
				MType: model.Gauge,
				Value: &oldGaugeValue1,
			},
			"MetricNameCounter": {
				ID:    "MetricNameCounter",
				MType: model.Counter,
				Delta: &oldCounterValue,
			},
		}
		metrics, err := storage.GetAllContext(ctx)
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)

		storagesEqual(t, NewMemStorage(), storageByMetricsArray(metrics))
	})
}

// Возвращает непустое хранилище с валидными данными
func newSuccessMemStorage() MemStorage {
	var delta1 int64 = 1298476200
	var value1 float64 = 131072
	var value2 float64 = 15288

	return MemStorage{
		"GetSet92": model.Metrics{
			ID:    "GetSet92",
			MType: model.Counter,
			Delta: &delta1,
		},
		"StackInuse": model.Metrics{
			ID:    "StackInuse",
			MType: model.Gauge,
			Value: &value1,
		},
		"MCacheSys": model.Metrics{
			ID:    "MCacheSys",
			MType: model.Gauge,
			Value: &value2,
		},
	}
}

// Проверяет сохранение в файл данных хранилища метрик
func TestMemStorage_SaveStorageToFile(t *testing.T) {
	emptyStorage := NewMemStorage()
	notEmptyStorage := newSuccessMemStorage()
	tests := []struct {
		test      string
		storage   MemStorage
		wantErr   bool
		errPrefix string
	}{
		{
			test:    "empty storage",
			storage: emptyStorage,
		},
		{
			test:    "not empty storage",
			storage: notEmptyStorage,
		},
	}
	for i := range tests {
		tt := &tests[i]
		t.Run(tt.test, func(t *testing.T) {
			file, err := os.CreateTemp(os.TempDir(), "storage*.json")
			require.NoError(t, err)
			defer os.Remove(file.Name())

			expectedStorage := tt.storage
			err = tt.storage.SaveStorageToFile(file.Name())
			require.NoError(t, err)

			data, err := os.ReadFile(file.Name())
			require.NoError(t, err)

			var metrics []model.Metrics
			err = json.Unmarshal(data, &metrics)
			require.NoError(t, err)

			storagesEqual(t, expectedStorage, storageByMetricsArray(metrics))
		})
	}

	t.Run("failed write to file", func(t *testing.T) {
		filePath := "./unknown directory/unknown.json"
		err := notEmptyStorage.SaveStorageToFile(filePath)
		require.Error(t, err)
		hasPrefix := strings.HasPrefix(err.Error(), fmt.Sprintf("failed write to file %q:", filePath))
		require.True(t, hasPrefix)
	})
}

// SF TODO
func TestMemStorage_Close(t *testing.T) {
	emptyStorage := NewMemStorage()
	notEmptyStorage := newSuccessMemStorage()
	tests := []struct {
		test    string
		storage MemStorage
	}{
		{
			test:    "empty storage",
			storage: emptyStorage,
		},
		{
			test:    "not empty storage",
			storage: notEmptyStorage,
		},
	}
	for _, tt := range tests {
		t.Run(tt.test, func(t *testing.T) {
			expectedStorage := tt.storage
			err := tt.storage.Close()
			require.NoError(t, err)
			storagesEqual(t, expectedStorage, tt.storage)
		})
	}
}
