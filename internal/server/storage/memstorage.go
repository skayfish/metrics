package storage

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

// SF TODO
func (storage *MemStorage) GetGauge(name string) (value float64, ok bool) {
	value, ok = storage.gauge[name]
	return
}

// SF TODO
func (storage *MemStorage) GetCounter(name string) (value int64, ok bool) {
	value, ok = storage.counter[name]
	return
}
