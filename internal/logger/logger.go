package logger

import "go.uber.org/zap"

// SF TODO
var Log *zap.Logger = zap.NewNop()

// SF TODO
var LogS *zap.SugaredLogger = Log.Sugar()

// SF TODO
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
