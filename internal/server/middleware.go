package server

import (
	"net/http"
	"time"

	"github.com/skayfish/metrics/internal/logger"
)

// Данные ответа
type responseData struct {
	status int    // Статус ответа
	size   uint64 // Размер данных ответа
}

// Обёртка над ответом запроса, с данными ответа
type loggingResponseWriter struct {
	responseWriter http.ResponseWriter // Ответ на запрос
	responseData   responseData        // Данные ответа
}

// Возвращает настройки ответа
//
//	@returns http.Header настройки ответа
func (l loggingResponseWriter) Header() http.Header {
	return l.responseWriter.Header()
}

// Записывает данные в ответ
//
//	@param data данные, которые нужно записать
//	@returns int размер записанных данных
//	@returns error ошибку, в случае некорректной записи данных в ответ
func (l *loggingResponseWriter) Write(data []byte) (int, error) {
	size, err := l.responseWriter.Write(data)
	l.responseData.size += uint64(size)
	return size, err
}

// Записывает статус код в ответ
//
//	@param statusCode статус код ответа
func (l *loggingResponseWriter) WriteHeader(statusCode int) {
	l.responseData.status = statusCode
	l.responseWriter.WriteHeader(statusCode)
}

// Возвращает обёртку над обработчиком запроса.
// Внутри обёртки записывает информацию о запросе и ответе в менеджер логирования
//
//	@param handler обработчик запроса
//	@returns http.Handler обёртку над обработчиком запроса
func LoggingMiddleware(handler http.Handler) http.Handler {
	logfn := func(resp http.ResponseWriter, req *http.Request) {
		logger.LogS.Infow("HTTP Request",
			"method", req.Method,
			"url", req.URL,
		)

		start := time.Now()

		loggingResp := loggingResponseWriter{
			responseWriter: resp,
			responseData: responseData{
				status: http.StatusOK,
				size:   0,
			}}
		handler.ServeHTTP(&loggingResp, req)

		duration := time.Since(start)

		logger.LogS.Infow("HTTP Response",
			"method", req.Method,
			"url", req.URL,
			"status", loggingResp.responseData.status,
			"size", loggingResp.responseData.size,
			"duration", duration,
		)
	}
	return http.HandlerFunc(logfn)
}
