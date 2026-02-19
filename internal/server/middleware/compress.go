package middleware

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/skayfish/metrics/internal/logger"
)

// Компрессор для сжатия ответа запроса
type compressWriter struct {
	// Ответ запроса
	response http.ResponseWriter

	// Компрессор для сжатия в gzip формат
	compressor *gzip.Writer

	// Полученные несжатые данные ответа
	body string
}

// Создаёт новый компрессор, для сжатия данных ответа на запрос
//
//	@param response объект записи ответа на запрос
//	@returns *compressWriter компрессор, в случае успеха
//	@returns error ошибку, в ином случае
func newCompressWriter(response http.ResponseWriter) (*compressWriter, error) {
	compressor, err := gzip.NewWriterLevel(response, gzip.BestSpeed)
	if err != nil {
		return nil, err
	}

	return &compressWriter{response: response, compressor: compressor}, nil
}

// Возвращает заголовки ответа
//
//	@returns http.Header заголовки ответа
func (cw *compressWriter) Header() http.Header {
	return cw.response.Header()
}

// Записывает gzip сжатые данные в ответ для типов данных: application/json и text/html.
// Добавляет заголовок Content-Encoding.
//
//	@param data данные для записи
//	@returns int размер записанных данных, в случае успеха
//	@returns error ошибку в ином случае
func (cw *compressWriter) Write(data []byte) (int, error) {
	cw.body += string(data)

	contentType := cw.response.Header().Get("Content-Type")
	if contentType == "application/json" || strings.HasPrefix(contentType, "text/html") {
		cw.response.Header().Add("Content-Encoding", "gzip")
		return cw.compressor.Write(data)
	}

	cw.compressor.Reset(bytes.NewBuffer(nil))

	return cw.response.Write(data)
}

// Записывает статус ответа и сохраняет заголовки ответа
//
//	@param statusCode статус ответа
func (cw *compressWriter) WriteHeader(statusCode int) {
	cw.response.WriteHeader(statusCode)
}

// Завершает работу компрессора.
//
//	@returns error ошибку в случае неудачи
func (cw *compressWriter) Close() error {
	return cw.compressor.Close()
}

// Декомпрессор данных запроса
type compressReader struct {
	// Тело запроса
	requestBody io.ReadCloser

	// Декомпрессор gzip сжатых данных
	decompressor *gzip.Reader
}

// Создаёт новый декомпрессор данных запроса
//
//	@param requestBody тело запроса
//	@returns *compressReader декомпрессор, в случае успеха
//	@returns error ошибку в ином случае
func newCompressReader(requestBody io.ReadCloser) (*compressReader, error) {
	decompressor, err := gzip.NewReader(requestBody)
	if err != nil {
		return nil, err
	}

	return &compressReader{requestBody: requestBody, decompressor: decompressor}, nil
}

// Закрывает чтение из тела запроса и закрывает чтение декомпрессора
//
//	@returns error ошибку в случае неудачи
func (cr *compressReader) Close() error {
	if err := cr.requestBody.Close(); err != nil {
		return err
	}

	return cr.decompressor.Close()
}

// Считывает данные запроса через декомпрессор, разжимая данные в формате gzip
//
//	@param data данные запроса
//	@returns n размер считанных данных, в случае успеха
//	@returns err ошибку в ином случае
func (cr *compressReader) Read(data []byte) (n int, err error) {
	return cr.decompressor.Read(data)
}

// Middleware-обёртка, которая сжимает данные ответа на запрос при необходимости,
// и разжимает данные запроса, если они сжаты. Работает со данными, сжатыми в формате gzip
//
//	@param handler обработчик, который нужно обернуть
//	@returns http.Handler middleware-обёртку
func CompressingMiddleware(handler http.Handler) http.Handler {
	fn := func(resp http.ResponseWriter, req *http.Request) {
		responseWriter := resp
		compressorUsed := false

		for _, contentType := range req.Header.Values("Accept-Encoding") {
			if !strings.HasPrefix(contentType, "gzip") {
				continue
			}

			compressor, err := newCompressWriter(resp)
			if err != nil {
				logger.LogS.Error("middleware: CompressingMiddleware: error creating gzip compression object: %s", err)
				resp.WriteHeader(http.StatusInternalServerError)
				return
			}

			responseWriter = compressor
			compressorUsed = true
			defer compressor.Close()
			break
		}

		if req.ContentLength != 0 {
			for _, contentEncoding := range req.Header.Values("Content-Encoding") {
				if !strings.HasPrefix(contentEncoding, "gzip") {
					continue
				}

				decompressor, err := newCompressReader(req.Body)
				if err != nil {
					bodyBytes, readErr := io.ReadAll(req.Body)
					if readErr != nil {
						http.Error(resp, "middleware: CompressingMiddleware: failed to read body", http.StatusInternalServerError)
						return
					}

					defer req.Body.Close()

					logger.LogS.Errorw(fmt.Sprintf("middleware: CompressingMiddleware: error creating gzip decompression object: %s", err),
						"METHOD", req.Method,
						"URL", req.URL,
						"HEADER", req.Header,
						"BODY", string(bodyBytes))
					resp.WriteHeader(http.StatusInternalServerError)
					return
				}

				req.Body = decompressor
				defer decompressor.Close()
				break
			}
		}

		handler.ServeHTTP(responseWriter, req)

		if compressorUsed {
			logger.LogS.Debugw("HTTP Response after compressing",
				"METHOD", req.Method,
				"URL", req.URL,
				"HEADER", responseWriter.(*compressWriter).Header(),
				"BODY", responseWriter.(*compressWriter).body)
		}
	}

	return http.HandlerFunc(fn)
}
