package encryption

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SF TODO
func TestSignHMAC(t *testing.T) {
	tests := []struct {
		test string

		data string
		key  string
		want string
	}{
		{
			test: "success",
			data: "",
			key:  "",
			want: "b613679a0814d9ec772f95d778c35fc5ff1697c493715653c6c712144292c5ad",
		},
		{
			test: "success",
			data: "some data",
			key:  "some key",
			want: "92003059a722e7632fc06d79b2c682849aa17195b617580464d048e12242c844",
		},
		{
			test: "success",
			data: "some data",
			key:  "some hard key",
			want: "e4d8e4265ce336ae76dac062e2e619c264d3058db417a0f776d7f0214781e89e",
		},
	}
	for _, tt := range tests {
		t.Run(tt.test, func(t *testing.T) {
			signed, err := SignHMAC([]byte(tt.data), []byte(tt.key), sha256.New)
			require.NoError(t, err)
			assert.Equal(t, tt.want, hex.EncodeToString(signed))
		})
	}
}

// SF TODO
func TestEqualHMAC(t *testing.T) {
	tests := []struct {
		test string

		data      string
		key       string
		targetHex string
		isEqual   bool
	}{
		{
			test:      "equal",
			data:      "",
			key:       "",
			targetHex: "b613679a0814d9ec772f95d778c35fc5ff1697c493715653c6c712144292c5ad",
			isEqual:   true,
		},
		{
			test:      "equal",
			data:      "some data",
			key:       "some key",
			targetHex: "92003059a722e7632fc06d79b2c682849aa17195b617580464d048e12242c844",
			isEqual:   true,
		},
		{
			test:      "equal",
			data:      "some data",
			key:       "some hard key",
			targetHex: "e4d8e4265ce336ae76dac062e2e619c264d3058db417a0f776d7f0214781e89e",
			isEqual:   true,
		},

		{
			test:      "not equal",
			data:      "",
			key:       "",
			targetHex: "e4d8e4265ce336ae76dac062e2e619c264d3058db417a0f776d7f0214781e89e",
			isEqual:   false,
		},
		{
			test:      "not equal",
			data:      "some data",
			key:       "some key",
			targetHex: "92003059a722e7632fc06d79",
			isEqual:   false,
		},
		{
			test:      "not equal",
			data:      "some data",
			key:       "some hard key",
			targetHex: "e4d8e4265ce336ae76dac062",
			isEqual:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.test, func(t *testing.T) {
			target, err := hex.DecodeString(tt.targetHex)
			require.NoError(t, err)
			ok, err := EqualHMAC([]byte(tt.data), []byte(tt.key), target, sha256.New)
			require.NoError(t, err)
			if tt.isEqual {
				assert.True(t, ok)
			} else {
				assert.False(t, ok)
			}
		})
	}
}
