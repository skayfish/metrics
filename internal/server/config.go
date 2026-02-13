package server

import (
	"github.com/skayfish/metrics/internal/flags"
	"go.uber.org/zap"
)

// SF TODO
type Config struct {
	// SF TODO
	Address flags.NetAddress

	// SF TODO
	LogLevel zap.AtomicLevel
}
