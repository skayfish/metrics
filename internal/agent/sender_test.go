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
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/skayfish/metrics/internal/model"
	"github.com/skayfish/metrics/internal/server/middleware"
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
func Test_filtrateMS(t *testing.T) {
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
			if gotRes := filtrateMS(&tt.metrics); !reflect.DeepEqual(gotRes, tt.wantRes) {
				t.Errorf("sender.filtrate() = %v, want %v", gotRes, tt.wantRes)
			}
		})
	}
}

// Проверяет запуск менеджера отправки метрик серверу
func Test_sender_Run(t *testing.T) {
	t.Run("metrics counting", func(t *testing.T) {
		router := chi.NewRouter()
		var handlerCounter atomic.Uint32
		var gaugeCounter atomic.Uint32
		var countCounter atomic.Uint32
		router.Post("/updates", func(resp http.ResponseWriter, req *http.Request) {
			curHandlerCounter := handlerCounter.Add(1)

			fmt.Printf("Handler %d: started\n", curHandlerCounter)

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
					curCountCounter := countCounter.Add(1)

					require.NotNil(t, metric.Delta)
					switch {
					case curHandlerCounter < 3:
						assert.Equal(t, int64(1), *metric.Delta)
					case curHandlerCounter < 5:
						assert.Equal(t, int64(5), *metric.Delta)
					default:
						t.Errorf("expected count metric times: 3, actual: %d", curCountCounter)
					}

				} else if metric.MType == model.Gauge {
					require.NotNil(t, metric.Value)
					gaugeCounter.Add(1)
				} else {
					t.Errorf("unknown metric type: %s", metric.MType)
				}
			}

			fmt.Printf("Handler %d: finished\n", curHandlerCounter)
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
				PollInterval:     99 * time.Millisecond,
				ReportInterval:   500 * time.Millisecond,
				RateLimit:        10,
			},
			client: resty.New(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
		defer cancel()
		err = sender.Run(ctx)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Equal(t, uint32(4), handlerCounter.Load())
		assert.Equal(t, uint32((28+3)*3), gaugeCounter.Load()) // (28:mem stats + 3:system stats) * 3:times
	})

	rateLimitTests := []struct {
		test                 string
		rateLimit            uint
		expectedHandlerCount uint32
	}{
		{
			test:                 "low rate limit",
			rateLimit:            1,
			expectedHandlerCount: 3, // ms, ss, ms+ss
		},
		{
			test:                 "good rate limit",
			rateLimit:            5,
			expectedHandlerCount: 12, // ms, ss, (ms+ss)*10
		},
	}
	for _, tt := range rateLimitTests {
		t.Run(tt.test, func(t *testing.T) {
			router := chi.NewRouter()
			var handlerCounter atomic.Uint32
			router.Post("/updates", func(resp http.ResponseWriter, req *http.Request) {
				hc := handlerCounter.Add(1)
				fmt.Printf("handler %d: started\n", hc)
				time.Sleep(time.Second)
				fmt.Printf("handler %d: finished\n", hc)
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
					PollInterval:     99 * time.Millisecond,
					ReportInterval:   200 * time.Millisecond,
					RateLimit:        tt.rateLimit,
				},
				client: resty.New(),
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2050*time.Millisecond)
			defer cancel()
			err = sender.Run(ctx)
			require.ErrorIs(t, err, context.DeadlineExceeded)
			assert.Equal(t, tt.expectedHandlerCount, handlerCounter.Load())
		})
	}

	t.Run("sign hmac", func(t *testing.T) {
		hmacKey := "some key"

		router := chi.NewRouter()
		mid := middleware.NewHMACMiddleware(hmacKey)
		h := mid.F(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			// do nothing
		})).(http.HandlerFunc)
		router.Post("/updates", func(resp http.ResponseWriter, req *http.Request) {
			require.NotEmpty(t, req.Header.Get("HashSHA256"), req.Header)

			rw := middleware.NewDefaultResponseWriter()
			h.ServeHTTP(&rw, req)

			assert.Equal(t, http.StatusOK, rw.Status)
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
				PollInterval:     99 * time.Millisecond,
				ReportInterval:   200 * time.Millisecond,
				RateLimit:        1,
				KeyEncryption:    &hmacKey,
			},
			client: resty.New(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		err = sender.Run(ctx)
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})
}
