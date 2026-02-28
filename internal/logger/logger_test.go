package logger

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

// Проверяет инициализацию менеджера логирования
func TestInit(t *testing.T) {
	tests := []struct {
		testName string
		level    Level
	}{
		{
			testName: "DebugLevel",
			level:    Level(zapcore.DebugLevel),
		},
		{
			testName: "InfoLevel",
			level:    Level(zapcore.InfoLevel),
		},
		{
			testName: "WarnLevel",
			level:    Level(zapcore.WarnLevel),
		},
		{
			testName: "ErrorLevel",
			level:    Level(zapcore.ErrorLevel),
		},
		{
			testName: "DPanicLevel",
			level:    Level(zapcore.DPanicLevel),
		},
		{
			testName: "PanicLevel",
			level:    Level(zapcore.PanicLevel),
		},
		{
			testName: "FatalLevel",
			level:    Level(zapcore.FatalLevel),
		},
		{
			testName: "Invalid",
			level:    Level(zapcore.InvalidLevel),
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			err := Init(tt.level)
			require.NoError(t, err)
		})
	}
}
