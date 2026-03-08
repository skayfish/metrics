package model

import (
	"errors"
)

const (
	// Метрика типа "счетчик"
	Counter = "counter"

	// Метрика типа "датчик"
	Gauge = "gauge"
)

// Данные метрики
type Metrics struct {
	ID    string   `json:"id"`              // Идентификатор метрики
	MType string   `json:"type"`            // Тип метрики [gauge, counter]
	Delta *int64   `json:"delta,omitempty"` // Значение метрики для counter типа
	Value *float64 `json:"value,omitempty"` // Значение метрики для gauge типа
	Hash  string   `json:"hash,omitempty"`
}

var (
	// Ошибка: значение метрики типа gauge - пустое
	ErrGaugeValueIsEmpty = errors.New(`gauge metric value is empty`)

	// Ошибка: значение метрики типа counter - пустое
	ErrCounterDeltaIsEmpty = errors.New(`counter metric delta is empty`)

	// Ошибка: неизвестный тип метрики
	ErrUnrecognizedMetricType = errors.New(`unrecognized metric type. Supported types: "gauge", "counter"`)
)

// Проверяет метрику на валидность
//	@returns error ошибку в случае некорректности
//	@returns nil если метрика валидна
func (m Metrics) Valid() error {
	switch m.MType {
	case Gauge:
		if m.Value == nil {
			return ErrGaugeValueIsEmpty
		}
	case Counter:
		if m.Delta == nil {
			return ErrCounterDeltaIsEmpty
		}
	default:
		return ErrUnrecognizedMetricType
	}

	return nil
}
