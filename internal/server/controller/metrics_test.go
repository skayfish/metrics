package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/skayfish/metrics/internal/logger"
	"github.com/skayfish/metrics/internal/model"
	"github.com/skayfish/metrics/internal/server/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Заполняет хранилище данных метриками
func updateMetrics(t *testing.T, storage storage.Storage, gaugeMetrics map[string]float64, counterMetrics map[string]int64) {
	for name, value := range counterMetrics {
		_, err := storage.Update(model.Metrics{ID: name, Delta: &value, MType: model.Counter})
		require.NoError(t, err)
	}

	for name, value := range gaugeMetrics {
		_, err := storage.Update(model.Metrics{ID: name, Value: &value, MType: model.Gauge})
		require.NoError(t, err)
	}
}

// Устанавливает уровень логирования для всего тестирования.
// @warning всегда возвращать через defer setLogLevel(t, "info")
func setLogLevel(t *testing.T, level string) {
	var logLevel logger.Level
	err := logLevel.Set(level)
	require.NoError(t, err)
	logger.Init(logLevel)
}

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
				body:        "Unknown metric`s type \"unknown\" [counter, gauge]",
			},
		},
		{
			testName:   "unknown value type",
			requestURL: "/update/counter/MetricName/string",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Metric`s value must be int64",
			},
		},
		{
			testName:   "unknown value type",
			requestURL: "/update/gauge/MetricName/string",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Metric`s value must be float64",
			},
		},
		{
			testName:       "update gauge metric",
			gaugeMetrics:   map[string]float64{"GaugeMetricName": -43.12257, "GaugeMetricName1": 413.127},
			counterMetrics: map[string]int64{"CounterMetricName": 4312, "CounterMetricName1": -4312, "CounterMetricName2": 12},
			requestURL:     "/update/gauge/GaugeMetricName/0.233000024133",
			want: want{
				status:      http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"GaugeMetricName","type":"gauge","value":0.233000024133}`,
			},
		},
		{
			testName:       "update counter metric",
			gaugeMetrics:   map[string]float64{"GaugeMetricName": -43.12257, "GaugeMetricName1": 413.127},
			counterMetrics: map[string]int64{"CounterMetricName": 4312, "CounterMetricName1": -4312, "CounterMetricName2": 12},
			requestURL:     "/update/counter/CounterMetricName/11",
			want: want{
				status:      http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"CounterMetricName","type":"counter","delta":4323}`,
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
				body:        "found not \"gauge\" metric type",
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
				body:        "found not \"counter\" metric type",
			},
		},
	}
	for _, logLevel := range []string{"debug", "info"} {
		setLogLevel(t, logLevel)
		for _, tt := range tests {
			t.Run(tt.testName, func(t *testing.T) {
				storage := storage.NewMemStorage()
				controller, err := NewMetricsController(&storage)
				require.NoError(t, err)

				updateMetrics(t, &storage, tt.gaugeMetrics, tt.counterMetrics)

				router := chi.NewRouter()
				router.Post("/update/{type}/{name}/{value}", controller.UpdateFromURL)
				server := httptest.NewServer(router)
				defer server.Close()

				request := resty.New().R()
				resp, err := request.Post(server.URL + tt.requestURL)
				require.NoError(t, err)

				assert.Equal(t, tt.want.status, resp.StatusCode(), resp.String())
				assert.Equal(t, tt.want.contentType, resp.Header().Get("Content-Type"), resp.String())
				switch tt.want.contentType {
				case "text/plain; charset=utf-8":
					assert.Equal(t, tt.want.body, resp.String())
				case "application/json":
					expectedMetric := model.Metrics{}
					buf := bytes.NewBuffer([]byte(tt.want.body))
					require.NoError(t, json.NewDecoder(buf).Decode(&expectedMetric))
					expectedMetricJSON, err := json.Marshal(expectedMetric)
					require.NoError(t, err)
					assert.Equal(t, string(expectedMetricJSON), resp.String())
				default:
					t.Error("Unexpected content type", tt.want.contentType)
				}
			})
		}
		setLogLevel(t, "info")
	}
}

// Проверяет работу обработчика запроса на обновление метрики, переданной в формате JSON
func TestMetricsController_UpdateFromJSON(t *testing.T) {
	type want struct {
		status      int
		contentType string
		body        string
	}
	tests := []struct {
		testName           string
		gaugeMetrics       map[string]float64
		counterMetrics     map[string]int64
		requestURL         string
		requestBody        string
		requestContentType string
		want               want
	}{
		{
			testName:           "expected json in request",
			requestURL:         "/update/",
			requestBody:        `{"id":"MetricName", "type":"unknown"}`,
			requestContentType: `text/plain`,
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Expected application/json content type",
			},
		},
		{
			testName:           "invalid json",
			requestURL:         "/update/",
			requestBody:        `{`,
			requestContentType: `application/json`,
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Failed unmarshall json: unexpected EOF",
			},
		},
		{
			testName:           "unrecognized type",
			requestURL:         "/update/",
			requestBody:        `{"id":"MetricName", "type":"unknown"}`,
			requestContentType: `application/json`,
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        `unrecognized metric type (supported types: "gauge", "counter"): id="MetricName"`,
			},
		},
		{
			testName:           "update gauge metric",
			gaugeMetrics:       map[string]float64{"GaugeMetricName": -43.12257, "GaugeMetricName1": 413.127},
			counterMetrics:     map[string]int64{"CounterMetricName": 4312, "CounterMetricName1": -4312, "CounterMetricName2": 12},
			requestURL:         "/update",
			requestBody:        `{"id":"GaugeMetricName", "type":"gauge", "value": 0.233000024133}`,
			requestContentType: `application/json`,
			want: want{
				status:      http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"GaugeMetricName","type":"gauge","value":0.233000024133}`,
			},
		},
		{
			testName:           "update counter metric",
			gaugeMetrics:       map[string]float64{"GaugeMetricName": -43.12257, "GaugeMetricName1": 413.127},
			counterMetrics:     map[string]int64{"CounterMetricName": 4312, "CounterMetricName1": -4312, "CounterMetricName2": 12},
			requestURL:         "/update/",
			requestBody:        `{"id":"CounterMetricName", "type":"counter", "delta": 11}`,
			requestContentType: `application/json`,
			want: want{
				status:      http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"CounterMetricName","type":"counter","delta":4323}`,
			},
		},
		{
			testName:           "found not gauge metric type",
			gaugeMetrics:       map[string]float64{"GaugeMetricName": -43.12257, "GaugeMetricName1": 413.127},
			counterMetrics:     map[string]int64{"CounterMetricName": 4312, "CounterMetricName1": -4312, "CounterMetricName2": 12},
			requestURL:         "/update",
			requestBody:        `{"id":"CounterMetricName", "type":"gauge", "value": 0.233000024133}`,
			requestContentType: `application/json`,
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        `found not "gauge" metric type`,
			},
		},
		{
			testName:           "found not counter metric type",
			gaugeMetrics:       map[string]float64{"GaugeMetricName": -43.12257, "GaugeMetricName1": 413.127},
			counterMetrics:     map[string]int64{"CounterMetricName": 4312, "CounterMetricName1": -4312, "CounterMetricName2": 12},
			requestURL:         "/update",
			requestBody:        `{"id":"GaugeMetricName", "type":"counter", "delta": 11}`,
			requestContentType: `application/json`,
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        `found not "counter" metric type`,
			},
		},
	}
	for _, logLevel := range []string{"debug", "info"} {
		setLogLevel(t, logLevel)
		for _, tt := range tests {
			t.Run(tt.testName, func(t *testing.T) {
				storage := storage.NewMemStorage()
				controller, err := NewMetricsController(&storage)
				require.NoError(t, err)

				updateMetrics(t, &storage, tt.gaugeMetrics, tt.counterMetrics)

				router := chi.NewRouter()
				router.Post("/update/", controller.UpdateFromJSON)
				router.Post("/update", controller.UpdateFromJSON)
				server := httptest.NewServer(router)
				defer server.Close()

				resp, err := resty.New().R().
					SetBody(tt.requestBody).
					SetHeader("Content-Type", tt.requestContentType).
					SetHeader("Accept", "application/json").
					Post(server.URL + tt.requestURL)
				require.NoError(t, err)

				assert.Equal(t, tt.want.status, resp.StatusCode())
				require.Equal(t, tt.want.contentType, resp.Header().Get("Content-Type"))

				switch tt.want.contentType {
				case "text/plain; charset=utf-8":
					assert.Equal(t, tt.want.body, resp.String())
				case "application/json":
					expectedMetric := model.Metrics{}
					buf := bytes.NewBuffer([]byte(tt.want.body))
					require.NoError(t, json.NewDecoder(buf).Decode(&expectedMetric))
					expectedMetricJSON, err := json.Marshal(expectedMetric)
					require.NoError(t, err)
					assert.Equal(t, string(expectedMetricJSON), resp.String())
				default:
					t.Error("Unexpected content type", tt.want.contentType)
				}
			})
		}
		setLogLevel(t, "info")
	}
}

// SF TODO
func TestMetricsController_Updates(t *testing.T) {
	type want struct {
		status      int
		contentType string
		body        string
	}
	tests := []struct {
		testName           string
		gaugeMetrics       map[string]float64
		counterMetrics     map[string]int64
		requestURL         string
		requestBody        string
		requestContentType string
		want               want
	}{
		{
			testName:           "expected json in request",
			requestURL:         "/updates/",
			requestBody:        `[{"id":"MetricName", "type":"unknown"}]`,
			requestContentType: `text/plain`,
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Expected application/json content type",
			},
		},
		{
			testName:           "invalid json",
			requestURL:         "/updates/",
			requestBody:        `{"id":"MetricName", "type":"counter"}`,
			requestContentType: `application/json`,
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Failed unmarshall json: json: cannot unmarshal object into Go value of type []model.Metrics",
			},
		},
		{
			testName:           "invalid json",
			requestURL:         "/updates/",
			requestBody:        `[`,
			requestContentType: `application/json`,
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Failed unmarshall json: unexpected EOF",
			},
		},
		{
			testName:           "unrecognized type",
			requestURL:         "/updates/",
			requestBody:        `[{"id":"MetricName", "type":"unknown"}]`,
			requestContentType: `application/json`,
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        `unrecognized metric type (supported types: "gauge", "counter"): id="MetricName"`,
			},
		},
		{
			testName:           "gauge value is empty",
			requestURL:         "/updates/",
			requestBody:        `[{"id":"MetricName", "type":"gauge"}]`,
			requestContentType: `application/json`,
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        `gauge metric value is empty: id="MetricName"`,
			},
		},
		{
			testName:           "counter delta is empty",
			requestURL:         "/updates/",
			requestBody:        `[{"id":"MetricName", "type":"counter"}]`,
			requestContentType: `application/json`,
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        `counter metric delta is empty: id="MetricName"`,
			},
		},
		{
			testName:           "update gauge metric",
			gaugeMetrics:       map[string]float64{"GaugeMetricName": -43.12257, "GaugeMetricName1": 413.127},
			counterMetrics:     map[string]int64{"CounterMetricName": 4312, "CounterMetricName1": -4312, "CounterMetricName2": 12},
			requestURL:         "/updates",
			requestBody:        `[{"id":"GaugeMetricName", "type":"gauge", "value": 0.233000024133}]`,
			requestContentType: `application/json`,
			want: want{
				status:      http.StatusOK,
				contentType: "application/json",
				body:        `[{"id":"GaugeMetricName","type":"gauge","value":0.233000024133}]`,
			},
		},
		{
			testName:           "update counter metric",
			gaugeMetrics:       map[string]float64{"GaugeMetricName": -43.12257, "GaugeMetricName1": 413.127},
			counterMetrics:     map[string]int64{"CounterMetricName": 4312, "CounterMetricName1": -4312, "CounterMetricName2": 12},
			requestURL:         "/updates/",
			requestBody:        `[{"id":"CounterMetricName", "type":"counter", "delta": 11}]`,
			requestContentType: `application/json`,
			want: want{
				status:      http.StatusOK,
				contentType: "application/json",
				body:        `[{"id":"CounterMetricName","type":"counter","delta":4323}]`,
			},
		},
		{
			testName:           "found not gauge metric type",
			gaugeMetrics:       map[string]float64{"GaugeMetricName": -43.12257, "GaugeMetricName1": 413.127},
			counterMetrics:     map[string]int64{"CounterMetricName": 4312, "CounterMetricName1": -4312, "CounterMetricName2": 12},
			requestURL:         "/updates",
			requestBody:        `[{"id":"CounterMetricName", "type":"gauge", "value": 0.233000024133}]`,
			requestContentType: `application/json`,
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        `found not "gauge" metric type`,
			},
		},
		{
			testName:           "found not counter metric type",
			gaugeMetrics:       map[string]float64{"GaugeMetricName": -43.12257, "GaugeMetricName1": 413.127},
			counterMetrics:     map[string]int64{"CounterMetricName": 4312, "CounterMetricName1": -4312, "CounterMetricName2": 12},
			requestURL:         "/updates",
			requestBody:        `[{"id":"GaugeMetricName", "type":"counter", "delta": 11}]`,
			requestContentType: `application/json`,
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        `found not "counter" metric type`,
			},
		},
	}
	for _, logLevel := range []string{"debug", "info"} {
		setLogLevel(t, logLevel)
		for _, tt := range tests {
			t.Run(tt.testName, func(t *testing.T) {
				storage := storage.NewMemStorage()
				controller, err := NewMetricsController(&storage)
				require.NoError(t, err)

				updateMetrics(t, &storage, tt.gaugeMetrics, tt.counterMetrics)

				router := chi.NewRouter()
				router.Post("/updates/", controller.Updates)
				router.Post("/updates", controller.Updates)
				server := httptest.NewServer(router)
				defer server.Close()

				resp, err := resty.New().R().
					SetBody(tt.requestBody).
					SetHeader("Content-Type", tt.requestContentType).
					SetHeader("Accept", "application/json").
					Post(server.URL + tt.requestURL)
				require.NoError(t, err)

				assert.Equal(t, tt.want.status, resp.StatusCode())
				require.Equal(t, tt.want.contentType, resp.Header().Get("Content-Type"))

				switch tt.want.contentType {
				case "text/plain; charset=utf-8":
					assert.Equal(t, tt.want.body, resp.String())
				case "application/json":
					expectedMetrics := []model.Metrics{}
					buf := bytes.NewBuffer([]byte(tt.want.body))
					require.NoError(t, json.NewDecoder(buf).Decode(&expectedMetrics))
					expectedMetricsJSON, err := json.Marshal(expectedMetrics)
					require.NoError(t, err)
					assert.Equal(t, string(expectedMetricsJSON), resp.String())
				default:
					t.Error("Unexpected content type", tt.want.contentType)
				}
			})
		}
		setLogLevel(t, "info")
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
				body:        "Unknown metric`s type \"unknown\" [counter, gauge]",
			},
		},
		{
			testName:   "not found gauge metric",
			requestURL: "/value/gauge/MetricName",
			want: want{
				status:      http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "Metric with id \"MetricName\", type \"gauge\" not found",
			},
		},
		{
			testName:   "not found counter metric",
			requestURL: "/value/counter/MetricName",
			want: want{
				status:      http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "Metric with id \"MetricName\", type \"counter\" not found",
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
				body:        "found not \"gauge\" metric type",
			},
		},
		{
			testName:     "found not counter metric type",
			gaugeMetrics: map[string]float64{"MetricName": -43.12257, "MetricName1": 413.127},
			requestURL:   "/value/counter/MetricName",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "found not \"counter\" metric type",
			},
		},
	}
	for _, logLevel := range []string{"debug", "info"} {
		setLogLevel(t, logLevel)
		for _, tt := range tests {
			t.Run(tt.testName, func(t *testing.T) {
				storage := storage.NewMemStorage()
				controller, err := NewMetricsController(&storage)
				require.NoError(t, err)

				updateMetrics(t, &storage, tt.gaugeMetrics, tt.counterMetrics)

				router := chi.NewRouter()
				router.Get("/value/{type}/{name}", controller.GetValueFromURL)
				server := httptest.NewServer(router)
				defer server.Close()

				request := resty.New().R()
				resp, err := request.Get(server.URL + tt.requestURL)
				require.NoError(t, err)

				assert.Equal(t, tt.want.status, resp.StatusCode())
				assert.Equal(t, tt.want.contentType, resp.Header().Get("Content-Type"))
				assert.Equal(t, tt.want.body, resp.String())
			})
		}
		setLogLevel(t, "info")
	}
}

// Проверяет работу обработчика получения конкретной метрики через json
func TestMetricsController_GetMetricFromJSON(t *testing.T) {
	type want struct {
		status      int
		contentType string
		body        string
	}
	tests := []struct {
		testName           string
		gaugeMetrics       map[string]float64
		counterMetrics     map[string]int64
		requestURL         string
		requestBody        string
		requestContentType string
		want               want
	}{
		{
			testName:           "expected json in request",
			requestURL:         "/value/",
			requestBody:        `{"id":"MetricName", "type":"unknown"}`,
			requestContentType: `text/plain`,
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Expected application/json content type",
			},
		},
		{
			testName:           "unknown type",
			requestURL:         "/value/",
			requestBody:        `{"id":"MetricName", "type":"unknown"}`,
			requestContentType: `application/json`,
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Unknown metric`s type \"unknown\" [counter, gauge]",
			},
		},
		{
			testName:           "not found gauge metric",
			requestURL:         "/value/",
			requestBody:        `{"id":"MetricName", "type":"gauge"}`,
			requestContentType: `application/json`,
			want: want{
				status:      http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "Metric with id \"MetricName\", type \"gauge\" not found",
			},
		},
		{
			testName:           "not found counter metric",
			requestURL:         "/value/",
			requestBody:        `{"id":"MetricName", "type":"counter"}`,
			requestContentType: `application/json`,
			want: want{
				status:      http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "Metric with id \"MetricName\", type \"counter\" not found",
			},
		},
		{
			testName:           "found gauge metric",
			gaugeMetrics:       map[string]float64{"MetricName": -43.12257, "MetricName1": 413.127},
			requestURL:         "/value/",
			requestBody:        `{"id":"MetricName", "type":"gauge"}`,
			requestContentType: `application/json`,
			want: want{
				status:      http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"MetricName", "type":"gauge", "value":-43.12257}`,
			},
		},
		{
			testName:           "found counter metric",
			counterMetrics:     map[string]int64{"MetricName": 4312, "MetricName1": -4312, "MetricName2": 12},
			requestURL:         "/value/",
			requestBody:        `{"id":"MetricName", "type":"counter"}`,
			requestContentType: `application/json`,
			want: want{
				status:      http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"MetricName", "type":"counter", "delta":4312}`,
			},
		},
		{
			testName:           "found not gauge metric type",
			counterMetrics:     map[string]int64{"MetricName": 4312},
			requestURL:         "/value/",
			requestBody:        `{"id":"MetricName", "type":"gauge"}`,
			requestContentType: `application/json`,
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "found not \"gauge\" metric type",
			},
		},
		{
			testName:           "found not counter metric type",
			gaugeMetrics:       map[string]float64{"MetricName": -43.12257, "MetricName1": 413.127},
			requestURL:         "/value/",
			requestBody:        `{"id":"MetricName", "type":"counter"}`,
			requestContentType: `application/json`,
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "found not \"counter\" metric type",
			},
		},
	}
	for _, logLevel := range []string{"debug", "info"} {
		setLogLevel(t, logLevel)
		for _, tt := range tests {
			t.Run(tt.testName, func(t *testing.T) {
				storage := storage.NewMemStorage()
				controller, err := NewMetricsController(&storage)
				require.NoError(t, err)

				updateMetrics(t, &storage, tt.gaugeMetrics, tt.counterMetrics)

				router := chi.NewRouter()
				router.Post("/value/", controller.GetMetricFromJSON)
				router.Post("/value", controller.GetMetricFromJSON)
				server := httptest.NewServer(router)
				defer server.Close()

				resp, err := resty.New().R().
					SetBody(tt.requestBody).
					SetHeader("Content-Type", tt.requestContentType).
					SetHeader("Accept", "application/json").
					Post(server.URL + tt.requestURL)
				require.NoError(t, err)

				assert.Equal(t, tt.want.status, resp.StatusCode(), resp.String())
				require.Equal(t, tt.want.contentType, resp.Header().Get("Content-Type"), resp.String())

				switch tt.want.contentType {
				case "text/plain; charset=utf-8":
					assert.Equal(t, tt.want.body, resp.String())
				case "application/json":
					expectedMetric := model.Metrics{}
					buf := bytes.NewBuffer([]byte(tt.want.body))
					require.NoError(t, json.NewDecoder(buf).Decode(&expectedMetric))
					expectedMetricJSON, err := json.MarshalIndent(expectedMetric, "", "    ")
					require.NoError(t, err)
					assert.Equal(t, string(expectedMetricJSON), resp.String())
				default:
					t.Error("Unexpected content type", tt.want.contentType)
				}
			})
		}
		setLogLevel(t, "info")
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
	for _, logLevel := range []string{"debug", "info"} {
		setLogLevel(t, logLevel)
		for _, tt := range tests {
			t.Run(tt.testName, func(t *testing.T) {
				storage := storage.NewMemStorage()
				controller, err := NewMetricsController(&storage)
				require.NoError(t, err)

				updateMetrics(t, &storage, tt.gaugeMetrics, tt.counterMetrics)

				router := chi.NewRouter()
				router.Get("/", controller.GetAllMetrics)
				server := httptest.NewServer(router)
				defer server.Close()

				request := resty.New().R()
				resp, err := request.Get(server.URL + tt.requestURL)
				require.NoError(t, err)

				assert.Equal(t, tt.want.status, resp.StatusCode())
				assert.Equal(t, tt.want.contentType, resp.Header().Get("Content-Type"))
				assert.False(t, len(resp.Body()) == 0, resp.String())
			})
		}
		setLogLevel(t, "info")
	}
}
