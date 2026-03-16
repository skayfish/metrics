package encryption

import (
	"crypto/hmac"
	"hash"
)

// SF TODO
func SignHMAC(data, key []byte, f func() hash.Hash) ([]byte, error) {
	h := hmac.New(f, key)
	_, err := h.Write(data)
	if err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

// SF TODO
func EqualHMAC(data, key, target []byte, f func() hash.Hash) (bool, error) {
	h := hmac.New(f, key)
	_, err := h.Write(data)
	if err != nil {
		return false, err
	}

	return hmac.Equal(h.Sum(nil), target), nil
}
