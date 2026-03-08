package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/skayfish/metrics/internal/logger"
	"github.com/skayfish/metrics/internal/model"
)

// Хранилище метрик в памяти приложения
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

// Ищет метрику в хранилище
//
//	@param id идентификатор метрики
//	@returns *model.Metrics метрика из хранилища
//	@returns error ошибку, если не удалось найти метрику
func (ms MemStorage) Get(id string) (*model.Metrics, error) {
	return ms.GetContext(context.Background(), id)
}

// Ищет метрику в хранилище
//
//	@param ctx контекст для завершения работы
//	@param id  идентификатор метрики
//	@returns *model.Metrics метрика из хранилища
//	@returns error ошибку, если не удалось найти метрику
func (ms MemStorage) GetContext(ctx context.Context, id string) (*model.Metrics, error) {
	metric, found := ms[id]
	if !found {
		return nil, fmt.Errorf("storage.MemStorage.GetContext: %w", ErrMetricNotFound)
	}

	return &metric, nil
}

// Возвращает все метрики в хранилище
//
//	@returns []model.Metrics метрики в хранилище
//	@returns error возвращает nil, нужно для удовлетворению интерфейса Storage
func (ms MemStorage) GetAll() ([]model.Metrics, error) {
	result := make([]model.Metrics, 0)
	for _, metric := range ms {
		result = append(result, metric)
	}

	return result, nil
}

// Возвращает все метрики в хранилище
//
//	@param ctx контекст для завершения работы
//	@returns []model.Metrics метрики в хранилище
//	@returns error ошибку, если контекст стал ошибочным
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

// Сохраняет данные хранилища метрик в json файл
//
//	@param filePath путь к json файлу
//	@returns error ошибку, если возникли проблемы при сохранении
func (ms MemStorage) SaveStorageToFile(filePath string) error {
	logger.LogS.Debugw("Save metrics storage to file", "file", filePath, "metrics storage", ms)

	metrics := make([]model.Metrics, 0, len(ms))
	for _, metric := range ms {
		metrics = append(metrics, metric)
	}

	metricsJSON, err := json.MarshalIndent(metrics, "", "    ")
	if err != nil {
		return fmt.Errorf("failed marshal metrics: %w", err)
	}

	if err = os.WriteFile(filePath, metricsJSON, 0644); err != nil {
		return fmt.Errorf("failed write to file %q: %w", filePath, err)
	}

	return nil
}

// Ничего не делает. Необходимо для удовлетворения интерфейсу [Storage]
//
//	@returns error nil
func (ms MemStorage) Close() error {
	return nil
}

// Проверка, что [MemStorage] удовлетворяет интерфейсу [Storage]
var _ Storage = (*MemStorage)(nil)
