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
	ErrFoundNotGaugeMetricType = errors.New(`found not "gauge" metric type`)

	// SF TODO
	ErrFoundNotCounterMetricType = errors.New(`found not "counter" metric type`)

	// SF TODO
	ErrUnrecognizedMetricType = errors.New(`unrecognized metric type. Supported types: "gauge", "counter"`)
)

var (
	// SF TODO
	ErrValueIsEmpty = errors.New(`gauge metric value is empty`)

	// SF TODO
	ErrDeltaIsEmpty = errors.New(`counter metric delta is empty`)
)

// SF TODO
var ErrNotFound = errors.New(`metric not found`)

// SF TODO
func (ms *MemStorage) Update(metric model.Metrics) error {
	switch metric.MType {
	case model.Gauge:
		if metric.Value == nil {
			return fmt.Errorf("storage: MemStorage.Update: %w", ErrValueIsEmpty)
		}
	case model.Counter:
		if metric.Delta == nil {
			return fmt.Errorf("storage: MemStorage.Update: %w", ErrDeltaIsEmpty)
		}
	default:
		return fmt.Errorf("storage: MemStorage.Update: %w", ErrUnrecognizedMetricType)
	}

	foundMetric, found := (*ms)[metric.ID]
	if found {
		switch metric.MType {
		case model.Gauge:
			if foundMetric.MType != model.Gauge {
				return fmt.Errorf("storage: MemStorage.Update: %w", ErrFoundNotGaugeMetricType)
			}
		case model.Counter:
			if foundMetric.MType != model.Counter {
				return fmt.Errorf("storage: MemStorage.Update: %w", ErrFoundNotCounterMetricType)
			}

			*metric.Delta += *foundMetric.Delta
		}
	}

	(*ms)[metric.ID] = metric

	return nil
}

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
		return ErrFoundNotGaugeMetricType
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
		return ErrFoundNotCounterMetricType
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
		return math.MaxFloat64, fmt.Errorf("storage: MemStorage.GetGauge: %w", ErrNotFound)
	}

	if metric.MType != model.Gauge {
		return math.MaxFloat64, fmt.Errorf("storage: MemStorage.GetGauge: %w", ErrFoundNotGaugeMetricType)
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
		return math.MaxInt64, fmt.Errorf("storage: MemStorage.GetCounter: %w", ErrNotFound)
	}

	if metric.MType != model.Counter {
		return math.MaxInt64, fmt.Errorf("storage: MemStorage.GetCounter: %w", ErrFoundNotCounterMetricType)
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
