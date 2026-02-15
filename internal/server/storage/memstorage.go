package storage

import (
	"errors"
	"fmt"
	"math"

	"github.com/skayfish/metrics/internal/model"
)

// Хранилище метрик
type MemStorage map[string]model.Metrics

// Создаёт пустое хранилище метрик
//
//	@returns пустое хранилище метрик
func NewMemStorage() MemStorage {
	return make(MemStorage, 0)
}

var (
	// SF TODO
	ErrIncorrectGaugeMetricType = errors.New(`storage: incorrect metric type, expected "gauge"`)

	// SF TODO
	ErrIncorrectCounterMetricType = errors.New(`storage: incorrect metric type, expected "counter"`)
)

// SF TODO
var ErrNotFound = errors.New(`storage: metric not found`)

// Обновляет данные датчика
//
//	@param name  название метрики датчика
//	@param value данные метрики датчика
//
// SF TODO
func (ms *MemStorage) UpdateGauge(id string, value float64) error {
	metric, found := (*ms)[id]
	if !found {
		(*ms)[id] = model.Metrics{
			ID:    id,
			MType: model.Gauge,
			Value: &value,
		}

		return nil
	}

	if metric.MType != model.Gauge {
		return ErrIncorrectGaugeMetricType
	}

	*(*ms)[id].Value = value

	return nil
}

// Обновляет данные счетчика
//
//	@param name  название метрики счетчика
//	@param value данные метрики счетчика
//
// SF TODO
func (ms *MemStorage) UpdateCounter(id string, value int64) error {
	metric, found := (*ms)[id]
	if !found {
		(*ms)[id] = model.Metrics{
			ID:    id,
			MType: model.Counter,
			Delta: &value,
		}

		return nil
	}

	if metric.MType != model.Counter {
		return ErrIncorrectCounterMetricType
	}

	*(*ms)[id].Delta += value

	return nil
}

// Возвращает значение конкретной метрики датчика
//
//	@param name название метрики датчика
//	@returns value значение метрики датчика
//	@returns
//		- true - если значение нашлось,
//		- false - в ином случае
//
// SF TODO
func (ms MemStorage) GetGauge(id string) (float64, error) {
	metric, found := ms[id]
	if !found {
		return math.MaxFloat64, fmt.Errorf("%w (id: %s)", ErrNotFound, id)
	}

	if metric.MType != model.Gauge {
		return math.MaxFloat64, ErrIncorrectGaugeMetricType
	}

	return *metric.Value, nil
}

// Возвращает значение конкретной метрики счетчика
//
//	@param name название метрики счетчика
//	@returns value значение метрики счетчика
//	@returns
//		true - если значение нашлось,
//		false - в ином случае
//
// SF TODO
func (ms MemStorage) GetCounter(id string) (int64, error) {
	metric, found := ms[id]
	if !found {
		return math.MaxInt64, fmt.Errorf("%w (id: %s)", ErrNotFound, id)
	}

	if metric.MType != model.Counter {
		return math.MaxInt64, ErrIncorrectCounterMetricType
	}

	return *metric.Delta, nil
}

// Возвращает все метрики датчиков
//
//	@returns все метрики датчиков:
//		- ключ - название метрики датчика,
//		- значение - значение метрики датчика
//
// SF TODO
func (ms MemStorage) GetMetrics() map[string]model.Metrics {
	return ms
}
