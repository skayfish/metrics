package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMemStorage(t *testing.T) {
	want := MemStorage{
		gauge:   make(map[string]float64),
		counter: make(map[string]int64),
	}
	storage := NewMemStorage()
	assert.Equal(t, want, storage)
}

func TestMemStorage_UpdateGauge(t *testing.T) {
	t.Run("Empty storage", func(t *testing.T) {
		storage := NewMemStorage()
		storage.UpdateGauge("MetricName", 5.123)

		assert.Equal(t, map[string]float64{"MetricName": 5.123}, storage.gauge)
		assert.Equal(t, map[string]int64{}, storage.counter)
	})
	t.Run("Empty storage", func(t *testing.T) {
		storage := NewMemStorage()
		storage.UpdateGauge("MetricName", 5)

		assert.Equal(t, map[string]float64{"MetricName": 5.0}, storage.gauge)
		assert.Equal(t, map[string]int64{}, storage.counter)
	})
	t.Run("Empty storage", func(t *testing.T) {
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
	t.Run("Empty storage", func(t *testing.T) {
		storage := NewMemStorage()
		storage.UpdateGauge("MetricName", 5.123)
		storage.UpdateGauge("MetricName", -34.4441)

		expected := map[string]float64{"MetricName": -34.4441}
		assert.Equal(t, expected, storage.gauge)
		assert.Equal(t, map[string]int64{}, storage.counter)
	})

	t.Run("Not empty storage", func(t *testing.T) {
		storage := NewMemStorage()
		storage.counter = map[string]int64{"MetricName": -34}
		storage.gauge = map[string]float64{"MetricName": -34.4441}
		storage.UpdateGauge("MetricName", 5.123)

		expected := map[string]float64{"MetricName": 5.123}
		assert.Equal(t, expected, storage.gauge)
		assert.Equal(t, map[string]int64{"MetricName": -34}, storage.counter)
	})
}

func TestMemStorage_UpdateCounter(t *testing.T) {
	t.Run("Empty storage", func(t *testing.T) {
		storage := NewMemStorage()
		storage.UpdateCounter("MetricName", 53)

		assert.Equal(t, map[string]float64{}, storage.gauge)
		assert.Equal(t, map[string]int64{"MetricName": 53}, storage.counter)
	})
	t.Run("Empty storage", func(t *testing.T) {
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
	t.Run("Empty storage", func(t *testing.T) {
		storage := NewMemStorage()
		storage.UpdateCounter("MetricName", 5)
		storage.UpdateCounter("MetricName", -34)

		expected := map[string]int64{"MetricName": -29}
		assert.Equal(t, map[string]float64{}, storage.gauge)
		assert.Equal(t, expected, storage.counter)
	})

	t.Run("Not empty storage", func(t *testing.T) {
		storage := NewMemStorage()
		storage.counter = map[string]int64{"MetricName": -34}
		storage.gauge = map[string]float64{"MetricName": -34.4441}
		storage.UpdateCounter("MetricName", 10)
		storage.UpdateCounter("MetricName", 10)

		expected := map[string]int64{"MetricName": -14}
		assert.Equal(t, map[string]float64{"MetricName": -34.4441}, storage.gauge)
		assert.Equal(t, expected, storage.counter)
	})
}
