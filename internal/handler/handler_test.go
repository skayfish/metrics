package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
		testName    string
		storage     storage.MemStorage
		method      string
		contentType string
		requestURL  string
		want        want
	}{
		// Error: not POST method
		{
			testName:    "Error: not POST method",
			storage:     storage.MemStorage{},
			method:      "GET",
			contentType: "text/plain",
			requestURL:  "/counter/MetricName/15",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Method of request must be POST\n",
			},
		},

		// Error: unknown Content-Type
		{
			testName:    "Error: unknown Content-Type",
			storage:     storage.MemStorage{},
			method:      "POST",
			contentType: "application/json",
			requestURL:  "/counter/MetricName/15",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Content-Type must be text/plain\n",
			},
		},

		// Error: not expected url
		{
			testName:    "Error: not expected url",
			storage:     storage.MemStorage{},
			method:      "POST",
			contentType: "text/plain",
			requestURL:  "/",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Expected request url: \"/update/{metric type}/{metric name}/{metric value}\"\n",
			},
		},
		{
			testName:    "Error: not expected url",
			storage:     storage.MemStorage{},
			method:      "POST",
			contentType: "text/plain",
			requestURL:  "/counter/MetricName/15/notexpected",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Expected request url: \"/update/{metric type}/{metric name}/{metric value}\"\n",
			},
		},
		{
			testName:    "Error: not expected url",
			storage:     storage.MemStorage{},
			method:      "POST",
			contentType: "text/plain",
			requestURL:  "/counter/MetricName",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Expected request url: \"/update/{metric type}/{metric name}/{metric value}\"\n",
			},
		},

		// Error: unknown type of metric
		{
			testName:    "Error: unknown type of metric",
			storage:     storage.MemStorage{},
			method:      "POST",
			contentType: "text/plain",
			requestURL:  "/unknown/",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Unknown metric`s type \"unknown\" [counter, gauge]\n",
			},
		},
		{
			testName:    "Error: unknown type of metric",
			storage:     storage.MemStorage{},
			method:      "POST",
			contentType: "text/plain",
			requestURL:  "/unknown",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Unknown metric`s type \"unknown\" [counter, gauge]\n",
			},
		},
		{
			testName:    "Error: unknown type of metric",
			storage:     storage.MemStorage{},
			method:      "POST",
			contentType: "text/plain",
			requestURL:  "/unknown/MetricName/15",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Unknown metric`s type \"unknown\" [counter, gauge]\n",
			},
		},

		// Error: not found name of metric in url
		{
			testName:    "Error: not found name of metric in url",
			storage:     storage.MemStorage{},
			method:      "POST",
			contentType: "text/plain",
			requestURL:  "/counter",
			want: want{
				status:      http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "Metric`s name not found in url\n",
			},
		},
		{
			testName:    "Error: not found name of metric in url",
			storage:     storage.MemStorage{},
			method:      "POST",
			contentType: "text/plain",
			requestURL:  "/counter/",
			want: want{
				status:      http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "Metric`s name not found in url\n",
			},
		},

		// Error: error type of metric value
		{
			testName:    "Error: error type of metric value [int64]",
			storage:     storage.MemStorage{},
			method:      "POST",
			contentType: "text/plain",
			requestURL:  "/counter/MetricName/",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Metric`s value must be int64\n",
			},
		},
		{
			testName:    "Error: error type of metric value [int64]",
			storage:     storage.MemStorage{},
			method:      "POST",
			contentType: "text/plain",
			requestURL:  "/counter/MetricName/string",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Metric`s value must be int64\n",
			},
		},
		{
			testName:    "Error: error type of metric value [float64]",
			storage:     storage.MemStorage{},
			method:      "POST",
			contentType: "text/plain",
			requestURL:  "/gauge/MetricName/",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Metric`s value must be float64\n",
			},
		},
		{
			testName:    "Error: error type of metric value [float64]",
			storage:     storage.MemStorage{},
			method:      "POST",
			contentType: "text/plain",
			requestURL:  "/gauge/MetricName/string",
			want: want{
				status:      http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "Metric`s value must be float64\n",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			emptyBody := ""
			request := httptest.NewRequest(tt.method, tt.requestURL, strings.NewReader(emptyBody))
			request.Header.Add("Content-Type", tt.contentType)
			recorder := httptest.NewRecorder()

			handler := CreateHandlerUpdate(&tt.storage)
			handler(recorder, request)

			res := recorder.Result()
			assert.Equal(t, tt.want.status, res.StatusCode)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))

			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)

			require.NoError(t, err)
			assert.Equal(t, tt.want.body, string(resBody))
		})
	}
}
