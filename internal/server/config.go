package server

import (
	"time"

	"github.com/skayfish/metrics/internal/flags"
	"github.com/skayfish/metrics/internal/logger"
)

// Конфигурация сервера
type Config struct {
	// Адрес, по которому сервер ждёт запросы
	Address flags.NetAddress

	// Уровень логирования
	LogLevel logger.Level

	// Интервал времени в секундах, по истечении которого текущие данные хранилища метрик сохраняются на диск
	StoreInterval time.Duration

	// Путь до файла, куда сохраняются данные хранилища метрик. По умолчанию во временный файл
	FileStoragePath string

	// Булево значение, определяющее, следует ли загружать ранее сохранённые значения из указанного файла при старте сервера
	ToRestore bool
}
