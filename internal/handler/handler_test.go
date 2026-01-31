package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/skayfish/metrics/internal/server/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Проверяет работу обработчика обновления метрики
func TestCreateHandlerUpdate(t *testing.T) {
	type want struct {
		status      int
		contentType string
		body        string
	}
	tests := []struct {
		testName   string
		storage    storage.MemStorage
		requestURL string
		want       want
	}{
		{
			testName:   "unknown type",
			storage:    storage.MemStorage{},
			requestURL: "/update/unknown/MetricName/15",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Unknown metric`s type \"unknown\" [counter, gauge]\n",
			},
		},
		{
			testName:   "unknown value type",
			storage:    storage.MemStorage{},
			requestURL: "/update/counter/MetricName/string",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Metric`s value must be int64\n",
			},
		},
		{
			testName:   "unknown value type",
			storage:    storage.MemStorage{},
			requestURL: "/update/gauge/MetricName/string",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Metric`s value must be float64\n",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			router := chi.NewRouter()
			router.Post("/update/{type}/{name}/{value}", CreateUpdateHandler(&tt.storage))
			server := httptest.NewServer(router)
			defer server.Close()

			request := resty.New().R()
			resp, err := request.Post(server.URL + tt.requestURL)
			require.NoError(t, err)

			assert.Equal(t, tt.want.status, resp.StatusCode())
			assert.Equal(t, tt.want.contentType, resp.Header().Get("Content-Type"))
			assert.Equal(t, tt.want.body, string(resp.Body()))
		})
	}
}
