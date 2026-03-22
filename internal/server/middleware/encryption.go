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

// Ключ заголовка с hmac подписью
const hmacHeaderKey = "HashSHA256"

// Middleware обёртка для запросов с hmac подписью
type hmacMiddleware struct {
	keyEncryption string // Ключ к подписи
}

// Создаёт новую middleware обёртку для запросов с hmac подписью
//
//	@param key ключ к подписи
//	@returns *hmacMiddleware middleware обёртка для запросов с hmac подписью
func NewHMACMiddleware(key string) hmacMiddleware {
	return hmacMiddleware{keyEncryption: key}
}

// Middleware функция-обёртка для запросов с hmac подписью
//
//	@param handler обработчик запросов, который нужно обернуть
//	@returns http.Handler обёрнутый обработчик запросов
func (m *hmacMiddleware) F(handler http.Handler) http.Handler {
	fn := func(resp http.ResponseWriter, req *http.Request) {
		const prefix = "middleware.hmacMiddleware.F"

		requestHMACHex := req.Header.Get(hmacHeaderKey)
		if requestHMACHex == "" {
			handler.ServeHTTP(resp, req)
			return
		}

		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(resp, "middleware: LoggingMiddleware: failed to read body", http.StatusInternalServerError)
			return
		}

		req.Body.Close()
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		requestHMAC, err := hex.DecodeString(requestHMACHex)
		if err != nil {
			logger.LogS.Errorw(fmt.Sprintf("%s: failed decode hex hmac: %v", prefix, err), "hmac in request", requestHMACHex)
			http.Error(resp, fmt.Sprintf("failed decode hex hmac: %v", err), http.StatusBadRequest)
			return
		}

		if ok, err := encryption.EqualHMAC(bodyBytes, []byte(m.keyEncryption), requestHMAC, sha256.New); err != nil {
			logger.LogS.Errorf("%s: failed equal hmac hash: %v", prefix, err)
			resp.WriteHeader(http.StatusInternalServerError)
			return
		} else if !ok {
			http.Error(resp, "Invalid hmac for this body", http.StatusBadRequest)
			return
		}

		tmpResponse := NewDefaultResponseWriter()
		handler.ServeHTTP(&tmpResponse, req)

		encryptedBody, err := encryption.SignHMAC([]byte(tmpResponse.Body), []byte(m.keyEncryption), sha256.New)
		if err != nil {
			logger.LogS.Errorw(fmt.Sprintf("%s: failed hmac sign: %v", prefix, err), "body", tmpResponse.Body, "key", m.keyEncryption)
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}

		tmpResponse.Headers.Add(hmacHeaderKey, hex.EncodeToString(encryptedBody))

		for key, values := range tmpResponse.Header() {
			for _, value := range values {
				resp.Header().Add(key, value)
			}
		}

		if tmpResponse.Status != -1 {
			resp.WriteHeader(tmpResponse.Status)
		}

		resp.Write([]byte(tmpResponse.Body))
	}
	return http.HandlerFunc(fn)
}
