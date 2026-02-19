package middleware

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type WriteErrorResponseWriter struct{}

func (WriteErrorResponseWriter) Header() http.Header { return http.Header{} }
func (*WriteErrorResponseWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("simulated write error")
}
func (*WriteErrorResponseWriter) WriteHeader(int) {}

// Проверяет создание нового компрессора для сжатия данных ответа
func Test_newCompressWriter(t *testing.T) {
	errorWriter := WriteErrorResponseWriter{}
	mockWriter := MockResponseWriter{}
	tests := []struct {
		testName string
		response http.ResponseWriter
		wantErr  bool
	}{
		{
			testName: "success",
			response: &errorWriter,
		},
		{
			testName: "success",
			response: &mockWriter,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			compressor, err := newCompressWriter(tt.response)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, compressor)
			} else {
				require.NoError(t, err)
				require.NotNil(t, compressor)
			}
		})
	}
}

// Проверяет, что компрессор для сжатия данных ответа правильно возвращает заголовки ответа на запрос
func Test_compressWriter_Header(t *testing.T) {
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
			response := MockResponseWriter{header: tt.want}
			compressor := compressWriter{response: &response, compressor: nil}
			assert.Equal(t, tt.want, compressor.Header())
			assert.Equal(t, "", compressor.body)
		})
	}
}

// Заглушка ответа на запрос
type CompressResponseWriter struct {
	compressedData string
	header         http.Header
	status         int
}

func (c *CompressResponseWriter) Header() http.Header { return c.header }
func (c *CompressResponseWriter) Write(data []byte) (int, error) {
	dataStr := string(data)
	c.compressedData += dataStr
	return len(dataStr), nil
}
func (c *CompressResponseWriter) WriteHeader(status int) { c.status = status }

// Разжимает данные, которые хранит CompressResponseWriter
func (c *CompressResponseWriter) decompress() ([]byte, error) {
	if len(c.compressedData) == 0 {
		return []byte{}, nil
	}

	buf := bytes.NewBufferString(c.compressedData)
	decompressor, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}

	b := bytes.NewBuffer(nil)
	// в переменную b записываются распакованные данные
	_, err = b.ReadFrom(decompressor)
	if err != nil {
		return nil, fmt.Errorf("failed decompress data: %w", err)
	}

	err = decompressor.Close()
	if err != nil {
		return nil, fmt.Errorf("failed close decompressor: %w", err)
	}

	return b.Bytes(), nil
}

// Проверяет правильную работу записи данных с использованием компрессора
func Test_compressWriter_Write(t *testing.T) {
	tests := []struct {
		header http.Header
	}{
		{header: http.Header{"Content-Type": {}}},
		{header: http.Header{"Content-Type": {""}}},
		{header: http.Header{"Content-Type": {"", "application/json", "text/html"}}},
		{header: http.Header{"Content-Type": {"text/plain"}}},
	}
	for _, tt := range tests {
		t.Run("no compress", func(t *testing.T) {
			writer := CompressResponseWriter{header: tt.header}
			compressor, _ := newCompressWriter(&writer)

			// first try
			data := []byte("success write")
			num, err := compressor.Write(data)
			require.NoError(t, err)
			assert.Equal(t, len(data), num)
			assert.Equal(t, string(data), compressor.body)
			assert.True(t, reflect.DeepEqual(tt.header, writer.header))

			_, err = writer.decompress()
			require.Error(t, err)

			// second try
			newData := []byte("success write: one more")
			num, err = compressor.Write(newData)
			require.NoError(t, err)
			assert.Equal(t, len(newData), num)
			assert.Equal(t, string(data)+string(newData), compressor.body)
			require.True(t, reflect.DeepEqual(tt.header, writer.header))

			_, err = writer.decompress()
			require.Error(t, err)
		})
	}

	tests = []struct {
		header http.Header
	}{
		{header: http.Header{"Content-Type": {"application/json"}}},
		{header: http.Header{"Content-Type": {"text/html"}}},
	}
	for _, tt := range tests {
		t.Run("success compress", func(t *testing.T) {
			writer := CompressResponseWriter{header: tt.header}
			compressor, _ := newCompressWriter(&writer)

			expectedHeader := tt.header

			// first try
			data := []byte("success write")
			num, err := compressor.Write(data)
			require.NoError(t, err)
			assert.Equal(t, len(data), num)
			assert.Equal(t, string(data), compressor.body)
			expectedHeader.Add("Content-Encoding", "gzip")
			require.True(t, reflect.DeepEqual(expectedHeader, writer.header))

			decompressedData, err := writer.decompress()
			require.ErrorIs(t, err, io.ErrUnexpectedEOF)
			assert.Empty(t, decompressedData)

			// second try
			newData := []byte("success write: one more")
			num, err = compressor.Write(newData)
			require.NoError(t, compressor.compressor.Close())
			require.NoError(t, err)
			assert.Equal(t, len(newData), num)
			assert.Equal(t, string(data)+string(newData), compressor.body)
			expectedHeader.Add("Content-Encoding", "gzip")
			require.True(t, reflect.DeepEqual(expectedHeader, writer.header))

			decompressedData, err = writer.decompress()
			require.NotErrorIs(t, err, io.ErrUnexpectedEOF)
			assert.Equal(t, []byte(string(data)+string(newData)), decompressedData)
		})
	}

	t.Run("error write", func(t *testing.T) {
		writer := WriteErrorResponseWriter{}
		compressor, _ := newCompressWriter(&writer)

		data := []byte("success write")
		num, err := compressor.Write(data)
		require.Error(t, err)
		assert.Equal(t, 0, num)
		assert.Equal(t, string(data), compressor.body)

		newData := []byte("success write: one more")
		num, err = compressor.Write(newData)
		require.Error(t, err)
		assert.Equal(t, 0, num)
		assert.Equal(t, string(data)+string(newData), compressor.body)
	})
}

// Проверяет правильную запись статуса ответа на запрос
func Test_compressWriter_WriteHeader(t *testing.T) {
	tests := []struct {
		testName string
		status   int
		header   http.Header
	}{
		{
			testName: "unknown status",
			status:   0,
		},
		{
			testName: "ok",
			status:   http.StatusOK,
			header:   http.Header{"Content-Type": {"application/json"}},
		},
		{
			testName: "not found",
			status:   http.StatusNotFound,
			header:   http.Header{"Content-Type": {"text/html"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			response := MockResponseWriter{}
			compressor := compressWriter{response: &response}
			compressor.WriteHeader(tt.status)
			assert.Equal(t, tt.status, response.status)
			assert.Equal(t, "", compressor.body)
			assert.Nil(t, compressor.compressor)

			// with compressing headers
			response = MockResponseWriter{header: tt.header}
			compressor = compressWriter{response: &response}
			compressor.WriteHeader(tt.status)
			assert.Equal(t, tt.status, response.status)
			assert.Equal(t, "", compressor.body)
			assert.Nil(t, compressor.compressor)
		})
	}
}

// Проверяет правильное закрытие компрессора
func Test_compressWriter_Close(t *testing.T) {
	tests := []struct {
		header http.Header
	}{
		{header: http.Header{"Content-Type": {"application/json"}}},
		{header: http.Header{"Content-Type": {"text/html"}}},
	}
	for _, tt := range tests {
		t.Run("success compress: close immediately", func(t *testing.T) {
			writer := CompressResponseWriter{header: tt.header}
			compressor, _ := newCompressWriter(&writer)

			require.NoError(t, compressor.Close())

			_, err := compressor.Write([]byte("some data"))
			require.Error(t, err)

			decompressedData, err := writer.decompress()
			require.NoError(t, err)
			assert.Empty(t, decompressedData)
		})
		t.Run("success compress: after write", func(t *testing.T) {
			writer := CompressResponseWriter{header: tt.header}
			compressor, _ := newCompressWriter(&writer)

			data := []byte("some data")
			num, err := compressor.Write(data)
			require.NoError(t, err)
			assert.Equal(t, len(data), num)

			decompressedData, err := writer.decompress()
			require.ErrorIs(t, err, io.ErrUnexpectedEOF)
			assert.Empty(t, decompressedData)

			require.NoError(t, compressor.Close())

			decompressedData, err = writer.decompress()
			require.NoError(t, err)
			assert.Equal(t, data, decompressedData)
		})
	}

	t.Run("no compress: close immediately", func(t *testing.T) {
		writer := CompressResponseWriter{header: http.Header{}}
		compressor, _ := newCompressWriter(&writer)

		require.NoError(t, compressor.Close())

		_, err := compressor.Write([]byte("some data"))
		require.NoError(t, err)

		decompressedData, err := writer.decompress()
		require.ErrorIs(t, err, io.ErrUnexpectedEOF)
		assert.Empty(t, decompressedData)
	})
	t.Run("no compress: after write", func(t *testing.T) {
		writer := CompressResponseWriter{header: http.Header{}}
		compressor, _ := newCompressWriter(&writer)

		data := []byte("some data")
		num, err := compressor.Write(data)
		require.NoError(t, err)
		assert.Equal(t, len(data), num)

		decompressedData, err := writer.decompress()
		require.ErrorIs(t, err, io.ErrUnexpectedEOF)
		assert.Empty(t, decompressedData)

		require.NoError(t, compressor.Close())

		decompressedData, err = writer.decompress()
		require.Error(t, err)
		assert.Empty(t, decompressedData)
	})
}

// Сжимает полученные данные
func compress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w, err := gzip.NewWriterLevel(&b, flate.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("failed init compress writer: %v", err)
	}

	_, err = w.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed write data to compress temporary buffer: %v", err)
	}

	err = w.Close()
	if err != nil {
		return nil, fmt.Errorf("failed compress data: %v", err)
	}

	return b.Bytes(), nil
}

// Проверяет создание нового декомпрессора данных ответа на запрос
func Test_newCompressReader(t *testing.T) {
	tests := []struct {
		testName    string
		requestBody string
	}{
		{testName: "success", requestBody: ""},
		{testName: "success", requestBody: "request body"},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			compressedReqBody, err := compress([]byte(tt.requestBody))
			require.NoError(t, err)

			request, err := http.NewRequest("", "", bytes.NewReader(compressedReqBody))
			require.NoError(t, err)

			decompressor, err := newCompressReader(request.Body)
			require.NoError(t, err)
			require.NotNil(t, decompressor)
		})
	}

	tests = []struct {
		testName    string
		requestBody string
	}{
		{testName: "failed", requestBody: ""},
		{testName: "failed", requestBody: "request body"},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			request, err := http.NewRequest("", "", bytes.NewReader([]byte(tt.requestBody)))
			require.NoError(t, err)

			decompressor, err := newCompressReader(request.Body)
			require.Error(t, err)
			require.Nil(t, decompressor)
		})
	}
}

// Проверяет правильное закрытие декомпрессора
func Test_compressReader_Close(t *testing.T) {
	tests := []struct {
		requestBody string
	}{
		{requestBody: ""},
		{requestBody: "request body"},
	}
	for _, tt := range tests {
		t.Run("success", func(t *testing.T) {
			compressedReqBody, err := compress([]byte(tt.requestBody))
			require.NoError(t, err)

			request, err := http.NewRequest("", "", bytes.NewReader(compressedReqBody))
			require.NoError(t, err)

			decompressor, err := newCompressReader(request.Body)
			require.NoError(t, err)
			require.NotNil(t, decompressor)

			require.NoError(t, decompressor.Close())
		})
	}
}

// Проверяет работу middleware-обёртки, которая сжимает данные ответа на запрос при необходимости,
// и разжимает данные запроса, если они сжаты. Работает со сжатием gzip
func TestCompressingMiddleware(t *testing.T) {
	validRequestBody := "{\"valid\": \"json\"}"

	checkRequest := func(r *http.Request) {
		data, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		assert.Equal(t, validRequestBody, string(data))
	}

	handlerJSON := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkRequest(r)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{\"key\":4431}"))
	})
	handlerHTML := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkRequest(r)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<body>html text</body>"))
	})
	handlerText := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkRequest(r)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("some text"))
	})

	type request struct {
		header     http.Header
		body       string
		compressed bool
	}
	type response struct {
		body       string
		compressed bool
	}

	prepareRequest := func(r request) *http.Request {
		body := r.body

		if r.compressed {
			data, err := compress([]byte(r.body))
			require.NoError(t, err)
			body = string(data)
		}

		request, err := http.NewRequest("post", "host:port", strings.NewReader(body))
		require.NoError(t, err)
		for tag, values := range r.header {
			for _, value := range values {
				request.Header.Add(tag, value)
			}
		}

		return request
	}

	tests := []struct {
		testName string
		request  request
		response response
	}{
		{
			testName: "accept encoding is empty",
			request: request{
				header:     http.Header{"Accept-Encoding": {""}},
				body:       validRequestBody,
				compressed: false,
			},
			response: response{
				compressed: false,
			},
		},
		{
			testName: "accept encoding is not gzip",
			request: request{
				header:     http.Header{"Accept-Encoding": {"br", "deflate"}},
				body:       validRequestBody,
				compressed: false,
			},
			response: response{
				compressed: false,
			},
		},
		{
			testName: "accept encoding is gzip",
			request: request{
				header:     http.Header{"Accept-Encoding": {"gzip"}},
				body:       validRequestBody,
				compressed: false,
			},
			response: response{
				compressed: true,
			},
		},
		{
			testName: "content encoding is empty",
			request: request{
				header:     http.Header{"Content-Encoding": {""}},
				body:       validRequestBody,
				compressed: false,
			},
			response: response{
				compressed: false,
			},
		},
		{
			testName: "content encoding is not gzip",
			request: request{
				header:     http.Header{"Content-Encoding": {"br"}},
				body:       validRequestBody,
				compressed: false,
			},
			response: response{
				compressed: false,
			},
		},
		{
			testName: "content encoding is gzip",
			request: request{
				header:     http.Header{"Content-Encoding": {"gzip"}},
				body:       validRequestBody,
				compressed: true,
			},
			response: response{
				compressed: false,
			},
		},
		{
			testName: "full gzip transfer",
			request: request{
				header:     http.Header{"Content-Encoding": {"gzip"}, "Accept-Encoding": {"gzip"}},
				body:       validRequestBody,
				compressed: true,
			},
			response: response{
				compressed: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			request := prepareRequest(tt.request)
			response := CompressResponseWriter{header: http.Header{}}
			CompressingMiddleware(handlerJSON).ServeHTTP(&response, request)

			assert.Equal(t, []string{"application/json"}, response.header.Values("Content-Type"))
			if tt.response.compressed {
				assert.Equal(t, []string{"gzip"}, response.header.Values("Content-Encoding"))
				decompressedData, err := response.decompress()
				require.NoError(t, err)
				assert.Equal(t, []byte("{\"key\":4431}"), decompressedData)
			} else {
				assert.Empty(t, response.header.Values("Content-Encoding"))
				assert.Equal(t, "{\"key\":4431}", response.compressedData)
			}

			request = prepareRequest(tt.request)
			response = CompressResponseWriter{header: http.Header{}}
			CompressingMiddleware(handlerHTML).ServeHTTP(&response, request)

			assert.Equal(t, []string{"text/html"}, response.header.Values("Content-Type"))
			if tt.response.compressed {
				assert.Equal(t, []string{"gzip"}, response.header.Values("Content-Encoding"))
				decompressedData, err := response.decompress()
				require.NoError(t, err)
				assert.Equal(t, []byte("<body>html text</body>"), decompressedData)
			} else {
				assert.Empty(t, response.header.Values("Content-Encoding"))
				assert.Equal(t, "<body>html text</body>", response.compressedData)
			}

			request = prepareRequest(tt.request)
			response = CompressResponseWriter{header: http.Header{}}
			CompressingMiddleware(handlerText).ServeHTTP(&response, request)

			assert.Equal(t, []string{"text/plain"}, response.header.Values("Content-Type"))
			assert.Empty(t, response.header.Values("Content-Encoding"))
			assert.Equal(t, "some text", response.compressedData)
		})
	}

	t.Run("failed create decompressor", func(t *testing.T) {
		req, err := http.NewRequest("", "", bytes.NewReader([]byte("some data")))
		require.NoError(t, err)

		header := http.Header{"Content-Encoding": {"gzip"}, "Accept-Encoding": {"gzip"}}
		for tag, values := range header {
			for _, value := range values {
				req.Header.Add(tag, value)
			}
		}

		response := CompressResponseWriter{header: http.Header{}}
		CompressingMiddleware(handlerJSON).ServeHTTP(&response, req)
		assert.Equal(t, http.StatusInternalServerError, response.status)
	})
}
