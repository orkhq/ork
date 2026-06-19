// Package logging defines structured logging interfaces used throughout ork.
// It provides a restricted DebugLogger for adapters and runners (preventing
// excessive output) and a full Logger for the orchestrator itself. The
// underlying implementation wraps go.uber.org/zap.
package logging

// Field is a lightweight structured field abstraction.
// This prevents leaking zap/zerolog types everywhere.
type Field struct {
	Key   string
	Value any
}

// DebugLogger is intentionally limited.
// Adapters and runners should NOT log at INFO or higher.
type DebugLogger interface {
	Debug(msg string, fields ...Field)
	Trace(msg string, fields ...Field)

	With(fields ...Field) DebugLogger
}

// Logger is the full logger used by the orchestrator.
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)

	With(fields ...Field) Logger

	AsDebugLogger() DebugLogger
}
