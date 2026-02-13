package logger_test

import (
	"testing"

	"github.com/skayfish/metrics/internal/logger"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

// Проверяет инициализацию менеджера логирования
func TestInit(t *testing.T) {
	tests := []struct {
		testName string
		level    logger.Level
	}{
		{
			testName: "DebugLevel",
			level:    logger.Level(zapcore.DebugLevel),
		},
		{
			testName: "InfoLevel",
			level:    logger.Level(zapcore.InfoLevel),
		},
		{
			testName: "WarnLevel",
			level:    logger.Level(zapcore.WarnLevel),
		},
		{
			testName: "ErrorLevel",
			level:    logger.Level(zapcore.ErrorLevel),
		},
		{
			testName: "DPanicLevel",
			level:    logger.Level(zapcore.DPanicLevel),
		},
		{
			testName: "PanicLevel",
			level:    logger.Level(zapcore.PanicLevel),
		},
		{
			testName: "FatalLevel",
			level:    logger.Level(zapcore.FatalLevel),
		},
		{
			testName: "Invalid",
			level:    logger.Level(zapcore.InvalidLevel),
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			err := logger.Init(tt.level)
			require.NoError(t, err)
		})
	}
}
