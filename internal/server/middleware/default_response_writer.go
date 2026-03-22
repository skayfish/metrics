package middleware

import "net/http"

// Базовый объект записи http ответов
type defaultResponseWriter struct {
	Headers http.Header // Заголовки ответа
	Status  int         // Статус ответа
	Body    string      // Тело ответа
}

// Создаёт новый базовый объект записи http ответов
func NewDefaultResponseWriter() defaultResponseWriter {
	return defaultResponseWriter{
		Headers: make(http.Header),
		Status:  -1,
	}
}

// Возвращает заголовки ответа
//	@returns http.Header заголовки ответа
func (rw *defaultResponseWriter) Header() http.Header {
	return rw.Headers
}

// Сохраняет данные тела ответа
//	@param data данные тела ответа
//	@returns int размер данных, которые удалось записать
//	@returns error nil, необходимо для поддержки интерфейса http.ResponseWriter
func (rw *defaultResponseWriter) Write(data []byte) (int, error) {
	rw.Body += string(data)
	if rw.Status == -1 {
		rw.Status = http.StatusOK
	}

	return len(data), nil
}

// Сохраняет статус ответа
//	@param statusCode статус ответа
func (rw *defaultResponseWriter) WriteHeader(statusCode int) {
	rw.Status = statusCode
}
