package middleware

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/skayfish/metrics/internal/logger"
)

// Данные ответа
type responseData struct {
	status int    // Статус ответа
	size   uint64 // Размер данных ответа
	body   string // Данные тела ответа
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
	l.responseData.body += string(data)
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
	fn := func(resp http.ResponseWriter, req *http.Request) {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(resp, "middleware: LoggingMiddleware: failed to read body", http.StatusInternalServerError)
			return
		}

		defer req.Body.Close()

		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		logger.LogS.Infow("HTTP Request",
			"METHOD", req.Method,
			"URL", req.URL,
			"HEADER", req.Header,
			"BODY", string(bodyBytes),
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
			"METHOD", req.Method,
			"URL", req.URL,
			"HEADER", loggingResp.Header(),
			"DURATION", duration,
			"STATUS_CODE", loggingResp.responseData.status,
			"SIZE", loggingResp.responseData.size,
			"BODY", loggingResp.responseData.body,
		)
	}
	return http.HandlerFunc(fn)
}
