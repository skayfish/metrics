package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/skayfish/metrics/internal/model"
	"github.com/skayfish/metrics/internal/server/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Проверяет работу обработчика обновления метрики через URL
func TestMetricsController_UpdateFromURL(t *testing.T) {
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
			gaugeMetrics:   map[string]float64{"GaugeMetricName": -43.12257, "GaugeMetricName1": 413.127},
			counterMetrics: map[string]int64{"CounterMetricName": 4312, "CounterMetricName1": -4312, "CounterMetricName2": 12},
			requestURL:     "/update/gauge/GaugeMetricName/0.233000024133",
			want: want{
				status:      http.StatusOK,
				contentType: "",
				body:        "",
			},
		},
		{
			testName:       "update counter metric",
			gaugeMetrics:   map[string]float64{"GaugeMetricName": -43.12257, "GaugeMetricName1": 413.127},
			counterMetrics: map[string]int64{"CounterMetricName": 4312, "CounterMetricName1": -4312, "CounterMetricName2": 12},
			requestURL:     "/update/counter/CounterMetricName/11",
			want: want{
				status:      http.StatusOK,
				contentType: "",
				body:        "",
			},
		},
		{
			testName:       "found not gauge metric type",
			gaugeMetrics:   map[string]float64{"GaugeMetricName": -43.12257, "GaugeMetricName1": 413.127},
			counterMetrics: map[string]int64{"CounterMetricName": 4312, "CounterMetricName1": -4312, "CounterMetricName2": 12},
			requestURL:     "/update/gauge/CounterMetricName/0.233000024133",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "found not \"gauge\" metric type\n",
			},
		},
		{
			testName:       "found not counter metric type",
			gaugeMetrics:   map[string]float64{"GaugeMetricName": -43.12257, "GaugeMetricName1": 413.127},
			counterMetrics: map[string]int64{"CounterMetricName": 4312, "CounterMetricName1": -4312, "CounterMetricName2": 12},
			requestURL:     "/update/counter/GaugeMetricName/11",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "found not \"counter\" metric type\n",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			storage := storage.NewMemStorage()
			controller, err := NewMetricsController(&storage)
			require.NoError(t, err)

			for name, value := range tt.counterMetrics {
				storage.UpdateCounter(name, value)
			}

			for name, value := range tt.gaugeMetrics {
				storage.UpdateGauge(name, value)
			}

			router := chi.NewRouter()
			router.Post("/update/{type}/{name}/{value}", controller.UpdateFromURL)
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

// Проверяет работу обработчика получения конкретной метрики через URL
func TestMetricsController_GetValueFromURL(t *testing.T) {
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
				contentType: "text/plain; charset=utf-8",
				body:        "Metric with id \"MetricName\", type \"gauge\" not found\n",
			},
		},
		{
			testName:   "not found counter metric",
			requestURL: "/value/counter/MetricName",
			want: want{
				status:      http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "Metric with id \"MetricName\", type \"counter\" not found\n",
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
		{
			testName:       "found not gauge metric type",
			counterMetrics: map[string]int64{"MetricName": 4312},
			requestURL:     "/value/gauge/MetricName",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "found not \"gauge\" metric type\n",
			},
		},
		{
			testName:     "found not counter metric type",
			gaugeMetrics: map[string]float64{"MetricName": -43.12257, "MetricName1": 413.127},
			requestURL:   "/value/counter/MetricName",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "found not \"counter\" metric type\n",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			storage := storage.NewMemStorage()
			controller, err := NewMetricsController(&storage)
			require.NoError(t, err)

			for name, value := range tt.counterMetrics {
				storage.UpdateCounter(name, value)
			}

			for name, value := range tt.gaugeMetrics {
				storage.UpdateGauge(name, value)
			}

			router := chi.NewRouter()
			router.Get("/value/{type}/{name}", controller.GetValueFromURL)
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

// Проверяет работу обработчика получения конкретной метрики через JSON
func TestMetricsController_GetValueFromJSON(t *testing.T) {
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
		requestBody    string
		want           want
	}{
		{
			testName:    "unknown type",
			requestURL:  "/value/",
			requestBody: `{"id":"MetricName", "type":"unknown"}`,
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Unknown metric`s type \"unknown\" [counter, gauge]\n",
			},
		},
		{
			testName:    "not found gauge metric",
			requestURL:  "/value/",
			requestBody: `{"id":"MetricName", "type":"gauge"}`,
			want: want{
				status:      http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "Metric with id \"MetricName\", type \"gauge\" not found\n",
			},
		},
		{
			testName:    "not found counter metric",
			requestURL:  "/value/",
			requestBody: `{"id":"MetricName", "type":"counter"}`,
			want: want{
				status:      http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "Metric with id \"MetricName\", type \"counter\" not found\n",
			},
		},
		{
			testName:     "found gauge metric",
			gaugeMetrics: map[string]float64{"MetricName": -43.12257, "MetricName1": 413.127},
			requestURL:   "/value/",
			requestBody:  `{"id":"MetricName", "type":"gauge"}`,
			want: want{
				status:      http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"MetricName", "type":"gauge", "value":-43.12257}`,
			},
		},
		{
			testName:       "found counter metric",
			counterMetrics: map[string]int64{"MetricName": 4312, "MetricName1": -4312, "MetricName2": 12},
			requestURL:     "/value/",
			requestBody:    `{"id":"MetricName", "type":"counter"}`,
			want: want{
				status:      http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"MetricName", "type":"counter", "delta":4312}`,
			},
		},
		{
			testName:       "found not gauge metric type",
			counterMetrics: map[string]int64{"MetricName": 4312},
			requestURL:     "/value/",
			requestBody:    `{"id":"MetricName", "type":"gauge"}`,
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "found not \"gauge\" metric type\n",
			},
		},
		{
			testName:     "found not counter metric type",
			gaugeMetrics: map[string]float64{"MetricName": -43.12257, "MetricName1": 413.127},
			requestURL:   "/value/",
			requestBody:  `{"id":"MetricName", "type":"counter"}`,
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "found not \"counter\" metric type\n",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			storage := storage.NewMemStorage()
			controller, err := NewMetricsController(&storage)
			require.NoError(t, err)

			for name, value := range tt.counterMetrics {
				storage.UpdateCounter(name, value)
			}

			for name, value := range tt.gaugeMetrics {
				storage.UpdateGauge(name, value)
			}

			router := chi.NewRouter()
			router.Post("/value/", controller.GetValueFromJSON)
			router.Post("/value", controller.GetValueFromJSON)
			server := httptest.NewServer(router)
			defer server.Close()

			resp, err := resty.New().R().
				SetBody(tt.requestBody).
				SetHeader("Content-Type", "application/json").
				SetHeader("Accept", "application/json").
				Post(server.URL + tt.requestURL)
			require.NoError(t, err)

			assert.Equal(t, tt.want.status, resp.StatusCode())
			require.Equal(t, tt.want.contentType, resp.Header().Get("Content-Type"))

			switch tt.want.contentType {
			case "text/plain; charset=utf-8":
				assert.Equal(t, tt.want.body, string(resp.Body()))
			case "application/json":
				expectedMetric := model.Metrics{}
				buf := bytes.NewBuffer([]byte(tt.want.body))
				require.NoError(t, json.NewDecoder(buf).Decode(&expectedMetric))
				expectedMetricJSON, err := json.MarshalIndent(expectedMetric, "", "    ")
				require.NoError(t, err)
				assert.Equal(t, string(expectedMetricJSON), string(resp.Body()))
			default:
				t.Error("Unexpected content type", tt.want.contentType)
			}
		})
	}
}

// Проверяет работу обработчика получения всех метрик
func TestMetricsController_GetAllMetrics(t *testing.T) {
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
			testName:   "no metrics",
			requestURL: "/",
			want: want{
				status:      http.StatusOK,
				contentType: "text/html; charset=UTF-8",
			},
		},
		{
			testName:       "many metrics",
			gaugeMetrics:   map[string]float64{"GaugeMetricName": -43.12257, "GaugeMetricName1": 413.127},
			counterMetrics: map[string]int64{"MetricName": 4312, "MetricName1": -4312, "MetricName2": 12},
			requestURL:     "/",
			want: want{
				status:      http.StatusOK,
				contentType: "text/html; charset=UTF-8",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			storage := storage.NewMemStorage()
			controller, err := NewMetricsController(&storage)
			require.NoError(t, err)

			for name, value := range tt.counterMetrics {
				storage.UpdateCounter(name, value)
			}

			for name, value := range tt.gaugeMetrics {
				storage.UpdateGauge(name, value)
			}

			router := chi.NewRouter()
			router.Get("/", controller.GetAllMetrics)
			server := httptest.NewServer(router)
			defer server.Close()

			request := resty.New().R()
			resp, err := request.Get(server.URL + tt.requestURL)
			require.NoError(t, err)

			assert.Equal(t, tt.want.status, resp.StatusCode())
			assert.Equal(t, tt.want.contentType, resp.Header().Get("Content-Type"))
			assert.False(t, len(resp.Body()) == 0, string(resp.Body()))
		})
	}
}
