package model

const (
	Counter = "counter"
	Gauge   = "gauge"
)

// NOTE: Не усложняем пример, вводя иерархическую вложенность структур.
// Ограничиваясь плоской моделью.
// Delta и Value объявлены через указатели,
// что бы отличать значение "0", от не заданного значения
// и соответственно не кодировать в структуру.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

// Хранилище метрик
type MemStorage struct {
	// Данные датчиков. Ключ - название датчика, значение - данные датчика
	gauge map[string]float64

	// Данные счетчиков. Ключ - название счетчика, значение - данные счетчика
	counter map[string]int64
}

// Создаёт пустое хранилище метрик
//
//	@returns пустое хранилище метрик
func NewMemStorage() MemStorage {
	return MemStorage{
		gauge:   make(map[string]float64),
		counter: make(map[string]int64),
	}
}

// Обновляет данные датчика
//
//	@param name  название датчика
//	@param value данные датчика
func (storage *MemStorage) UpdateGauge(name string, value float64) {
	storage.gauge[name] = value
}

// Обновляет данные счетчика
//
//	@param name  название счетчика
//	@param value данные счетчика
func (storage *MemStorage) UpdateCounter(name string, value int64) {
	storage.counter[name] += value
}
