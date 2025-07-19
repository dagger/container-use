package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
	"golang.org/x/term"
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

	// Check if stdout is a TTY (interactive) vs piped/redirected (non-interactive)
	isInteractive := term.IsTerminal(int(os.Stdout.Fd()))

	if !isInteractive {
		// For non-interactive use (like MCP protocol), log to file to avoid interference
		logFile := "/tmp/container-use.debug.stderr.log"
		if v, ok := os.LookupEnv("CONTAINER_USE_STDERR_FILE"); ok {
			logFile = v
		}

		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file %s: %w", logFile, err)
		}
		writers = append(writers, file)
	} else {
		// For interactive use, log to stderr by default
		if v, ok := os.LookupEnv("CONTAINER_USE_STDERR_FILE"); ok {
			if v == "/dev/stderr" || v == "" {
				writers = append(writers, os.Stderr)
			} else {
				file, err := os.OpenFile(v, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
				if err != nil {
					return fmt.Errorf("failed to open log file %s: %w", v, err)
				}
				writers = append(writers, file)
			}
		} else {
			// Default to stderr for interactive use
			writers = append(writers, os.Stderr)
		}
	}

	if len(writers) == 0 {
		fmt.Fprintf(os.Stderr, "%s Logging disabled. Set CONTAINER_USE_STDERR_FILE and CONTAINER_USE_LOG_LEVEL environment variables\n", time.Now().Format(time.DateTime))
	}

	logLevel := parseLogLevel(os.Getenv("CONTAINER_USE_LOG_LEVEL"))
	logWriter = io.MultiWriter(writers...)
	
	var handler slog.Handler
	if !isInteractive {
		// For non-interactive use, use plain text handler for file logging
		handler = slog.NewTextHandler(logWriter, &slog.HandlerOptions{
			Level: logLevel,
		})
	} else {
		// For interactive use, use tint for prettier output
		handler = tint.NewHandler(logWriter, &tint.Options{
			Level:      logLevel,
			TimeFormat: time.Kitchen,
		})
	}
	slog.SetDefault(slog.New(handler))

	return nil
}
