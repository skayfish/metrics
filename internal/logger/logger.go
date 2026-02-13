package logger

import "go.uber.org/zap"

// Менеджер логирования
var Log *zap.Logger = zap.NewNop()

// Менеджер логирования с дополнительным функционалом.
//
// Более медленный менеджер логирования, но более удобное использование
var LogS *zap.SugaredLogger = Log.Sugar()

// Инициализирует менеджер логирования
//	@param level уровень логирования
//	@returns error ошибку, если не удалось создать менеджеры логирования
func Init(level zap.AtomicLevel) error {
	var config zap.Config
	if level.Level() == zap.DebugLevel {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}

	config.Level = level

	logger, err := config.Build()
	if err != nil {
		return err
	}

	Log = logger
	LogS = logger.Sugar()

	return nil
}
