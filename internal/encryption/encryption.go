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
func CheckHMAC(data, key []byte, f func() hash.Hash) error {
	h := hmac.New(f, key)
	_, err := h.Write(data)
	return err
}
