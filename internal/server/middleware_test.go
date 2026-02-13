package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ResponseMockWriter struct {
	header http.Header
}

func (r ResponseMockWriter) Header() http.Header        { return r.header }
func (r *ResponseMockWriter) Write([]byte) (int, error) { return 50, nil }
func (r *ResponseMockWriter) WriteHeader(int)           {}

// Проверяет, что обёртка с логированием над ответом запроса возвращает правильные настройки ответа
func Test_loggingResponseWriter_Header(t *testing.T) {
	tests := []struct {
		testName   string
		headerData map[string][]string
	}{
		{
			testName:   "not empty header",
			headerData: map[string][]string{"Content-Type": {"text/plain"}},
		},
		{
			testName:   "empty header",
			headerData: map[string][]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			resp := loggingResponseWriter{
				responseWriter: &ResponseMockWriter{header: http.Header(tt.headerData)},
			}
			assert.Equal(t, http.Header(tt.headerData), resp.Header())
		})
	}
}

// Проверяет, что обёртка с логированием над ответом запроса правильно записывает размер данных ответа
func Test_loggingResponseWriter_Write(t *testing.T) {
	resp := loggingResponseWriter{
		responseWriter: &ResponseMockWriter{},
	}

	size, err := resp.Write([]byte("first"))
	require.NoError(t, err)
	assert.Equal(t, 50, size)
	assert.Equal(t, uint64(50), resp.responseData.size)

	size, err = resp.Write([]byte("second"))
	require.NoError(t, err)
	assert.Equal(t, 50, size)
	assert.Equal(t, uint64(100), resp.responseData.size)

	size, err = resp.Write([]byte("third"))
	require.NoError(t, err)
	assert.Equal(t, 50, size)
	assert.Equal(t, uint64(150), resp.responseData.size)
}

// Проверяет, что обёртка с логированием над ответом запроса правильно записывает статус ответа
func Test_loggingResponseWriter_WriteHeader(t *testing.T) {
	tests := []struct {
		testName string
		status   int
	}{
		{
			testName: "unknown status",
			status:   0,
		},
		{
			testName: "ok",
			status:   http.StatusOK,
		},
		{
			testName: "not found",
			status:   http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			resp := loggingResponseWriter{
				responseWriter: &ResponseMockWriter{},
			}
			resp.WriteHeader(tt.status)
			assert.Equal(t, tt.status, resp.responseData.status)
		})
	}
}

// Проверяет, что обёртка с логированием вызывает переданную ей функцию
func TestLoggingMiddleware(t *testing.T) {
	tests := []struct {
		testName string
		handler  http.Handler
		panic    bool
	}{
		{
			testName: "panic",
			handler: http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
				panic("panic")
			}),
			panic: true,
		},
		{
			testName: "no panic",
			handler:  http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {

			req, err := http.NewRequest("post", "http://localhost:8080/", strings.NewReader(""))
			require.NoError(t, err)
			middleware := func() { LoggingMiddleware(tt.handler).ServeHTTP(&ResponseMockWriter{}, req) }
			if tt.panic {
				require.Panics(t, middleware)
			} else {
				require.NotPanics(t, middleware)
			}
		})
	}
}
