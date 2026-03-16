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

// Формирует хранилище по метрикам
func storageByMetricsArray(metrics []model.Metrics) *MemStorage {
	storage := NewMemStorage()
	for _, metric := range metrics {
		storage.metrics[metric.ID] = metric
	}

	return &storage
}

// Проверяет создание нового хранилища метрик
func TestNewMemStorage(t *testing.T) {
	want := make(map[string]model.Metrics, 0)
	storage := NewMemStorage()
	assert.Equal(t, want, storage.metrics)
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
func storagesEqual(t *testing.T, expected, storage *MemStorage) {
	if expected == nil || storage == nil {
		require.Equal(t, expected, storage)
		return
	}

	if len(expected.metrics) != len(storage.metrics) {
		t.Errorf("Mismatched storage sizes: len(expected)=%d vs len(RHS)=%d", len(expected.metrics), len(storage.metrics))
	}

	for id, metric := range expected.metrics {
		metricRHS, found := storage.metrics[id]
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
		storage.metrics["ID"] = model.Metrics{
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
		storage.metrics["ID"] = model.Metrics{
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

		expectedStorage := NewMemStorage()
		expectedStorage.metrics[expected.ID] = expected
		storagesEqual(t, &expectedStorage, &storage)
		metricsEqual(t, expected, *metric)

		metric, err = storage.Update(expected)
		require.NoError(t, err)

		*expected.Delta += delta
		expectedStorage = NewMemStorage()
		expectedStorage.metrics[expected.ID] = expected
		storagesEqual(t, &expectedStorage, &storage)
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

		expectedStorage := NewMemStorage()
		expectedStorage.metrics[expected.ID] = expected
		storagesEqual(t, &expectedStorage, &storage)
		metricsEqual(t, expected, *metric)

		value = 4.0023
		metric, err = storage.Update(expected)
		require.NoError(t, err)

		*expected.Value = value
		expectedStorage = NewMemStorage()
		expectedStorage.metrics[expected.ID] = expected
		storagesEqual(t, &expectedStorage, &storage)
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
		storage.metrics[expectedCounter.ID] = expectedCounter
		storage.metrics[expectedGauge.ID] = expectedGauge

		metric, err := storage.Update(expectedCounter)
		require.NoError(t, err)

		*expectedCounter.Delta += delta
		expectedStorage := NewMemStorage()
		expectedStorage.metrics[expectedGauge.ID] = expectedGauge
		expectedStorage.metrics[expectedCounter.ID] = expectedCounter
		storagesEqual(t, &expectedStorage, &storage)
		metricsEqual(t, expectedCounter, *metric)

		delta = int64(10)
		metric, err = storage.Update(expectedCounter)
		require.NoError(t, err)

		*expectedCounter.Delta += delta
		expectedStorage = NewMemStorage()
		expectedStorage.metrics[expectedGauge.ID] = expectedGauge
		expectedStorage.metrics[expectedCounter.ID] = expectedCounter
		storagesEqual(t, &expectedStorage, &storage)
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
		storage.metrics[expectedCounter.ID] = expectedCounter
		storage.metrics[expectedGauge.ID] = expectedGauge

		metric, err := storage.Update(expectedGauge)
		require.NoError(t, err)

		expectedStorage := NewMemStorage()
		expectedStorage.metrics[expectedGauge.ID] = expectedGauge
		expectedStorage.metrics[expectedCounter.ID] = expectedCounter
		storagesEqual(t, &expectedStorage, &storage)
		metricsEqual(t, expectedGauge, *metric)

		value = -0.43441
		metric, err = storage.Update(expectedGauge)
		require.NoError(t, err)

		*expectedGauge.Value = value
		expectedStorage = NewMemStorage()
		expectedStorage.metrics[expectedGauge.ID] = expectedGauge
		expectedStorage.metrics[expectedCounter.ID] = expectedCounter
		storagesEqual(t, &expectedStorage, &storage)
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
		storage := NewMemStorage()
		storage.metrics["MetricNameGauge"] = model.Metrics{
			ID:    "MetricNameGauge",
			MType: model.Gauge,
			Value: &gaugeValue,
		}
		storage.metrics["MetricNameCounter"] = model.Metrics{
			ID:    "MetricNameCounter",
			MType: model.Counter,
			Delta: &counterValue,
		}

		_, err := storage.Get("UnknownMetricName")

		require.ErrorIs(t, err, ErrMetricNotFound)
	})
	t.Run("found gauge", func(t *testing.T) {
		gaugeValue := -34.4441
		counterValue := int64(-36)
		storage := NewMemStorage()
		storage.metrics["MetricNameGauge"] = model.Metrics{
			ID:    "MetricNameGauge",
			MType: model.Gauge,
			Value: &gaugeValue,
		}
		storage.metrics["MetricNameCounter"] = model.Metrics{
			ID:    "MetricNameCounter",
			MType: model.Counter,
			Delta: &counterValue,
		}

		metric, err := storage.Get("MetricNameGauge")
		require.NoError(t, err)
		metricsEqual(t, storage.metrics["MetricNameGauge"], *metric)
	})
	t.Run("found counter", func(t *testing.T) {
		gaugeValue := -34.4441
		counterValue := int64(-36)
		storage := NewMemStorage()
		storage.metrics["MetricNameGauge"] = model.Metrics{
			ID:    "MetricNameGauge",
			MType: model.Gauge,
			Value: &gaugeValue,
		}
		storage.metrics["MetricNameCounter"] = model.Metrics{
			ID:    "MetricNameCounter",
			MType: model.Counter,
			Delta: &counterValue,
		}
		metric, err := storage.Get("MetricNameCounter")
		require.NoError(t, err)
		metricsEqual(t, storage.metrics["MetricNameCounter"], *metric)
	})
}

// Проверяет получение значений всех метрик из хранилища
func TestMemStorage_GetAll(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		storage := NewMemStorage()
		metrics, err := storage.GetAll()
		require.NoError(t, err)

		storagesEqual(t, &storage, storageByMetricsArray(metrics))
	})
	t.Run("get all", func(t *testing.T) {
		oldGaugeValue := -34.4441
		oldGaugeValue1 := 0.1
		oldCounterValue := int64(-36)
		storage := NewMemStorage()
		storage.metrics["MetricNameGauge"] = model.Metrics{
			ID:    "MetricNameGauge",
			MType: model.Gauge,
			Value: &oldGaugeValue,
		}
		storage.metrics["MetricNameGauge1"] = model.Metrics{
			ID:    "MetricNameGauge1",
			MType: model.Gauge,
			Value: &oldGaugeValue1,
		}
		storage.metrics["MetricNameCounter"] = model.Metrics{
			ID:    "MetricNameCounter",
			MType: model.Counter,
			Delta: &oldCounterValue,
		}

		metrics, err := storage.GetAll()
		require.NoError(t, err)

		storagesEqual(t, &storage, storageByMetricsArray(metrics))
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

		expected := NewMemStorage()
		storagesEqual(t, &expected, storageByMetricsArray(metrics))
	})
	t.Run("get all", func(t *testing.T) {
		storage := NewMemStorage()
		storage.metrics["MetricNameGauge"] = model.Metrics{
			ID:    "MetricNameGauge",
			MType: model.Gauge,
			Value: &oldGaugeValue,
		}
		storage.metrics["MetricNameGauge1"] = model.Metrics{
			ID:    "MetricNameGauge1",
			MType: model.Gauge,
			Value: &oldGaugeValue1,
		}
		storage.metrics["MetricNameCounter"] = model.Metrics{
			ID:    "MetricNameCounter",
			MType: model.Counter,
			Delta: &oldCounterValue,
		}

		metrics, err := storage.GetAllContext(context.Background())
		require.NoError(t, err)

		metricsMap := make(map[string]model.Metrics, 0)
		for _, metric := range metrics {
			metricsMap[metric.ID] = metric
		}

		actualMetricsStorage := NewMemStorage()
		actualMetricsStorage.metrics = metricsMap
		storagesEqual(t, &storage, &actualMetricsStorage)
	})
	t.Run("canceled context", func(t *testing.T) {
		storage := NewMemStorage()
		storage.metrics["MetricNameGauge"] = model.Metrics{
			ID:    "MetricNameGauge",
			MType: model.Gauge,
			Value: &oldGaugeValue,
		}
		storage.metrics["MetricNameGauge1"] = model.Metrics{
			ID:    "MetricNameGauge1",
			MType: model.Gauge,
			Value: &oldGaugeValue1,
		}
		storage.metrics["MetricNameCounter"] = model.Metrics{
			ID:    "MetricNameCounter",
			MType: model.Counter,
			Delta: &oldCounterValue,
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		metrics, err := storage.GetAllContext(ctx)
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)

		expected := NewMemStorage()
		storagesEqual(t, &expected, storageByMetricsArray(metrics))
	})
}

// Возвращает непустое хранилище с валидными данными
func newSuccessMemStorage() *MemStorage {
	var delta1 int64 = 1298476200
	var value1 float64 = 131072
	var value2 float64 = 15288

	storage := NewMemStorage()
	storage.metrics["StackInuse"] = model.Metrics{
		ID:    "StackInuse",
		MType: model.Gauge,
		Value: &value1,
	}
	storage.metrics["MCacheSys"] = model.Metrics{
		ID:    "MCacheSys",
		MType: model.Gauge,
		Value: &value2,
	}
	storage.metrics["GetSet92"] = model.Metrics{
		ID:    "GetSet92",
		MType: model.Counter,
		Delta: &delta1,
	}

	return &storage
}

// Проверяет сохранение в файл данных хранилища метрик
func TestMemStorage_SaveStorageToFile(t *testing.T) {
	emptyStorage := NewMemStorage()
	notEmptyStorage := newSuccessMemStorage()
	tests := []struct {
		test      string
		storage   *MemStorage
		wantErr   bool
		errPrefix string
	}{
		{
			test:    "empty storage",
			storage: &emptyStorage,
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

// Проверяет закрытие хранилище в памяти приложения
func TestMemStorage_Close(t *testing.T) {
	emptyStorage := NewMemStorage()
	notEmptyStorage := newSuccessMemStorage()
	tests := []struct {
		test    string
		storage *MemStorage
	}{
		{
			test:    "empty storage",
			storage: &emptyStorage,
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
