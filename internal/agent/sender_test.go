package agent

import (
	"net/http"
	"reflect"
	"runtime"
	"testing"
	"time"
)

func TestNewSender(t *testing.T) {
	type args struct {
		serverAddress  string
		pollInterval   time.Duration
		reportInterval time.Duration
	}
	tests := []struct {
		name string
		args args
		want sender
	}{
		{
			"Success",
			args{
				serverAddress:  "http://localhost:8080",
				pollInterval:   10 * time.Second,
				reportInterval: 0,
			},
			sender{
				serverAddress:  "http://localhost:8080",
				pollInterval:   10 * time.Second,
				reportInterval: 0,
				pollCount:      0,
				totalTime:      0,
				client:         http.Client{},
			},
		},
		{
			"Success",
			args{
				serverAddress:  "localhost",
				pollInterval:   1000 * time.Minute,
				reportInterval: 99 * time.Nanosecond,
			},
			sender{
				serverAddress:  "localhost",
				pollInterval:   1000 * time.Minute,
				reportInterval: 99 * time.Nanosecond,
				pollCount:      0,
				totalTime:      0,
				client:         http.Client{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSender(tt.args.serverAddress, tt.args.pollInterval, tt.args.reportInterval)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSender() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sender_filtrate(t *testing.T) {
	type fields struct {
		serverAddress  string
		pollInterval   time.Duration
		reportInterval time.Duration
		pollCount      int64
		totalTime      time.Duration
		client         http.Client
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
				serverAddress:  "http://localhost:8080",
				pollInterval:   0,
				reportInterval: 0,
				pollCount:      0,
				totalTime:      0,
				client:         http.Client{},
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
				serverAddress:  "http://localhost:8080",
				pollInterval:   10,
				reportInterval: 10,
				pollCount:      10,
				totalTime:      11,
				client:         http.Client{},
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
				serverAddress:  tt.fields.serverAddress,
				pollInterval:   tt.fields.pollInterval,
				reportInterval: tt.fields.reportInterval,
				pollCount:      tt.fields.pollCount,
				totalTime:      tt.fields.totalTime,
				client:         tt.fields.client,
			}
			if gotRes := obj.filtrate(tt.args.metrics); !reflect.DeepEqual(gotRes, tt.wantRes) {
				t.Errorf("sender.filtrate() = %v, want %v", gotRes, tt.wantRes)
			}
		})
	}
}

func Test_sender_Run(t *testing.T) {
	type fields struct {
		serverAddress  string
		pollInterval   time.Duration
		reportInterval time.Duration
		pollCount      int64
		totalTime      time.Duration
		client         http.Client
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &sender{
				serverAddress:  tt.fields.serverAddress,
				pollInterval:   tt.fields.pollInterval,
				reportInterval: tt.fields.reportInterval,
				pollCount:      tt.fields.pollCount,
				totalTime:      tt.fields.totalTime,
				client:         tt.fields.client,
			}
			if err := obj.Run(); (err != nil) != tt.wantErr {
				t.Errorf("sender.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_sender_send(t *testing.T) {
	type fields struct {
		serverAddress  string
		pollInterval   time.Duration
		reportInterval time.Duration
		pollCount      int64
		totalTime      time.Duration
		client         http.Client
	}
	type args struct {
		gaugeMetrics map[string]float64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &sender{
				serverAddress:  tt.fields.serverAddress,
				pollInterval:   tt.fields.pollInterval,
				reportInterval: tt.fields.reportInterval,
				pollCount:      tt.fields.pollCount,
				totalTime:      tt.fields.totalTime,
				client:         tt.fields.client,
			}
			if err := obj.send(tt.args.gaugeMetrics); (err != nil) != tt.wantErr {
				t.Errorf("sender.send() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_sender_sendGaugeMetric(t *testing.T) {
	type fields struct {
		serverAddress  string
		pollInterval   time.Duration
		reportInterval time.Duration
		pollCount      int64
		totalTime      time.Duration
		client         http.Client
	}
	type args struct {
		name  string
		value float64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &sender{
				serverAddress:  tt.fields.serverAddress,
				pollInterval:   tt.fields.pollInterval,
				reportInterval: tt.fields.reportInterval,
				pollCount:      tt.fields.pollCount,
				totalTime:      tt.fields.totalTime,
				client:         tt.fields.client,
			}
			if err := obj.sendGaugeMetric(tt.args.name, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("sender.sendGaugeMetric() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_sender_sendGaugeMetrics(t *testing.T) {
	type fields struct {
		serverAddress  string
		pollInterval   time.Duration
		reportInterval time.Duration
		pollCount      int64
		totalTime      time.Duration
		client         http.Client
	}
	type args struct {
		metrics map[string]float64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &sender{
				serverAddress:  tt.fields.serverAddress,
				pollInterval:   tt.fields.pollInterval,
				reportInterval: tt.fields.reportInterval,
				pollCount:      tt.fields.pollCount,
				totalTime:      tt.fields.totalTime,
				client:         tt.fields.client,
			}
			if err := obj.sendGaugeMetrics(tt.args.metrics); (err != nil) != tt.wantErr {
				t.Errorf("sender.sendGaugeMetrics() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_sender_sendCounterMetric(t *testing.T) {
	type fields struct {
		serverAddress  string
		pollInterval   time.Duration
		reportInterval time.Duration
		pollCount      int64
		totalTime      time.Duration
		client         http.Client
	}
	type args struct {
		name  string
		value int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &sender{
				serverAddress:  tt.fields.serverAddress,
				pollInterval:   tt.fields.pollInterval,
				reportInterval: tt.fields.reportInterval,
				pollCount:      tt.fields.pollCount,
				totalTime:      tt.fields.totalTime,
				client:         tt.fields.client,
			}
			if err := obj.sendCounterMetric(tt.args.name, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("sender.sendCounterMetric() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_sender_sendCounterMetrics(t *testing.T) {
	type fields struct {
		serverAddress  string
		pollInterval   time.Duration
		reportInterval time.Duration
		pollCount      int64
		totalTime      time.Duration
		client         http.Client
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &sender{
				serverAddress:  tt.fields.serverAddress,
				pollInterval:   tt.fields.pollInterval,
				reportInterval: tt.fields.reportInterval,
				pollCount:      tt.fields.pollCount,
				totalTime:      tt.fields.totalTime,
				client:         tt.fields.client,
			}
			if err := obj.sendCounterMetrics(); (err != nil) != tt.wantErr {
				t.Errorf("sender.sendCounterMetrics() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
