package main

import (
	"context"
	"fmt"
	"os"

	"dagger.io/dagger"
	"github.com/mark3labs/mcp-go/server"
)

var dag *dagger.Client

func main() {
	var err error
	dag, err = dagger.Connect(context.Background(), dagger.WithLogOutput(os.Stderr))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting dagger: %v\n", err)
		os.Exit(1)
	}
	defer dag.Close()

	// if err := LoadEnvironments(); err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error loading environments: %v\n", err)
	// 	os.Exit(1)
	// }

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
