package agent

import (
	"net/http"
	"reflect"
	"runtime"
	"testing"
	"time"
)

// Проверяет создание менеджера отправки метрик серверу
func TestNewSender(t *testing.T) {
	type args struct {
		config Configuration
	}
	tests := []struct {
		name string
		args args
		want sender
	}{
		{
			"Success",
			args{
				config: Configuration{
					SecureConnection: false,
					Host:             "localhost",
					Port:             "8080",
					PollInterval:     10 * time.Second,
					ReportInterval:   0,
				},
			},
			sender{
				config: Configuration{
					SecureConnection: false,
					Host:             "localhost",
					Port:             "8080",
					PollInterval:     10 * time.Second,
					ReportInterval:   0,
				},
				pollCount: 0,
				totalTime: 0,
				client:    http.Client{},
			},
		},
		{
			"Success",
			args{
				config: Configuration{
					SecureConnection: true,
					Host:             "localhost2",
					Port:             "5050",
					PollInterval:     1000 * time.Minute,
					ReportInterval:   99 * time.Nanosecond,
				},
			},
			sender{
				config: Configuration{
					SecureConnection: true,
					Host:             "localhost2",
					Port:             "5050",
					PollInterval:     1000 * time.Minute,
					ReportInterval:   99 * time.Nanosecond,
				},
				pollCount: 0,
				totalTime: 0,
				client:    http.Client{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewSender(tt.args.config); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSender() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Проверяет фильтрацию метрик
func Test_sender_filtrate(t *testing.T) {
	type fields struct {
		config    Configuration
		pollCount int64
		totalTime time.Duration
		client    http.Client
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
			"Success",
			fields{
				config:    Configuration{},
				pollCount: 0,
				totalTime: 0,
				client:    http.Client{},
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
			"Success",
			fields{
				config: Configuration{
					SecureConnection: false,
					Host:             "localhost",
					Port:             "8080",
					PollInterval:     10 * time.Second,
					ReportInterval:   0,
				},
				pollCount: 10,
				totalTime: 11,
				client:    http.Client{},
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
				totalTime: tt.fields.totalTime,
				client:    tt.fields.client,
			}
			if gotRes := obj.filtrate(tt.args.metrics); !reflect.DeepEqual(gotRes, tt.wantRes) {
				t.Errorf("sender.filtrate() = %v, want %v", gotRes, tt.wantRes)
			}
		})
	}
}
