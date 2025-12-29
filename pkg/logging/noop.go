package logging

type NoopDebugLogger struct{}

func (n *NoopDebugLogger) Debug(string, ...Field) {}
func (n *NoopDebugLogger) Trace(string, ...Field) {}
func (n *NoopDebugLogger) With(...Field) DebugLogger {
	return n
}
