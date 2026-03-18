package middleware

import "net/http"

// Базовый объект записи http ответов
type defaultResponseWriter struct {
	headers http.Header // Заголовки ответа
	status  int         // Статус ответа
	body    string      // Тело ответа
}

// Создаёт новый базовый объект записи http ответов
func newDefaultResponseWriter() defaultResponseWriter {
	return defaultResponseWriter{
		headers: make(http.Header),
		status:  -1,
	}
}

// Возвращает заголовки ответа
//	@returns http.Header заголовки ответа
func (rw *defaultResponseWriter) Header() http.Header {
	return rw.headers
}

// Сохраняет данные тела ответа
//	@param data данные тела ответа
//	@returns int размер данных, которые удалось записать
//	@returns error nil, необходимо для поддержки интерфейса http.ResponseWriter
func (rw *defaultResponseWriter) Write(data []byte) (int, error) {
	rw.body += string(data)
	if rw.status == -1 {
		rw.status = http.StatusOK
	}

	return len(data), nil
}

// Сохраняет статус ответа
//	@param statusCode статус ответа
func (rw *defaultResponseWriter) WriteHeader(statusCode int) {
	rw.status = statusCode
}
