package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Проверяет создание нового хранилища метрик
func TestNewMemStorage(t *testing.T) {
	want := MemStorage{
		gauge:   make(map[string]float64),
		counter: make(map[string]int64),
	}
	storage := NewMemStorage()
	assert.Equal(t, want, storage)
}

// Проверяет обновление метрики датчика
func TestMemStorage_UpdateGauge(t *testing.T) {
	t.Run("empty storage", func(t *testing.T) {
		storage := NewMemStorage()
		storage.UpdateGauge("MetricName", 5.123)

		assert.Equal(t, map[string]float64{"MetricName": 5.123}, storage.gauge)
		assert.Equal(t, map[string]int64{}, storage.counter)
	})
	t.Run("empty storage", func(t *testing.T) {
		storage := NewMemStorage()
		storage.UpdateGauge("MetricName", 5)

		assert.Equal(t, map[string]float64{"MetricName": 5.0}, storage.gauge)
		assert.Equal(t, map[string]int64{}, storage.counter)
	})
	t.Run("empty storage", func(t *testing.T) {
		storage := NewMemStorage()
		storage.UpdateGauge("MetricName", 5)
		storage.UpdateGauge("MetricNameNew", -34.4441)

		expected := map[string]float64{
			"MetricName":    5.0,
			"MetricNameNew": -34.4441,
		}
		assert.Equal(t, expected, storage.gauge)
		assert.Equal(t, map[string]int64{}, storage.counter)
	})
	t.Run("empty storage", func(t *testing.T) {
		storage := NewMemStorage()
		storage.UpdateGauge("MetricName", 5.123)
		storage.UpdateGauge("MetricName", -34.4441)

		expected := map[string]float64{"MetricName": -34.4441}
		assert.Equal(t, expected, storage.gauge)
		assert.Equal(t, map[string]int64{}, storage.counter)
	})

	t.Run("not empty storage", func(t *testing.T) {
		storage := NewMemStorage()
		storage.counter = map[string]int64{"MetricName": -34}
		storage.gauge = map[string]float64{"MetricName": -34.4441}
		storage.UpdateGauge("MetricName", 5.123)

		expected := map[string]float64{"MetricName": 5.123}
		assert.Equal(t, expected, storage.gauge)
		assert.Equal(t, map[string]int64{"MetricName": -34}, storage.counter)
	})
}

// Проверяет обновление метрики счетчика
func TestMemStorage_UpdateCounter(t *testing.T) {
	t.Run("empty storage", func(t *testing.T) {
		storage := NewMemStorage()
		storage.UpdateCounter("MetricName", 53)

		assert.Equal(t, map[string]float64{}, storage.gauge)
		assert.Equal(t, map[string]int64{"MetricName": 53}, storage.counter)
	})
	t.Run("empty storage", func(t *testing.T) {
		storage := NewMemStorage()
		storage.UpdateCounter("MetricName", 5)
		storage.UpdateCounter("MetricNameNew", -34)

		expected := map[string]int64{
			"MetricName":    5,
			"MetricNameNew": -34,
		}
		assert.Equal(t, map[string]float64{}, storage.gauge)
		assert.Equal(t, expected, storage.counter)
	})
	t.Run("empty storage", func(t *testing.T) {
		storage := NewMemStorage()
		storage.UpdateCounter("MetricName", 5)
		storage.UpdateCounter("MetricName", -34)

		expected := map[string]int64{"MetricName": -29}
		assert.Equal(t, map[string]float64{}, storage.gauge)
		assert.Equal(t, expected, storage.counter)
	})

	t.Run("not empty storage", func(t *testing.T) {
		storage := MemStorage{
			counter: map[string]int64{"MetricName": -36},
			gauge:   map[string]float64{"MetricName": -34.4441},
		}
		storage.UpdateCounter("MetricName", 10)
		storage.UpdateCounter("MetricName", 10)

		expected := map[string]int64{"MetricName": -16}
		assert.Equal(t, map[string]float64{"MetricName": -34.4441}, storage.gauge)
		assert.Equal(t, expected, storage.counter)
	})
}

// Проверяет получение значения метрики датчика из хранилища
func TestMemStorage_GetGauge(t *testing.T) {
	t.Run("empty storage", func(t *testing.T) {
		storage := NewMemStorage()
		_, ok := storage.GetGauge("MetricName")

		assert.Equal(t, false, ok)
	})

	t.Run("found", func(t *testing.T) {
		storage := MemStorage{
			counter: map[string]int64{"MetricName": -36},
			gauge:   map[string]float64{"MetricName": -34.4441},
		}
		value, ok := storage.GetGauge("MetricName")

		assert.Equal(t, true, ok)
		assert.Equal(t, -34.4441, value)
	})

	t.Run("not found", func(t *testing.T) {
		storage := MemStorage{
			counter: map[string]int64{"MetricName": -36},
			gauge:   map[string]float64{"MetricName": -34.4441},
		}
		_, ok := storage.GetGauge("UnknownMetricName")

		assert.Equal(t, false, ok)
	})
}

// Проверяет получение значения метрики счетчика из хранилища
func TestMemStorage_GetCounter(t *testing.T) {
	t.Run("empty storage", func(t *testing.T) {
		storage := NewMemStorage()
		_, ok := storage.GetCounter("MetricName")

		assert.Equal(t, false, ok)
	})

	t.Run("found", func(t *testing.T) {
		storage := MemStorage{
			counter: map[string]int64{"MetricName": -36},
			gauge:   map[string]float64{"MetricName": -34.4441},
		}
		value, ok := storage.GetCounter("MetricName")

		assert.Equal(t, true, ok)
		assert.Equal(t, int64(-36), value)
	})

	t.Run("not found", func(t *testing.T) {
		storage := MemStorage{
			counter: map[string]int64{"MetricName": -36},
			gauge:   map[string]float64{"MetricName": -34.4441},
		}
		_, ok := storage.GetGauge("UnknownMetricName")

		assert.Equal(t, false, ok)
	})
}

// Проверяет получение значений метрик датчиков из хранилища
func TestMemStorage_GetGauges(t *testing.T) {
	t.Run("empty storage", func(t *testing.T) {
		storage := NewMemStorage()
		gauges := storage.GetGauges()

		assert.Equal(t, map[string]float64{}, gauges)
	})

	t.Run("get all", func(t *testing.T) {
		storage := MemStorage{
			counter: map[string]int64{"MetricName": -36},
			gauge:   map[string]float64{"MetricName": -34.4441, "MetricName1": 0.1},
		}
		gauges := storage.GetGauges()

		assert.Equal(t, storage.gauge, gauges)
	})
}

// Проверяет получение значений метрик счетчиков из хранилища
func TestMemStorage_GetCounters(t *testing.T) {
	t.Run("empty storage", func(t *testing.T) {
		storage := NewMemStorage()
		counters := storage.GetCounters()

		assert.Equal(t, map[string]int64{}, counters)
	})

	t.Run("get all", func(t *testing.T) {
		storage := MemStorage{
			counter: map[string]int64{"MetricName": -36, "MetricName1": 9999},
			gauge:   map[string]float64{"MetricName": -34.4441, "MetricName1": 0.1},
		}
		counters := storage.GetCounters()

		assert.Equal(t, storage.counter, counters)
	})
}
