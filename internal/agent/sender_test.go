package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
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
				pollCount: 0,
				client:    nil,
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
				pollCount: 0,
				client:    nil,
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
func Test_sender_filtrate(t *testing.T) {
	type fields struct {
		config    Config
		pollCount int64
		totalTime time.Duration
		client    *resty.Client
	}
	type args struct {
		metrics runtime.MemStats
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantRes map[string]float64
	}{
		{
			"success",
			fields{
				config:    Config{},
				pollCount: 0,
				client:    resty.New(),
			},
			args{
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
			},
			map[string]float64{
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
			"success",
			fields{
				config: Config{
					SecureConnection: false,
					Host:             "localhost",
					Port:             8080,
					RetryMaxWaitTime: retryMaxWaitTime,
					RetryWaitTime:    retryWaitTime,
					PollInterval:     10 * time.Second,
					ReportInterval:   0,
				},
				pollCount: 10,
				client:    resty.New(),
			},
			args{
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
			},
			map[string]float64{
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
			obj := &sender{
				config:    tt.fields.config,
				pollCount: tt.fields.pollCount,
				client:    tt.fields.client,
			}
			if gotRes := obj.filtrate(tt.args.metrics); !reflect.DeepEqual(gotRes, tt.wantRes) {
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
		router.Post("/update/{type}/{name}/{value}", func(resp http.ResponseWriter, req *http.Request) {
			mType := chi.URLParam(req, "type")
			mName := chi.URLParam(req, "name")
			mValue := chi.URLParam(req, "value")

			if mType == model.Counter && mName == "PollCount" {
				value, err := strconv.ParseInt(mValue, 10, 64)
				require.NoError(t, err)
				switch {
				case handlerCounter == 0:
					assert.Equal(t, int64(1), value)
					fmt.Print("Handler count 0 succeed\n")
				case handlerCounter < 3:
					assert.Equal(t, int64(5), value)
					fmt.Printf("Handler count %d succeed\n", handlerCounter)
				default:
					t.Errorf("expected handler call count = 3, actual = %d", handlerCounter+1)
				}

				handlerCounter++
			} else if mType == model.Gauge {
				gaugeCounter++
			}
		})
		router.Post("/update", func(resp http.ResponseWriter, req *http.Request) {
			require.Equal(t, "application/json", req.Header.Get("Content-Type"))

			decompressor, err := gzip.NewReader(req.Body)
			require.NoError(t, err)
			defer decompressor.Close()

			var buf bytes.Buffer
			_, err = buf.ReadFrom(decompressor)
			require.NoError(t, err)

			metric := model.Metrics{}
			require.NoError(t, json.NewDecoder(&buf).Decode(&metric))

			if metric.MType == model.Counter && metric.ID == "PollCount" {
				require.NotNil(t, metric.Delta)
				switch {
				case handlerCounter == 0:
					assert.Equal(t, int64(1), *metric.Delta)
					fmt.Print("Handler count 0 succeed\n")
				case handlerCounter < 3:
					assert.Equal(t, int64(5), *metric.Delta)
					fmt.Printf("Handler count %d succeed\n", handlerCounter)
				default:
					t.Errorf("expected handler call count = 3, actual = %d", handlerCounter+1)
				}

				handlerCounter++
			} else if metric.MType == model.Gauge {
				gaugeCounter++
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
				PollInterval:     10 * time.Millisecond,
				ReportInterval:   50 * time.Millisecond,
			},
			pollCount: 0,
			client:    resty.New(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
		defer cancel()
		err = sender.Run(ctx)
		require.Equal(t, context.DeadlineExceeded, errors.Unwrap(err))
		assert.Equal(t, 3, handlerCounter)
		assert.Equal(t, 28*handlerCounter, gaugeCounter)
	})
}
