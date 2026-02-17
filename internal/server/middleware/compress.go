package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/skayfish/metrics/internal/logger"
)

// SF TODO
type compressWriter struct {
	// SF TODO
	response http.ResponseWriter

	// SF TODO
	compressor *gzip.Writer
}

// SF TODO
func newCompressWriter(response http.ResponseWriter) (*compressWriter, error) {
	compressor, err := gzip.NewWriterLevel(response, gzip.BestSpeed)
	if err != nil {
		return nil, err
	}

	return &compressWriter{response: response, compressor: compressor}, nil
}

// SF TODO
func (cw *compressWriter) Header() http.Header {
	return cw.response.Header()
}

// SF TODO
func (cw *compressWriter) Write(data []byte) (int, error) {
	contentType := cw.response.Header().Get("Content-Type")
	if contentType == "application/json" || strings.HasPrefix(contentType, "text/html") {
		cw.response.Header().Add("Content-Encoding", "gzip")
		return cw.compressor.Write(data)
	}

	return cw.response.Write(data)
}

// SF TODO
func (cw *compressWriter) WriteHeader(statusCode int) {
	cw.response.WriteHeader(statusCode)
}

// SF TODO
func (cw *compressWriter) Close() error {
	return cw.compressor.Close()
}

// SF TODO
type compressReader struct {
	// SF TODO
	requestBody io.ReadCloser

	// SF TODO
	decompressor *gzip.Reader
}

// SF TODO
func newCompressReader(requestBody io.ReadCloser) (*compressReader, error) {
	decompressor, err := gzip.NewReader(requestBody)
	if err != nil {
		return nil, err
	}

	return &compressReader{requestBody: requestBody, decompressor: decompressor}, nil
}

// SF TODO
func (cr *compressReader) Close() error {
	if err := cr.requestBody.Close(); err != nil {
		return err
	}

	return cr.decompressor.Close()
}

// SF TODO
func (cr *compressReader) Read(data []byte) (n int, err error) {
	return cr.decompressor.Read(data)
}

// SF TODO
func CompressingMiddleware(handler http.Handler) http.Handler {
	fn := func(resp http.ResponseWriter, req *http.Request) {
		responseWriter := resp

		if strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
			compressor, err := newCompressWriter(resp)
			if err != nil {
				logger.LogS.Error("middleware: CompressingMiddleware: error creating gzip compression object: %s", err)
				resp.WriteHeader(http.StatusInternalServerError)
				return
			}

			responseWriter = compressor
			defer compressor.Close()
		}

		if strings.Contains(req.Header.Get("Content-Encoding"), "gzip") {
			decompressor, err := newCompressReader(req.Body)
			if err != nil {
				logger.LogS.Errorf("middleware: CompressingMiddleware: error creating gzip decompression object: %s", err)
				resp.WriteHeader(http.StatusInternalServerError)
				return
			}

			req.Body = decompressor
			defer decompressor.Close()
		}

		handler.ServeHTTP(responseWriter, req)
	}

	return http.HandlerFunc(fn)
}
