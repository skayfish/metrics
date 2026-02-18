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

// SF TODO
type compressWriter struct {
	// SF TODO
	response http.ResponseWriter

	// SF TODO
	compressor *gzip.Writer

	//SF TODO
	body string
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
	cw.body += string(data)

	contentType := cw.response.Header().Get("Content-Type")
	if contentType == "application/json" || strings.HasPrefix(contentType, "text/html") {
		cw.response.Header().Add("Content-Encoding", "gzip")
		return cw.compressor.Write(data)
	}

	cw.compressor.Reset(bytes.NewBuffer(nil))

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
