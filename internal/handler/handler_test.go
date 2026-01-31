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
func TestCreateUpdateHandler(t *testing.T) {
	type want struct {
		status      int
		contentType string
		body        string
	}
	tests := []struct {
		testName       string
		gaugeMetrics   map[string]float64
		counterMetrics map[string]int64
		requestURL     string
		want           want
	}{
		{
			testName:   "unknown type",
			requestURL: "/update/unknown/MetricName/15",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Unknown metric`s type \"unknown\" [counter, gauge]\n",
			},
		},
		{
			testName:   "unknown value type",
			requestURL: "/update/counter/MetricName/string",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Metric`s value must be int64\n",
			},
		},
		{
			testName:   "unknown value type",
			requestURL: "/update/gauge/MetricName/string",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Metric`s value must be float64\n",
			},
		},
		{
			testName:       "update gauge metric",
			gaugeMetrics:   map[string]float64{"MetricName": -43.12257, "MetricName1": 413.127},
			counterMetrics: map[string]int64{"MetricName": 4312, "MetricName1": -4312, "MetricName2": 12},
			requestURL:     "/update/gauge/MetricName/0.233000024133",
			want: want{
				status:      http.StatusOK,
				contentType: "",
				body:        "",
			},
		},
		{
			testName:       "update counter metric",
			gaugeMetrics:   map[string]float64{"MetricName": -43.12257, "MetricName1": 413.127},
			counterMetrics: map[string]int64{"MetricName": 4312, "MetricName1": -4312, "MetricName2": 12},
			requestURL:     "/update/counter/MetricName/11",
			want: want{
				status:      http.StatusOK,
				contentType: "",
				body:        "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			storage := storage.NewMemStorage()
			for name, value := range tt.counterMetrics {
				storage.UpdateCounter(name, value)
			}

			for name, value := range tt.gaugeMetrics {
				storage.UpdateGauge(name, value)
			}

			router := chi.NewRouter()
			router.Post("/update/{type}/{name}/{value}", CreateUpdateHandler(&storage))
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

// Проверяет работу обработчика получения конкретной метрики
func TestCreateGetValueHandler(t *testing.T) {
	type want struct {
		status      int
		contentType string
		body        string
	}
	tests := []struct {
		testName       string
		gaugeMetrics   map[string]float64
		counterMetrics map[string]int64
		requestURL     string
		want           want
	}{
		{
			testName:   "unknown type",
			requestURL: "/value/unknown/MetricName",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Unknown metric`s type \"unknown\" [counter, gauge]\n",
			},
		},
		{
			testName:   "not found gauge metric",
			requestURL: "/value/gauge/MetricName",
			want: want{
				status:      http.StatusNotFound,
				contentType: "",
				body:        "",
			},
		},
		{
			testName:   "not found counter metric",
			requestURL: "/value/counter/MetricName",
			want: want{
				status:      http.StatusNotFound,
				contentType: "",
				body:        "",
			},
		},
		{
			testName:     "found gauge metric",
			gaugeMetrics: map[string]float64{"MetricName": -43.12257, "MetricName1": 413.127},
			requestURL:   "/value/gauge/MetricName",
			want: want{
				status:      http.StatusOK,
				contentType: "text/plain; charset=utf-8",
				body:        "-43.12257",
			},
		},
		{
			testName:       "found counter metric",
			counterMetrics: map[string]int64{"MetricName": 4312, "MetricName1": -4312, "MetricName2": 12},
			requestURL:     "/value/counter/MetricName",
			want: want{
				status:      http.StatusOK,
				contentType: "text/plain; charset=utf-8",
				body:        "4312",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			storage := storage.NewMemStorage()
			for name, value := range tt.counterMetrics {
				storage.UpdateCounter(name, value)
			}

			for name, value := range tt.gaugeMetrics {
				storage.UpdateGauge(name, value)
			}

			router := chi.NewRouter()
			router.Get("/value/{type}/{name}", CreateGetValueHandler(&storage))
			server := httptest.NewServer(router)
			defer server.Close()

			request := resty.New().R()
			resp, err := request.Get(server.URL + tt.requestURL)
			require.NoError(t, err)

			assert.Equal(t, tt.want.status, resp.StatusCode())
			assert.Equal(t, tt.want.contentType, resp.Header().Get("Content-Type"))
			assert.Equal(t, tt.want.body, string(resp.Body()))
		})
	}
}
