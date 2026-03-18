package middleware

import "net/http"

// SF TODO
type defaultResponseWriter struct {
	headers http.Header // SF TODO
	status  int         // SF TODO
	body    string      // SF TODO
}

// SF TODO
func newDefaultResponseWriter() defaultResponseWriter {
	return defaultResponseWriter{
		headers: make(http.Header),
		status:  -1,
	}
}

// SF TODO
func (rw *defaultResponseWriter) Header() http.Header {
	return rw.headers
}

// SF TODO
func (rw *defaultResponseWriter) Write(data []byte) (int, error) {
	rw.body += string(data)
	if rw.status == -1 {
		rw.status = http.StatusOK
	}

	return len(data), nil
}

func (rw *defaultResponseWriter) WriteHeader(statusCode int) {
	rw.status = statusCode
}
