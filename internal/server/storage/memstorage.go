package storage

// Хранилище метрик
type MemStorage struct {
	// Данные датчиков. Ключ - название метрики датчика, значение - данные датчика
	gauge map[string]float64

	// Данные счетчиков. Ключ - название метрики счетчика, значение - данные счетчика
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
//	@param name  название метрики датчика
//	@param value данные метрики датчика
func (storage *MemStorage) UpdateGauge(name string, value float64) {
	storage.gauge[name] = value
}

// Обновляет данные счетчика
//
//	@param name  название метрики счетчика
//	@param value данные метрики счетчика
func (storage *MemStorage) UpdateCounter(name string, value int64) {
	storage.counter[name] += value
}

// Возвращает значение конкретной метрики датчика
//	@param name название метрики датчика
//	@returns value значение метрики датчика
//	@returns
//		- true - если значение нашлось,
//		- false - в ином случае
func (storage *MemStorage) GetGauge(name string) (value float64, ok bool) {
	value, ok = storage.gauge[name]
	return
}

// Возвращает значение конкретной метрики счетчика
//	@param name название метрики счетчика
//	@returns value значение метрики счетчика
//	@returns
//		true - если значение нашлось,
//		false - в ином случае
func (storage *MemStorage) GetCounter(name string) (value int64, ok bool) {
	value, ok = storage.counter[name]
	return
}

// Возвращает все метрики датчиков
//	@returns все метрики датчиков:
//		- ключ - название метрики датчика,
//		- значение - значение метрики датчика
func (storage *MemStorage) GetGauges() map[string]float64 {
	return storage.gauge
}

// Возвращает все метрики счетчиков
//	@returns все метрики счетчиков:
//		- ключ - название метрики счетчика,
//		- значение - значение метрики счетчика
func (storage *MemStorage) GetCounters() map[string]int64 {
	return storage.counter
}
