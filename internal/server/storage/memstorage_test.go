package storage

import (
	"math"
	"reflect"
	"testing"

	"github.com/skayfish/metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func metricsEqual(t *testing.T, expected, metrics model.Metrics) {
	require.Equal(t, reflect.TypeFor[model.Metrics]().NumField(), 5)

	assert.Equal(t, expected.ID, metrics.ID)
	assert.Equal(t, expected.Hash, metrics.Hash)
	require.Equal(t, expected.MType, metrics.MType)
	switch expected.MType {
	case model.Gauge:
		require.NotNil(t, expected.Value)
		require.NotNil(t, metrics.Value)
		float64Equal(t, *expected.Value, *metrics.Value)
	case model.Counter:
		require.NotNil(t, expected.Delta)
		require.NotNil(t, metrics.Delta)
		assert.Equal(t, *expected.Delta, *metrics.Delta)
	default:
		t.Error("Unknown metric type")
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
	t.Run("value is empty", func(t *testing.T) {
		storage := NewMemStorage()
		metric, err := storage.Update(model.Metrics{
			ID:    "ID",
			MType: model.Gauge,
		})
		require.ErrorIs(t, err, model.ErrValueIsEmpty)
		assert.Nil(t, metric)
	})
	t.Run("delta is empty", func(t *testing.T) {
		storage := NewMemStorage()
		metric, err := storage.Update(model.Metrics{
			ID:    "ID",
			MType: model.Counter,
		})
		require.ErrorIs(t, err, model.ErrDeltaIsEmpty)
		assert.Nil(t, metric)
	})
	t.Run("unrecognized metric type", func(t *testing.T) {
		storage := NewMemStorage()
		metric, err := storage.Update(model.Metrics{
			ID:    "ID",
			MType: "invalid",
		})
		require.ErrorIs(t, err, model.ErrUnrecognizedMetricType)
		assert.Nil(t, metric)
	})
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

// Проверяет обновление/добавление метрики датчика
func TestMemStorage_UpdateGauge(t *testing.T) {
	t.Run("add new", func(t *testing.T) {
		storage := NewMemStorage()
		err := storage.UpdateGauge("MetricName", 5.123)
		require.NoError(t, err)

		value := 5.123
		storagesEqual(t, MemStorage{
			"MetricName": {
				ID:    "MetricName",
				MType: model.Gauge,
				Value: &value,
			},
		}, storage)
	})
	t.Run("add new", func(t *testing.T) {
		storage := NewMemStorage()
		storage.UpdateGauge("MetricName", 5)

		value := float64(5)
		storagesEqual(t, MemStorage{
			"MetricName": {
				ID:    "MetricName",
				MType: model.Gauge,
				Value: &value,
			},
		}, storage)
	})
	t.Run("add two metrics", func(t *testing.T) {
		storage := NewMemStorage()
		storage.UpdateGauge("MetricName", 5)
		storage.UpdateGauge("MetricNameNew", -34.4441)

		value1 := 5.0
		value2 := -34.4441
		expected := MemStorage{
			"MetricName": {
				ID:    "MetricName",
				MType: model.Gauge,
				Value: &value1,
			},
			"MetricNameNew": {
				ID:    "MetricNameNew",
				MType: model.Gauge,
				Value: &value2,
			},
		}

		storagesEqual(t, expected, storage)
	})
	t.Run("add and update", func(t *testing.T) {
		storage := NewMemStorage()
		storage.UpdateGauge("MetricName", 5.123)
		storage.UpdateGauge("MetricName", -34.4441)

		value := -34.4441
		expected := MemStorage{
			"MetricName": {
				ID:    "MetricName",
				MType: model.Gauge,
				Value: &value,
			},
		}

		storagesEqual(t, expected, storage)
	})

	t.Run("update old", func(t *testing.T) {
		oldGaugeValue := -34.4441
		oldCounterValue := int64(-15)
		storage := MemStorage{
			"MetricNameGauge": {
				ID:    "MetricNameGauge",
				MType: model.Gauge,
				Value: &oldGaugeValue,
			},
			"MetricNameCounter": {
				ID:    "MetricNameCounter",
				MType: model.Counter,
				Delta: &oldCounterValue,
			},
		}

		value := 5.123
		(&storage).UpdateGauge("MetricNameGauge", value)

		expected := MemStorage{
			"MetricNameGauge": {
				ID:    "MetricNameGauge",
				MType: model.Gauge,
				Value: &value,
			},
			"MetricNameCounter": {
				ID:    "MetricNameCounter",
				MType: model.Counter,
				Delta: &oldCounterValue,
			},
		}

		storagesEqual(t, expected, storage)
	})
}

// Проверяет обновление/добавление метрики счетчика
func TestMemStorage_UpdateCounter(t *testing.T) {
	t.Run("add new", func(t *testing.T) {
		storage := NewMemStorage()
		storage.UpdateCounter("MetricName", 53)

		value := int64(53)
		expected := MemStorage{
			"MetricName": {
				ID:    "MetricName",
				MType: model.Counter,
				Delta: &value,
			},
		}

		storagesEqual(t, expected, storage)
	})
	t.Run("add two metrics", func(t *testing.T) {
		storage := NewMemStorage()
		storage.UpdateCounter("MetricName1", 5)
		storage.UpdateCounter("MetricName2", -34)

		value1 := int64(5)
		value2 := int64(-34)
		expected := MemStorage{
			"MetricName1": {
				ID:    "MetricName1",
				MType: model.Counter,
				Delta: &value1,
			},
			"MetricName2": {
				ID:    "MetricName2",
				MType: model.Counter,
				Delta: &value2,
			},
		}

		storagesEqual(t, expected, storage)
	})
	t.Run("add and update", func(t *testing.T) {
		storage := NewMemStorage()
		storage.UpdateCounter("MetricName", 5)
		storage.UpdateCounter("MetricName", -34)

		value := int64(-29)
		expected := MemStorage{
			"MetricName": {
				ID:    "MetricName",
				MType: model.Counter,
				Delta: &value,
			},
		}

		storagesEqual(t, expected, storage)
	})

	t.Run("not empty storage", func(t *testing.T) {
		oldGaugeValue := -34.4441
		oldCounterValue := int64(-36)
		storage := MemStorage{
			"MetricNameGauge": {
				ID:    "MetricNameGauge",
				MType: model.Gauge,
				Value: &oldGaugeValue,
			},
			"MetricNameCounter": {
				ID:    "MetricNameCounter",
				MType: model.Counter,
				Delta: &oldCounterValue,
			},
		}
		storage.UpdateCounter("MetricNameCounter", 10)
		storage.UpdateCounter("MetricNameCounter", 10)

		value := int64(-16)
		expected := MemStorage{
			"MetricNameGauge": {
				ID:    "MetricNameGauge",
				MType: model.Gauge,
				Value: &oldGaugeValue,
			},
			"MetricNameCounter": {
				ID:    "MetricNameCounter",
				MType: model.Counter,
				Delta: &value,
			},
		}

		storagesEqual(t, expected, storage)
	})
}

// Проверяет получение значения метрики датчика из хранилища
func TestMemStorage_GetGauge(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		storage := NewMemStorage()
		_, err := storage.GetGauge("MetricName")

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
		_, err := storage.GetGauge("UnknownMetricName")

		require.ErrorIs(t, err, ErrMetricNotFound)
	})
	t.Run("found", func(t *testing.T) {
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

		value, err := storage.GetGauge("MetricNameGauge")

		assert.NoError(t, err)
		float64Equal(t, -34.4441, value)
	})
	t.Run("incorrect type", func(t *testing.T) {
		counterValue := int64(-36)
		storage := MemStorage{
			"MetricNameCounter": {
				ID:    "MetricNameCounter",
				MType: model.Counter,
				Delta: &counterValue,
			},
		}

		// Поиск Gauge метрики с id = MetricNameCounter
		_, err := storage.GetGauge("MetricNameCounter")

		require.ErrorIs(t, err, ErrFoundNotGaugeMetricType)
	})
}

// Проверяет получение значения метрики счетчика из хранилища
func TestMemStorage_GetCounter(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		storage := NewMemStorage()
		_, err := storage.GetCounter("MetricName")

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
		_, err := storage.GetGauge("UnknownMetricName")

		require.ErrorIs(t, err, ErrMetricNotFound)
	})
	t.Run("found", func(t *testing.T) {
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
		value, err := storage.GetCounter("MetricNameCounter")

		assert.NoError(t, err)
		assert.Equal(t, int64(-36), value)
	})
	t.Run("incorrect type", func(t *testing.T) {
		gaugeValue := -34.4441
		storage := MemStorage{
			"MetricNameGauge": {
				ID:    "MetricNameGauge",
				MType: model.Gauge,
				Value: &gaugeValue,
			},
		}

		// Поиск counter метрики с id = MetricNameGauge
		_, err := storage.GetCounter("MetricNameGauge")

		require.ErrorIs(t, err, ErrFoundNotCounterMetricType)
	})
}

// Проверяет получение значений всех метрик из хранилища
func TestMemStorage_GetAll(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		storage := NewMemStorage()
		metrics, err := storage.GetAll()
		require.NoError(t, err)

		assert.Equal(t, []model.Metrics{}, metrics)
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

		metricsMap := make(map[string]model.Metrics, 0)
		for _, metric := range metrics {
			metricsMap[metric.ID] = metric
		}

		storagesEqual(t, storage, MemStorage(metricsMap))
	})
}
