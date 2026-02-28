package middleware

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockResponseWriter struct {
	header http.Header
	status int
}

func (r MockResponseWriter) Header() http.Header        { return r.header }
func (r *MockResponseWriter) Write([]byte) (int, error) { return 50, nil }
func (r *MockResponseWriter) WriteHeader(status int)    { r.status = status }

// Проверяет, что обёртка с логированием над ответом запроса возвращает правильные заголовки ответа
func Test_loggingResponseWriter_Header(t *testing.T) {
	tests := []struct {
		testName string
		want     http.Header
	}{
		{
			testName: "not empty header",
			want:     http.Header{"Content-Type": {"text/plain"}},
		},
		{
			testName: "empty header",
			want:     http.Header{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			resp := loggingResponseWriter{responseWriter: &MockResponseWriter{header: tt.want}}
			assert.Equal(t, tt.want, resp.Header())
		})
	}
}

// Проверяет, что обёртка с логированием над ответом запроса правильно записывает размер данных ответа
func Test_loggingResponseWriter_Write(t *testing.T) {
	resp := loggingResponseWriter{
		responseWriter: &MockResponseWriter{},
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
				responseWriter: &MockResponseWriter{},
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
			middleware := func() { LoggingMiddleware(tt.handler).ServeHTTP(&MockResponseWriter{}, req) }
			if tt.panic {
				require.Panics(t, middleware)
			} else {
				require.NotPanics(t, middleware)
			}
		})
	}
}
