package main

import (
	"testing"
	"time"

	"github.com/skayfish/metrics/internal/flags"
	"github.com/skayfish/metrics/internal/logger"
	"github.com/skayfish/metrics/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

// Проверяет парс флагов, указанных при запуске программы
func Test_parseConfig(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		config, err := parseConfig()
		require.NoError(t, err)
		assert.Equal(t, server.Config{
			Address: flags.NetAddress{
				Host: "localhost",
				Port: 8080,
			},
			LogLevel:        logger.Level(zapcore.InfoLevel),
			StoreInterval:   300 * time.Second,
			FileStoragePath: "",
			ToRestore:       false,
			DatabaseDSN:     nil,
			MigrationsPath:  "../../migrations",
		}, *config)
	})
}
