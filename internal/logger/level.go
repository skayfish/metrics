package logger

import "go.uber.org/zap/zapcore"

// Уровень логирования
type Level zapcore.Level

// Возвращает тип значения уровня логирования для документации
//	@returns тип значения уровня логирования для документации
func (Level) Type() string {
	return `["debug","info","warn","error","dpanic","panic","fatal"]`
}

// Обрабатывает входную строку уровня логирования и заполняет структуру
//	@param input входная строка уровня логирования
//	@returns ошибку в случае передачи некорректных данных
func (l *Level) Set(input string) error {
	temp := zapcore.Level(*l)
	err := temp.Set(input)
	*l = Level(temp)
	return err
}

// Возвращает строковое представление уровня логирования
//	@returns строковое представление адреса
func (l Level) String() string {
	return zapcore.Level(l).String()
}
