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

	return nil
}
