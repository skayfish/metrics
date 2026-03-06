package storage

import (
	"context"
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
)

// Обновляет/добавляет метрику в хранилище
//
//	@param metric метрика для добавления/обновления
//	@returns *model.Metrics обновленную метрику, в случае успеха
//	@returns error возможную ошибку
func (ms *MemStorage) Update(metric model.Metrics) (*model.Metrics, error) {
	return ms.UpdateContext(context.Background(), metric)
}

func (ms *MemStorage) UpdateContext(ctx context.Context, metric model.Metrics) (*model.Metrics, error) {
	const prefix = "storage.MemStorage.UpdateContext"

	// SF LOGIC перенести проверку валидности выше
	if err := metric.Valid(); err != nil {
		return nil, fmt.Errorf("%s: %w", prefix, err)
	}

	foundMetric, found := (*ms)[metric.ID]
	if found {
		switch metric.MType {
		case model.Gauge:
			if foundMetric.MType != model.Gauge {
				return nil, fmt.Errorf("%s: %w", prefix, ErrFoundNotGaugeMetricType)
			}
		case model.Counter:
			if foundMetric.MType != model.Counter {
				return nil, fmt.Errorf("%s: %w", prefix, ErrFoundNotCounterMetricType)
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

// SF TODO
func (ms MemStorage) Get(id string) (*model.Metrics, error) {
	return ms.GetContext(context.Background(), id)
}

// SF TODO
func (ms MemStorage) GetContext(ctx context.Context, id string) (*model.Metrics, error) {
	metric, found := ms[id]
	if !found {
		return nil, fmt.Errorf("storage.MemStorage.Get: %w", ErrMetricNotFound)
	}

	return &metric, nil
}

// Возвращает значение конкретной метрики датчика
//
//	@param id идентификатор датчика
//	@returns float64 значение метрики датчика, в случае успеха
//	@returns error ошибку в иных случаях
func (ms MemStorage) GetGauge(id string) (float64, error) {
	metric, found := ms[id]
	if !found {
		return math.MaxFloat64, fmt.Errorf("storage: MemStorage.GetGauge: %w", ErrMetricNotFound)
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
		return math.MaxInt64, fmt.Errorf("storage: MemStorage.GetCounter: %w", ErrMetricNotFound)
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
//
// SF TODO
func (ms MemStorage) GetAll() ([]model.Metrics, error) {
	result := make([]model.Metrics, 0)
	for _, metric := range ms {
		result = append(result, metric)
	}

	return result, nil
}

// SF TODO
func (ms MemStorage) GetAllContext(ctx context.Context) ([]model.Metrics, error) {
	const prefix = "storage.MemStorage.GetAllContext"

	result := make([]model.Metrics, 0)
	for _, metric := range ms {
		if err := ctx.Err(); err != nil {
			return []model.Metrics{}, fmt.Errorf("%s: %w", prefix, err)
		}

		result = append(result, metric)
	}

	return result, nil
}

// SF TODO
var _ Storage = (*MemStorage)(nil)
