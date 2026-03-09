package main

import (
	"testing"
	"time"

	"github.com/skayfish/metrics/internal/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SF TODO
func Test_parseConfig(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		config, err := parseConfig()
		require.NoError(t, err)
		assert.Equal(t, agent.Config{
			SecureConnection: false,
			Host:             "localhost",
			Port:             8080,
			RetryMaxWaitTime: 10 * time.Second,
			RetryWaitTime:    2 * time.Second,
			PollInterval:     2 * time.Second,
			ReportInterval:   10 * time.Second,
		}, *config)
	})
}
