package logging

import (
	"go.uber.org/zap"
)

type zapLogger struct {
	l *zap.Logger
}

func NewZapLogger(l *zap.Logger) Logger {
	return &zapLogger{l: l}
}

func (z *zapLogger) Debug(msg string, fields ...Field) {
	z.l.Debug(msg, zapFields(fields)...)
}

func (z *zapLogger) Info(msg string, fields ...Field) {
	z.l.Info(msg, zapFields(fields)...)
}

func (z *zapLogger) Warn(msg string, fields ...Field) {
	z.l.Warn(msg, zapFields(fields)...)
}

func (z *zapLogger) Error(msg string, fields ...Field) {
	z.l.Error(msg, zapFields(fields)...)
}

func (z *zapLogger) With(fields ...Field) Logger {
	return &zapLogger{
		l: z.l.With(zapFields(fields)...),
	}
}

func (z *zapLogger) AsDebugLogger() DebugLogger {
	return &zapDebugLogger{l: z.l}
}

type zapDebugLogger struct {
	l *zap.Logger
}

func (z *zapDebugLogger) Debug(msg string, fields ...Field) {
	z.l.Debug(msg, zapFields(fields)...)
}

func (z *zapDebugLogger) Trace(msg string, fields ...Field) {
	// zap has no Trace level; map to Debug with a marker
	z.l.Debug("[trace] "+msg, zapFields(fields)...)
}

func (z *zapDebugLogger) With(fields ...Field) DebugLogger {
	return &zapDebugLogger{
		l: z.l.With(zapFields(fields)...),
	}
}

func zapFields(fields []Field) []zap.Field {
	if len(fields) == 0 {
		return nil
	}

	out := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		out = append(out, zap.Any(f.Key, f.Value))
	}
	return out
}
