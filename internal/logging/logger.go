package logging

import (
	"context"

	"go.uber.org/zap"
)

type loggerKey struct{}

var fallbackLogger *zap.SugaredLogger

func init() {
	config := zap.NewProductionConfig()
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.LevelKey = "severity"
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)

	if logger, err := config.Build(); err != nil {
		fallbackLogger = zap.NewNop().Sugar()
	} else {
		fallbackLogger = logger.Named("default").Sugar()
	}
}

// WithLogger return context with logger
func WithLogger(ctx context.Context, logger *zap.SugaredLogger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// FromContext extract logger from context
func FromContext(ctx context.Context) *zap.SugaredLogger {
	if logger, ok := ctx.Value(loggerKey{}).(*zap.SugaredLogger); ok {
		return logger
	}
	return fallbackLogger
}
