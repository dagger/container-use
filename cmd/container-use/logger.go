package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"
)

var (
	logWriter = io.Discard
)

func parseLogLevel(levelStr string) slog.Level {
	switch levelStr {
	case "debug", "DEBUG":
		return slog.LevelDebug
	case "info", "INFO":
		return slog.LevelInfo
	case "warn", "WARN", "warning", "WARNING":
		return slog.LevelWarn
	case "error", "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func setupLogger() error {
	var writers []io.Writer

	logFile := "/tmp/container-use.debug.stderr.log"
	if v, ok := os.LookupEnv("CONTAINER_USE_STDERR_FILE"); ok {
		logFile = v
	}

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", logFile, err)
	}
	writers = append(writers, file)

	if len(writers) == 0 {
		fmt.Fprintf(os.Stderr, "%s Logging disabled. Set CONTAINER_USE_STDERR_FILE and CONTAINER_USE_LOG_LEVEL environment variables\n", time.Now().Format(time.DateTime))
	}

	logLevel := parseLogLevel(os.Getenv("CONTAINER_USE_LOG_LEVEL"))
	logWriter = io.MultiWriter(writers...)
	handler := slog.NewTextHandler(logWriter, &slog.HandlerOptions{
		Level: logLevel,
	})
	slog.SetDefault(slog.New(handler))

	return nil
}
