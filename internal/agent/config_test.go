package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Проверяет возвращаемый тип соединения
func TestConfig_getConnectionType(t *testing.T) {
	tests := []struct {
		testName string
		config   Config
		want     string
	}{
		{
			"http",
			Config{SecureConnection: false},
			"http",
		},
		{
			"http",
			Config{
				SecureConnection: false,
				Host:             "host",
				Port:             43143,
				RetryMaxWaitTime: 2 * time.Second,
				RetryWaitTime:    2 * time.Second,
				PollInterval:     2 * time.Second,
				ReportInterval:   2 * time.Second,
			},
			"http",
		},
		{
			"https",
			Config{SecureConnection: true},
			"https",
		},
		{
			"https",
			Config{
				SecureConnection: true,
				Host:             "host",
				Port:             43143,
				RetryMaxWaitTime: 2 * time.Second,
				RetryWaitTime:    2 * time.Second,
				PollInterval:     2 * time.Second,
				ReportInterval:   2 * time.Second,
			},
			"https",
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.config.getConnectionType())
		})
	}
}
