package main

import (
	"context"
	"fmt"

	"github.com/dagger/container-use/repository"
)

func envOrDefault(ctx context.Context, arg string, repo *repository.Repository) (string, error) {
	if arg != "" {
		return arg, nil
	}
	if list, err := repo.List(ctx); err != nil {
		return "", err
	} else if len(list) == 0 {
		return "", fmt.Errorf("no environment found")
	} else if len(list) == 1 {
		return list[0].ID, nil
	} else {
		return "", fmt.Errorf("please specify an environment")
	}
}

func firstOrEmpty(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return ""
}
