package middleware

import (
	"net/http"
	"testing"

	"github.com/skayfish/metrics/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Проверяет создание объекта записи http ответов
func Test_newDefaultResponseWriter(t *testing.T) {
	rw := newDefaultResponseWriter()
	require.Empty(t, rw.headers)
	assert.Empty(t, rw.body)
	assert.Equal(t, -1, rw.status)
}

// Проверяет возврат заголовков объекта записи http ответов
func Test_defaultResponseWriter_Header(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		rw := newDefaultResponseWriter()
		require.Empty(t, rw.Header())
	})
	t.Run("not empty", func(t *testing.T) {
		rw := newDefaultResponseWriter()
		headers := rw.Header()
		headers.Add("key", "value1")
		headers.Add("key", "value2")
		testutil.HeadersEqual(t, map[string][]string{"Key": {"value1", "value2"}}, rw.Header())
	})
}

// Проверяет запись тела для объекта записи http ответов
func Test_defaultResponseWriter_Write(t *testing.T) {
	tests := []struct {
		test string
		data []byte
		body string
	}{
		{
			test: "empty",
			data: []byte(""),
			body: "",
		},
		{
			test: "some write",
			data: []byte("some write"),
			body: "some writesome write",
		},
		{
			test: "some write",
			data: []byte("another some write"),
			body: "another some writeanother some write",
		},
	}
	for _, tt := range tests {
		t.Run(tt.test, func(t *testing.T) {
			rw := newDefaultResponseWriter()
			num := len(tt.data)

			n, err := rw.Write(tt.data)
			require.NoError(t, err)
			assert.Equal(t, num, n)
			assert.Equal(t, string(tt.data), rw.body)

			// Повтор записи
			n, err = rw.Write(tt.data)
			require.NoError(t, err)
			assert.Equal(t, num*2, n*2)
			assert.Equal(t, tt.body, rw.body)
		})
	}
	t.Run("status", func(t *testing.T) {
		rw := newDefaultResponseWriter()

		data := []byte("some data")
		n, err := rw.Write(data)
		require.NoError(t, err)
		assert.Equal(t, len(data), n)

		assert.Equal(t, rw.status, http.StatusOK)
	})
	t.Run("status", func(t *testing.T) {
		rw := newDefaultResponseWriter()
		rw.WriteHeader(http.StatusInternalServerError)

		data := []byte("some data")
		n, err := rw.Write(data)
		require.NoError(t, err)
		assert.Equal(t, len(data), n)

		assert.Equal(t, rw.status, http.StatusInternalServerError)
	})
}

// Проверяет запись заголовков для объекта записи http ответов
func Test_defaultResponseWriter_WriteHeader(t *testing.T) {
	tests := []struct {
		test       string
		statusCode int
	}{
		{
			test:       "ok",
			statusCode: http.StatusOK,
		},
		{
			test:       "internal error",
			statusCode: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.test, func(t *testing.T) {
			rw := newDefaultResponseWriter()
			rw.WriteHeader(tt.statusCode)

			assert.Equal(t, tt.statusCode, rw.status)
		})
	}

	t.Run("do nothing", func(t *testing.T) {
		rw := newDefaultResponseWriter()
		assert.Equal(t, -1, rw.status)
	})
}
