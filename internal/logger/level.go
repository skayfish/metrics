package logger

import "go.uber.org/zap/zapcore"

// SF TODO
type Level struct {
	// SF TODO
	Lvl zapcore.Level
}

// SF TODO
func (Level) Type() string {
	return `["debug","info","warn","error","dpanic","panic","fatal"]`
}

// SF TODO
func (l *Level) Set(input string) error {
	return l.Lvl.Set(input)
}

// SF TODO
func (l Level) String() string {
	return l.Lvl.String()
}
