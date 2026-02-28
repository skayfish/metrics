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
	// Ошибка: найден не gauge тип метрики в хранилище
	ErrFoundNotGaugeMetricType = errors.New(`found not "gauge" metric type`)

	// Ошибка: найден не counter тип метрики в хранилище
	ErrFoundNotCounterMetricType = errors.New(`found not "counter" metric type`)

	// Ошибка: неизвестный тип метрики
	ErrUnrecognizedMetricType = errors.New(`unrecognized metric type. Supported types: "gauge", "counter"`)
)

var (
	// Ошибка: значение метрики типа gauge - пустое
	ErrValueIsEmpty = errors.New(`gauge metric value is empty`)

	// Ошибка: значение метрики типа counter - пустое
	ErrDeltaIsEmpty = errors.New(`counter metric delta is empty`)
)

// Ошибка: метрика не найдена в хранилище
var ErrNotFound = errors.New(`metric not found`)

// Обновляет/добавляет метрику в хранилище
//
//	@param metric метрика для добавления/обновления
//	@returns *model.Metrics обновленную метрику, в случае успеха
//	@returns error возможную ошибку
func (ms *MemStorage) Update(metric model.Metrics) (*model.Metrics, error) {
	switch metric.MType {
	case model.Gauge:
		if metric.Value == nil {
			return nil, fmt.Errorf("storage: MemStorage.Update: %w", ErrValueIsEmpty)
		}
	case model.Counter:
		if metric.Delta == nil {
			return nil, fmt.Errorf("storage: MemStorage.Update: %w", ErrDeltaIsEmpty)
		}
	default:
		return nil, fmt.Errorf("storage: MemStorage.Update: %w", ErrUnrecognizedMetricType)
	}

	foundMetric, found := (*ms)[metric.ID]
	if found {
		switch metric.MType {
		case model.Gauge:
			if foundMetric.MType != model.Gauge {
				return nil, fmt.Errorf("storage: MemStorage.Update: %w", ErrFoundNotGaugeMetricType)
			}
		case model.Counter:
			if foundMetric.MType != model.Counter {
				return nil, fmt.Errorf("storage: MemStorage.Update: %w", ErrFoundNotCounterMetricType)
			}

			*metric.Delta += *foundMetric.Delta
		}
	}

	(*ms)[metric.ID] = metric

	return &metric, nil
}

// Обновляет/добавляет датчик в хранилище
//
//	@param id    идентификатор датчика
//	@param value данные метрики датчика
//	@returns возможную ошибку
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
		return fmt.Errorf("storage: MemStorage.UpdateGauge: %w", ErrFoundNotGaugeMetricType)
	}

	*(*ms)[id].Value = value

	return nil
}

// Обновляет данные счетчика
//
//	@param name  идентификатор счетчика
//	@param value данные метрики счетчика
//	@returns возможную ошибку
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
		return fmt.Errorf("storage: MemStorage.UpdateCounter: %w", ErrFoundNotCounterMetricType)
	}

	*(*ms)[id].Delta += value

	return nil
}

// Возвращает значение конкретной метрики датчика
//
//	@param id идентификатор датчика
//	@returns float64 значение метрики датчика, в случае успеха
//	@returns error ошибку в иных случаях
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
//	@param id идентификатор счетчика
//	@returns float64 значение метрики счетчика, в случае успеха
//	@returns error ошибку в иных случаях
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
//		- ключ     - идентификатор метрики,
//		- значение - данные метрики
func (ms MemStorage) GetMetrics() map[string]model.Metrics {
	return ms
}
