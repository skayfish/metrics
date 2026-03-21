package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/skayfish/metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Таймаут ожидания подключения к серверу
const retryMaxWaitTime = 10 * time.Second

// Частота попыток подключения к серверу (например, раз в 2 секунды)
const retryWaitTime time.Duration = 2 * time.Second

// Проверяет создание менеджера отправки метрик серверу
func TestNewSender(t *testing.T) {
	type args struct {
		config Config
	}
	tests := []struct {
		name string
		args args
		want sender
	}{
		{
			"success",
			args{
				config: Config{
					SecureConnection: false,
					Host:             "localhost",
					Port:             8080,
					RetryMaxWaitTime: retryMaxWaitTime,
					RetryWaitTime:    retryWaitTime,
					PollInterval:     10 * time.Second,
					ReportInterval:   0,
				},
			},
			sender{
				config: Config{
					SecureConnection: false,
					Host:             "localhost",
					Port:             8080,
					RetryMaxWaitTime: retryMaxWaitTime,
					RetryWaitTime:    retryWaitTime,
					PollInterval:     10 * time.Second,
					ReportInterval:   0,
				},
				client: nil,
			},
		},
		{
			"success",
			args{
				config: Config{
					SecureConnection: true,
					Host:             "localhost2",
					Port:             5050,
					RetryMaxWaitTime: retryMaxWaitTime,
					RetryWaitTime:    retryWaitTime,
					PollInterval:     1000 * time.Minute,
					ReportInterval:   99 * time.Nanosecond,
				},
			},
			sender{
				config: Config{
					SecureConnection: true,
					Host:             "localhost2",
					Port:             5050,
					RetryMaxWaitTime: retryMaxWaitTime,
					RetryWaitTime:    retryWaitTime,
					PollInterval:     1000 * time.Minute,
					ReportInterval:   99 * time.Nanosecond,
				},
				client: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSender(tt.args.config)
			got.client = nil
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSender() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Проверяет фильтрацию метрик
func Test_filtrate(t *testing.T) {
	tests := []struct {
		name    string
		metrics runtime.MemStats
		wantRes map[string]float64
	}{
		{
			name: "success",
			metrics: runtime.MemStats{
				Alloc:         0,
				TotalAlloc:    0,
				Sys:           0,
				Lookups:       0,
				Mallocs:       0,
				Frees:         0,
				HeapAlloc:     0,
				HeapSys:       0,
				HeapIdle:      0,
				HeapInuse:     0,
				HeapReleased:  0,
				HeapObjects:   0,
				StackInuse:    0,
				StackSys:      0,
				MSpanInuse:    0,
				MSpanSys:      0,
				MCacheInuse:   0,
				MCacheSys:     0,
				BuckHashSys:   0,
				GCSys:         0,
				OtherSys:      0,
				NextGC:        0,
				LastGC:        0,
				PauseTotalNs:  0,
				PauseNs:       [256]uint64{},
				PauseEnd:      [256]uint64{},
				NumGC:         0,
				NumForcedGC:   0,
				GCCPUFraction: 0,
				EnableGC:      false,
				DebugGC:       false,
			},
			wantRes: map[string]float64{
				"Alloc":         0,
				"TotalAlloc":    0,
				"Sys":           0,
				"Lookups":       0,
				"Mallocs":       0,
				"Frees":         0,
				"HeapAlloc":     0,
				"HeapSys":       0,
				"HeapIdle":      0,
				"HeapInuse":     0,
				"HeapReleased":  0,
				"HeapObjects":   0,
				"StackInuse":    0,
				"StackSys":      0,
				"MSpanInuse":    0,
				"MSpanSys":      0,
				"MCacheInuse":   0,
				"MCacheSys":     0,
				"BuckHashSys":   0,
				"GCSys":         0,
				"OtherSys":      0,
				"NextGC":        0,
				"LastGC":        0,
				"PauseTotalNs":  0,
				"NumGC":         0,
				"NumForcedGC":   0,
				"GCCPUFraction": 0,
			},
		},
		{
			name: "success",
			metrics: runtime.MemStats{
				Alloc:         5,
				TotalAlloc:    9,
				Sys:           4,
				Lookups:       3,
				Mallocs:       0,
				Frees:         0,
				HeapAlloc:     4,
				HeapSys:       0,
				HeapIdle:      1,
				HeapInuse:     0,
				HeapReleased:  0,
				HeapObjects:   7,
				StackInuse:    0,
				StackSys:      0,
				MSpanInuse:    6,
				MSpanSys:      0,
				MCacheInuse:   6,
				MCacheSys:     0,
				BuckHashSys:   0,
				GCSys:         9,
				OtherSys:      4,
				NextGC:        2,
				LastGC:        3,
				PauseTotalNs:  1,
				PauseNs:       [256]uint64{},
				PauseEnd:      [256]uint64{},
				NumGC:         2,
				NumForcedGC:   3,
				GCCPUFraction: 5,
				EnableGC:      false,
				DebugGC:       false,
			},
			wantRes: map[string]float64{
				"Alloc":         5,
				"TotalAlloc":    9,
				"Sys":           4,
				"Lookups":       3,
				"Mallocs":       0,
				"Frees":         0,
				"HeapAlloc":     4,
				"HeapSys":       0,
				"HeapIdle":      1,
				"HeapInuse":     0,
				"HeapReleased":  0,
				"HeapObjects":   7,
				"StackInuse":    0,
				"StackSys":      0,
				"MSpanInuse":    6,
				"MSpanSys":      0,
				"MCacheInuse":   6,
				"MCacheSys":     0,
				"BuckHashSys":   0,
				"GCSys":         9,
				"OtherSys":      4,
				"NextGC":        2,
				"LastGC":        3,
				"PauseTotalNs":  1,
				"NumGC":         2,
				"NumForcedGC":   3,
				"GCCPUFraction": 5,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotRes := filtrate(&tt.metrics); !reflect.DeepEqual(gotRes, tt.wantRes) {
				t.Errorf("sender.filtrate() = %v, want %v", gotRes, tt.wantRes)
			}
		})
	}
}

// Проверяет запуск менеджера отправки метрик серверу
func Test_sender_Run(t *testing.T) {
	t.Run("correct poll counting", func(t *testing.T) {
		router := chi.NewRouter()
		handlerCounter := 0
		gaugeCounter := 0
		router.Post("/updates", func(resp http.ResponseWriter, req *http.Request) {
			require.Equal(t, "application/json", req.Header.Get("Content-Type"))
			require.Equal(t, "gzip", req.Header.Get("Content-Encoding"))

			decompressor, err := gzip.NewReader(req.Body)
			require.NoError(t, err)
			defer decompressor.Close()

			var buf bytes.Buffer
			_, err = buf.ReadFrom(decompressor)
			require.NoError(t, err)

			metrics := []model.Metrics{}
			require.NoError(t, json.NewDecoder(&buf).Decode(&metrics))

			for _, metric := range metrics {
				if metric.MType == model.Counter && metric.ID == "PollCount" {
					require.NotNil(t, metric.Delta)
					switch {
					case handlerCounter == 0:
						assert.Equal(t, int64(1), *metric.Delta)
						fmt.Print("Handler count 1: succeed\n")
					case handlerCounter == 1:
						assert.Equal(t, int64(4), *metric.Delta)
						fmt.Printf("Handler count %d: succeed\n", handlerCounter+1)
					case handlerCounter == 2:
						assert.Equal(t, int64(5), *metric.Delta)
						fmt.Printf("Handler count %d: succeed\n", handlerCounter+1)
					default:
						t.Errorf("expected handler call count = 3, actual = %d", handlerCounter+1)
					}

					handlerCounter++
				} else if metric.MType == model.Gauge {
					require.NotNil(t, metric.Value)
					gaugeCounter++
				} else {
					t.Errorf("unknown metric type: %s", metric.MType)
				}
			}
		})

		server := httptest.NewServer(router)
		defer server.Close()

		hostPort := strings.Split(server.URL[7:], ":")
		port, err := strconv.Atoi(string(hostPort[1]))
		require.NoError(t, err)

		sender := sender{
			config: Config{
				SecureConnection: false,
				Host:             string(hostPort[0]),
				Port:             port,
				RetryMaxWaitTime: retryMaxWaitTime,
				RetryWaitTime:    retryWaitTime,
				PollInterval:     100 * time.Millisecond,
				ReportInterval:   500 * time.Millisecond,
			},
			client: resty.New(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
		defer cancel()
		err = sender.Run(ctx)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Equal(t, 3, handlerCounter)
		assert.Equal(t, 28*handlerCounter, gaugeCounter)
	})
}
