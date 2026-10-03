package logkit_test

import (
	"context"
	"os"

	"github.com/lemon4ksan/foundation/async/logkit"
)

func ExampleLogger() {
	// Initialize default configuration
	cfg := logkit.DefaultConfig(logkit.LevelInfo)
	cfg.Output = os.Stdout
	logger := logkit.New(cfg)
	defer logger.Close()

	// Log messages with structured context
	logger.Info("user logged in",
		logkit.String("username", "john_doe"),
		logkit.Int("attempts", 3),
	)
}

func ExampleLogger_With() {
	cfg := logkit.DefaultConfig(logkit.LevelInfo)
	cfg.Output = os.Stdout
	baseLog := logkit.New(cfg)
	defer baseLog.Close()

	authLog := baseLog.With(logkit.Component("auth"))
	dbLog := baseLog.With(logkit.Component("postgres"))

	authLog.Info("token verified", logkit.String("sub", "usr_100"))
	dbLog.Warn("slow query detected", logkit.Int("duration_ms", 125))
}

func ExampleLogger_contextAware() {
	cfg := logkit.DefaultConfig(logkit.LevelInfo)
	cfg.Output = os.Stdout
	logger := logkit.New(cfg)
	defer logger.Close()

	ctx := context.Background()
	reqCtx := logkit.WithCorrelationID(ctx, logkit.GenerateCorrelationID())

	logger.InfoContext(reqCtx, "processing user transaction",
		logkit.String("action", "transfer_funds"),
	)
}
