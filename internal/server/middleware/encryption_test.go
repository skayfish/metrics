package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/skayfish/metrics/internal/encryption"
	"github.com/skayfish/metrics/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Проверяет создание middleware обёртки для запросов с hmac подписью
func TestNewHMACMiddleware(t *testing.T) {
	key := "some key"
	mid := NewHMACMiddleware(key)
	require.NotNil(t, mid)
	require.NotNil(t, mid.keyEncryption)
	assert.Equal(t, key, mid.keyEncryption)
}

// Проверяет вызов middleware обёртки для запросов с hmac подписью
func Test_hmacMiddleware_F(t *testing.T) {
	checkRequest := func(r *http.Request, expected string) {
		data, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		r.Body.Close()
		r.Body = io.NopCloser(bytes.NewBuffer(data))

		assert.Equal(t, expected, string(data))
	}

	key := "some key"
	tests := []struct {
		test string

		hmac            string
		body            string
		wantHandlerUsed bool
		bodyResult      string
		statusResult    int
	}{
		{
			test:            "no hmac",
			body:            "",
			wantHandlerUsed: true,
			bodyResult:      "",
			statusResult:    http.StatusOK,
		},
		{
			test:            "no hmac",
			body:            "some body",
			wantHandlerUsed: true,
			bodyResult:      "",
			statusResult:    http.StatusOK,
		},
		{
			test:            "no hmac",
			body:            "some body",
			wantHandlerUsed: true,
			bodyResult:      "some body result",
			statusResult:    http.StatusOK,
		},

		{
			test:            "empty bodies",
			hmac:            "70d19601c3b9123566d9f6cec756de0d8fb3e57d2474dd82c3005edee7106ad7",
			body:            "",
			wantHandlerUsed: true,
			bodyResult:      "",
			statusResult:    http.StatusOK,
		},
		{
			test:            "empty result body",
			hmac:            "2a2629ba328d5376b44f88536047a12500d33bc43045a7407c29a88312bc2a48",
			body:            "some body",
			wantHandlerUsed: true,
			bodyResult:      "",
			statusResult:    http.StatusOK,
		},
		{
			test:            "empty body",
			hmac:            "70d19601c3b9123566d9f6cec756de0d8fb3e57d2474dd82c3005edee7106ad7",
			body:            "",
			wantHandlerUsed: true,
			bodyResult:      "some body",
			statusResult:    http.StatusOK,
		},

		{
			test:            "invalid hmac",
			hmac:            "invalid hmac",
			body:            "",
			wantHandlerUsed: false,
			bodyResult:      "failed decode hex hmac",
			statusResult:    http.StatusBadRequest,
		},
		{
			test:            "invalid hmac",
			hmac:            "invalid hmac",
			body:            "some body",
			wantHandlerUsed: false,
			bodyResult:      "failed decode hex hmac",
			statusResult:    http.StatusBadRequest,
		},

		{
			test:            "hmac not for body",
			hmac:            "70d19601c3b9123566d9f6cec756de0d8fb3e57d2474dd82c3005edee7106ad7",
			body:            "hmac is invalid for this body",
			wantHandlerUsed: false,
			bodyResult:      "Invalid hmac for this body",
			statusResult:    http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.test, func(t *testing.T) {
			mid := NewHMACMiddleware(key)
			handlerUsed := false

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerUsed = true
				checkRequest(r, tt.body)
				w.Header().Set("Content-Type", "text/plain")
				w.Write([]byte(tt.bodyResult))
			})

			request, err := http.NewRequest("POST", "host:port", strings.NewReader(tt.body))
			request.Header.Add(hmacHeaderKey, tt.hmac)
			require.NoError(t, err)
			response := newDefaultResponseWriter()
			mid.F(handler).ServeHTTP(&response, request)

			checkRequest(request, tt.body)
			if tt.hmac == "" {
				assert.True(t, handlerUsed)
				assert.Equal(t, tt.bodyResult, response.body)
				assert.Equal(t, tt.statusResult, response.status)
				testutil.HeadersEqual(t, map[string][]string{"Content-Type": {"text/plain"}}, response.Header())
			} else if tt.wantHandlerUsed {
				assert.True(t, handlerUsed)
				assert.Equal(t, tt.bodyResult, response.body)
				assert.Equal(t, tt.statusResult, response.status)
				hmacResult, err := encryption.SignHMAC([]byte(tt.bodyResult), []byte(key), sha256.New)
				require.NoError(t, err)
				assert.Equal(t, hex.EncodeToString(hmacResult), response.headers.Get(hmacHeaderKey))
				expectedHeaders := map[string][]string{
					"Content-Type": {"text/plain"},
					"Hashsha256":   {hex.EncodeToString(hmacResult)},
				}
				testutil.HeadersEqual(t, expectedHeaders, response.Header())
				t.Logf("\n%+v\n%+v", expectedHeaders, response.Header())
			} else {
				assert.False(t, handlerUsed)
				assert.Contains(t, response.body, tt.bodyResult)
				assert.Equal(t, tt.statusResult, response.status)
				assert.Empty(t, response.Header().Get(hmacHeaderKey))
			}
		})
	}
}
