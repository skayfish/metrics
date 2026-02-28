package model

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
