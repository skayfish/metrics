package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Менеджер логирования
var Log *zap.Logger = zap.NewNop()

// Менеджер логирования с дополнительным функционалом.
//
// Более медленный менеджер логирования, но более удобное использование
var LogS *zap.SugaredLogger = Log.Sugar()

// Текущий установленный уровень логирования. По умолчанию - некорректный. Устанавливается с помощью Init(...)
var CurLogLevel Level = Level(zapcore.InvalidLevel)

// Проверяет является ли текущий установленный уровень логирования - debug
func IsDebug() bool {
	return CurLogLevel == Level(zap.DebugLevel)
}

// Инициализирует менеджер логирования
//
//	@warning вызов defer logger.Log.Sync() после инициализации - обязателен!
//	@param level уровень логирования
//	@returns error ошибку, если не удалось создать менеджеры логирования
func Init(level Level) error {
	var config zap.Config
	if zapcore.Level(level) == zap.DebugLevel {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}

	config.Level = zap.NewAtomicLevelAt(zapcore.Level(level))

	logger, err := config.Build()
	if err != nil {
		return err
	}

	Log = logger
	LogS = logger.Sugar()

	CurLogLevel = level

	return nil
}

// Закрывает менеджер логирования
func Close() {
	Log = zap.NewNop()
	LogS = Log.Sugar()
}
