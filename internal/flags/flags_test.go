package flags_test

import (
	"errors"
	"testing"

	"github.com/skayfish/metrics/internal/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Проверяет преобразование адреса в строку
func TestNetAddress_String(t *testing.T) {
	tests := []struct {
		testName string
		addr     flags.NetAddress
		want     string
	}{
		{
			"empty host",
			flags.NetAddress{Host: "", Port: 13},
			":13",
		},
		{
			"empty host",
			flags.NetAddress{Host: "", Port: -135},
			":-135",
		},
		{
			"not empty host",
			flags.NetAddress{Host: "host", Port: -135},
			"host:-135",
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.addr.String())
		})
	}
}

// Проверяет обработку входной строки адресу
func TestNetAddress_Set(t *testing.T) {
	tests := []struct {
		testName string
		input    string
		wantErr  bool
		err      error
		host     string
		port     int
	}{
		{
			testName: "invalid format",
			input:    "",
			wantErr:  true,
			err:      errors.New("invalid format: expected 'host:port', got ''"),
		},
		{
			testName: "invalid format",
			input:    "beleb_.erda",
			wantErr:  true,
			err:      errors.New("invalid format: expected 'host:port', got 'beleb_.erda'"),
		},
		{
			testName: "invalid format",
			input:    "::",
			wantErr:  true,
			err:      errors.New("invalid format: expected 'host:port', got '::'"),
		},
		{
			testName: "invalid port",
			input:    ":port",
			wantErr:  true,
			err:      errors.New("invalid port: 'port'"),
		},
		{
			testName: "port not in range",
			input:    ":-1",
			wantErr:  true,
			err:      errors.New("port must be in range 1–65535, got -1"),
		},
		{
			testName: "port not in range",
			input:    ":0",
			wantErr:  true,
			err:      errors.New("port must be in range 1–65535, got 0"),
		},
		{
			testName: "port not in range",
			input:    ":999999",
			wantErr:  true,
			err:      errors.New("port must be in range 1–65535, got 999999"),
		},
		{
			testName: "empty host",
			input:    ":9999",
			wantErr:  false,
			host:     "",
			port:     9999,
		},
		{
			testName: "correct",
			input:    "localhost:9999",
			wantErr:  false,
			host:     "localhost",
			port:     9999,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			addr := flags.NetAddress{}
			err := addr.Set(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.err, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.host, addr.Host)
				assert.Equal(t, tt.port, addr.Port)
			}
		})
	}
}

// Проверяет возвращаемый тип значения для документации
func TestNetAddress_Type(t *testing.T) {
	want := "host:port"
	tests := []struct {
		testName string
		addr     flags.NetAddress
	}{
		{
			"empty host",
			flags.NetAddress{Host: "", Port: 13},
		},
		{
			"empty host",
			flags.NetAddress{Host: "", Port: -135},
		},
		{
			"not empty host",
			flags.NetAddress{Host: "host", Port: -135},
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Equal(t, want, tt.addr.Type())
		})
	}
}
