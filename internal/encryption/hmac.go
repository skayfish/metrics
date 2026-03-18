package encryption

import (
	"crypto/hmac"
	"hash"
)

// Создаёт подпись hmac для данных
//
//	@param data данные для подписи
//	@param key  ключ для подписи
//	@param f    функция хеширования для алгоритма hmac
//	@returns []byte подпись hmac для данных
//	@returns error ошибку, если при создании подписи возникли проблемы
func SignHMAC(data, key []byte, f func() hash.Hash) ([]byte, error) {
	h := hmac.New(f, key)
	_, err := h.Write(data)
	if err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

// Проверяет подписи hmac на равенство
//
//	@param data   данные для подписи
//	@param key    ключ для подписи
//	@param target подпись для сравнения
//	@param f      функция хеширования для алгоритма hmac
//	@returns bool положительный ответ, если подписи равны, отрицательный в ином случае
//	@returns error ошибку, если при проверки подписей на равенство возникли проблемы
func EqualHMAC(data, key, target []byte, f func() hash.Hash) (bool, error) {
	h := hmac.New(f, key)
	_, err := h.Write(data)
	if err != nil {
		return false, err
	}

	return hmac.Equal(h.Sum(nil), target), nil
}
