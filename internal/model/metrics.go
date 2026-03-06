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
	ErrValueIsEmpty = errors.New(`gauge metric value is empty`)

	// Ошибка: значение метрики типа counter - пустое
	ErrDeltaIsEmpty = errors.New(`counter metric delta is empty`)

	// Ошибка: неизвестный тип метрики
	ErrUnrecognizedMetricType = errors.New(`unrecognized metric type. Supported types: "gauge", "counter"`)
)

// SF TODO
func (m Metrics) Valid() error {
	switch m.MType {
	case Gauge:
		if m.Value == nil {
			return ErrValueIsEmpty
		}
	case Counter:
		if m.Delta == nil {
			return ErrDeltaIsEmpty
		}
	default:
		return ErrUnrecognizedMetricType
	}

	return nil
}
