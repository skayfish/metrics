package logger_test

import (
	"testing"

	"github.com/skayfish/metrics/internal/logger"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// SF TODO
func TestInit(t *testing.T) {
	tests := []struct {
		testName string
		level    zap.AtomicLevel
	}{
		{
			testName: "DebugLevel",
			level:    zap.NewAtomicLevelAt(zapcore.DebugLevel),
		},
		{
			testName: "InfoLevel",
			level:    zap.NewAtomicLevelAt(zapcore.InfoLevel),
		},
		{
			testName: "WarnLevel",
			level:    zap.NewAtomicLevelAt(zapcore.WarnLevel),
		},
		{
			testName: "ErrorLevel",
			level:    zap.NewAtomicLevelAt(zapcore.ErrorLevel),
		},
		{
			testName: "DPanicLevel",
			level:    zap.NewAtomicLevelAt(zapcore.DPanicLevel),
		},
		{
			testName: "PanicLevel",
			level:    zap.NewAtomicLevelAt(zapcore.PanicLevel),
		},
		{
			testName: "FatalLevel",
			level:    zap.NewAtomicLevelAt(zapcore.FatalLevel),
		},
		{
			testName: "Invalid",
			level:    zap.NewAtomicLevelAt(zapcore.InvalidLevel),
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			err := logger.Init(tt.level)
			require.NoError(t, err)
		})
	}
}
