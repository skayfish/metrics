package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"

	"github.com/skayfish/metrics/internal/encryption"
	"github.com/skayfish/metrics/internal/logger"
)

// SF TODO
type hmacResponseWriter struct {
	headers http.Header // SF TODO
	status  int         // SF TODO
	body    string      // SF TODO
}

// SF TODO
func newHMACResponseWriter() hmacResponseWriter {
	return hmacResponseWriter{
		headers: make(http.Header),
		status:  -1,
	}
}

// SF TODO
func (rw *hmacResponseWriter) Header() http.Header {
	return rw.headers
}

// SF TODO
func (rw *hmacResponseWriter) Write(data []byte) (int, error) {
	rw.body += string(data)
	return len(data), nil
}

func (rw *hmacResponseWriter) WriteHeader(statusCode int) {
	rw.status = statusCode
}

// SF TODO
type hmacMiddleware struct {
	keyEncryption *string // SF TODO
}

// SF TODO
func NewHMACMiddleware(key *string) (*hmacMiddleware, error) {
	const prefix = "middleware.NewHMACMiddleware"

	if key == nil {
		return nil, fmt.Errorf("%s: key encryption is nil", prefix)
	}

	return &hmacMiddleware{keyEncryption: key}, nil
}

// SF TODO
func (m *hmacMiddleware) F(handler http.Handler) http.Handler {
	fn := func(resp http.ResponseWriter, req *http.Request) {
		const prefix = "middleware.hmacMiddleware.f"

		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(resp, "middleware: LoggingMiddleware: failed to read body", http.StatusInternalServerError)
			return
		}

		defer req.Body.Close()
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		requestHMACHex := req.Header.Get("HashSHA256")
		requestHMAC, err := hex.DecodeString(requestHMACHex)
		if err != nil {
			logger.LogS.Errorw(fmt.Sprintf("%s: failed decode hex hmac: %v", prefix, err), "hmac", requestHMACHex)
			http.Error(resp, fmt.Sprintf("failed decode hex hmac: %v", err), http.StatusBadRequest)
			return
		}

		if ok, err := encryption.EqualHMAC(bodyBytes, []byte(*m.keyEncryption), requestHMAC, sha256.New); err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			return
		} else if !ok {
			http.Error(resp, "unexpected hmac", http.StatusBadRequest)
			return
		}

		tmpResponse := newHMACResponseWriter()
		handler.ServeHTTP(&tmpResponse, req)

		encryptedBody, err := encryption.SignHMAC([]byte(tmpResponse.body), []byte(*m.keyEncryption), sha256.New)
		if err != nil {
			logger.LogS.Errorw(fmt.Sprintf("%s: failed hmac sign: %v", prefix, err), "body", tmpResponse.body, "key", m.keyEncryption)
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}

		tmpResponse.headers.Add("HashSHA256", hex.EncodeToString(encryptedBody))

		for key, values := range tmpResponse.Header() {
			for _, value := range values {
				resp.Header().Add(key, value)
			}
		}

		if tmpResponse.status != -1 {
			resp.WriteHeader(tmpResponse.status)
		}

		resp.Write([]byte(tmpResponse.body))
	}
	return http.HandlerFunc(fn)
}
