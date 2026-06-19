package logging

import "go.uber.org/zap"

// NewRootZapLogger creates a configured zap.Logger for the ork process. It
// selects console encoding for TTY sessions and JSON encoding otherwise, and
// sets the log level to Debug when debug mode is enabled.
func NewRootZapLogger(isTTY bool, debug bool) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()

	if isTTY {
		cfg.Encoding = "console"
		cfg.EncoderConfig.TimeKey = "" // hide timestamps
	} else {
		cfg.Encoding = "json"
	}

	if debug {
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	return cfg.Build()
}
