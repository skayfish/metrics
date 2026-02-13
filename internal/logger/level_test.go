package logger

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

// SF TODO
func TestLevel_Type(t *testing.T) {
	tests := []struct {
		testName string
		level    zapcore.Level
	}{
		{
			testName: "DebugLevel",
			level:    zapcore.DebugLevel,
		},
		{
			testName: "InfoLevel",
			level:    zapcore.InfoLevel,
		},
		{
			testName: "WarnLevel",
			level:    zapcore.WarnLevel,
		},
		{
			testName: "ErrorLevel",
			level:    zapcore.ErrorLevel,
		},
		{
			testName: "DPanicLevel",
			level:    zapcore.DPanicLevel,
		},
		{
			testName: "PanicLevel",
			level:    zapcore.PanicLevel,
		},
		{
			testName: "FatalLevel",
			level:    zapcore.FatalLevel,
		},
		{
			testName: "Invalid",
			level:    zapcore.InvalidLevel,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Equal(t, `["debug","info","warn","error","dpanic","panic","fatal"]`, Level{tt.level}.Type())
		})
	}
}

// SF TODO
func TestLevel_Set(t *testing.T) {
	tests := []struct {
		testName string
		input    string
		level    zapcore.Level
		wantErr  bool
	}{
		{
			testName: "DebugLevel",
			input:    "debug",
			level:    zapcore.DebugLevel,
		},
		{
			testName: "InfoLevel",
			input:    "info",
			level:    zapcore.InfoLevel,
		},
		{
			testName: "WarnLevel",
			input:    "warn",
			level:    zapcore.WarnLevel,
		},
		{
			testName: "ErrorLevel",
			input:    "error",
			level:    zapcore.ErrorLevel,
		},
		{
			testName: "DPanicLevel",
			input:    "dpanic",
			level:    zapcore.DPanicLevel,
		},
		{
			testName: "PanicLevel",
			input:    "panic",
			level:    zapcore.PanicLevel,
		},
		{
			testName: "FatalLevel",
			input:    "fatal",
			level:    zapcore.FatalLevel,
		},
		{
			testName: "Invalid",
			input:    "invalid",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			level := Level{Lvl: tt.level}
			err := level.Set(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.level, level.Lvl)
			}
		})
	}
}

// SF TODO
func TestLevel_String(t *testing.T) {
	tests := []struct {
		testName string
		level    zapcore.Level
		want     string
	}{
		{
			testName: "DebugLevel",
			level:    zapcore.DebugLevel,
			want:     "debug",
		},
		{
			testName: "InfoLevel",
			level:    zapcore.InfoLevel,
			want:     "info",
		},
		{
			testName: "WarnLevel",
			level:    zapcore.WarnLevel,
			want:     "warn",
		},
		{
			testName: "ErrorLevel",
			level:    zapcore.ErrorLevel,
			want:     "error",
		},
		{
			testName: "DPanicLevel",
			level:    zapcore.DPanicLevel,
			want:     "dpanic",
		},
		{
			testName: "PanicLevel",
			level:    zapcore.PanicLevel,
			want:     "panic",
		},
		{
			testName: "FatalLevel",
			level:    zapcore.FatalLevel,
			want:     "fatal",
		},
		{
			testName: "Invalid",
			level:    zapcore.InvalidLevel,
			want:     fmt.Sprintf("Level(%d)", zapcore.InvalidLevel),
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Equal(t, tt.want, Level{Lvl: tt.level}.String())
		})
	}
}
