package logging

// NoopDebugLogger is a DebugLogger implementation that discards all log output.
// It is useful as a default logger when no logging is configured.
type NoopDebugLogger struct{}

func (n *NoopDebugLogger) Debug(string, ...Field) {}
func (n *NoopDebugLogger) Trace(string, ...Field) {}
func (n *NoopDebugLogger) With(...Field) DebugLogger {
	return n
}
