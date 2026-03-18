package testutil

import (
	"math"
	"net/http"
	"reflect"
	"testing"

	"github.com/skayfish/metrics/internal/model"
	"github.com/skayfish/metrics/internal/server/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Формирует хранилище по метрикам
func StorageByMetricsArray(metrics []model.Metrics) *storage.MemStorage {
	storage := storage.NewMemStorage()
	for _, metric := range metrics {
		storage.Update(metric)
	}

	return &storage
}

// Формирует словарь метрик
func MapByMetricsArray(metrics []model.Metrics) map[string]model.Metrics {
	result := make(map[string]model.Metrics, len(metrics))
	for _, m := range metrics {
		result[m.ID] = m
	}

	return result
}

// Сравнивает два вещественных числа на равенство
//
//	@param t        экземпляр теста
//	@param expected ожидаемое значение
//	@param num      реальное значение
func Float64Equal(t *testing.T, expected, num float64) {
	assert.Truef(t, math.Abs(expected-num) <= 1e-10, "expected(%f) != num(%f)", expected, num)
}

func MetricsEqual(t *testing.T, expected, target model.Metrics) {
	require.Equal(t, reflect.TypeFor[model.Metrics]().NumField(), 5)

	assert.Equal(t, expected.ID, target.ID)
	assert.Equal(t, expected.Hash, target.Hash)
	require.Equal(t, expected.MType, target.MType)
	switch expected.MType {
	case model.Gauge:
		require.NotNil(t, expected.Value)
		require.NotNil(t, target.Value)
		Float64Equal(t, *expected.Value, *target.Value)
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
func StoragesEqual(t *testing.T, expected, storage *storage.MemStorage) {
	if expected == nil || storage == nil {
		require.Equal(t, expected, storage)
		return
	}

	expectedArr, err := expected.GetAll()
	require.NoError(t, err)
	expectedMetrics := MapByMetricsArray(expectedArr)

	storageArr, err := storage.GetAll()
	require.NoError(t, err)
	storageMetrics := MapByMetricsArray(storageArr)

	if expLen, sLen := len(expectedMetrics), len(storageMetrics); len(expectedMetrics) != len(storageMetrics) {
		t.Errorf("Mismatched storage sizes: len(expected)=%d vs len(storage)=%d", expLen, sLen)
	}

	for id, metric := range expectedMetrics {
		metricRHS, found := storageMetrics[id]
		if !found {
			t.Errorf("RHS storage does not contain ID: %s", id)
		}

		MetricsEqual(t, metric, metricRHS)
	}
}

// SF TODO
func HeadersEqual(t *testing.T, expected map[string][]string, actual http.Header) {
	assert.True(t, reflect.DeepEqual(http.Header(expected), actual))
}
