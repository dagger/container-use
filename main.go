package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"dagger.io/dagger"
	"github.com/mark3labs/mcp-go/server"
)

var dag *dagger.Client
var debugWriter io.Writer

func main() {
	// Set up debug logging if LOG_FILE env var is set
	debugWriter = os.Stderr
	if logFile := os.Getenv("LOG_FILE"); logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		} else {
			// Tee stderr to both original stderr and log file
			debugWriter = io.MultiWriter(os.Stderr, f)
		}
	}

	var err error
	dag, err = dagger.Connect(context.Background(), dagger.WithLogOutput(os.Stderr))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting dagger: %v\n", err)
		os.Exit(1)
	}
	defer dag.Close()

	if err := LoadContainers(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading containers: %v\n", err)
		os.Exit(1)
	}

	s := server.NewMCPServer(
		"Dagger",
		"1.0.0",
	)

	for _, t := range tools {
		s.AddTool(t.Definition, t.Handler)
	}

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
