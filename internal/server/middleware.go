package server

import (
	"net/http"
	"time"

	"github.com/skayfish/metrics/internal/logger"
)

// SF TODO
type responseData struct {
	// SF TODO
	status int

	// SF TODO
	size uint64
}

// SF TODO
type loggingResponseWriter struct {
	responseWriter http.ResponseWriter
	responseData   responseData
}

// SF TODO
func (l loggingResponseWriter) Header() http.Header {
	return l.responseWriter.Header()
}

// SF TODO
func (l *loggingResponseWriter) Write(data []byte) (int, error) {
	size, err := l.responseWriter.Write(data)
	l.responseData.size += uint64(size)
	return size, err
}

// SF TODO
func (l *loggingResponseWriter) WriteHeader(statusCode int) {
	l.responseData.status = statusCode
	l.responseWriter.WriteHeader(statusCode)
}

// SF TODO
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
