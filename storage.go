package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"dagger.io/dagger"
	"github.com/mitchellh/go-homedir"
)

func saveState(c *Environment) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}

	environmentDir, err := homedir.Expand(fmt.Sprintf("~/.config/environment-use/%s", c.ID))
	if err != nil {
		return err
	}
	statesDir := filepath.Join(environmentDir, "states")
	if err := os.MkdirAll(statesDir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(environmentDir, "environment.json"), data, 0644); err != nil {
		return err
	}

	latest := c.History.Latest()
	stateID, err := latest.container.ID(context.Background())
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(statesDir, fmt.Sprintf("%d", latest.Version)), []byte(stateID), 0644); err != nil {
		return err
	}

	return nil
}

func loadState() (map[string]*Environment, error) {
	stateDir, err := homedir.Expand("~/.config/environment-use")
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(stateDir)
	if err != nil {
		return nil, err
	}
	environments := map[string]*Environment{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := entry.Name()
		stateFile := filepath.Join(stateDir, id, "environment.json")
		data, err := os.ReadFile(stateFile)
		if err != nil {
			return nil, err
		}
		var c Environment
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		for _, revision := range c.History {
			state := filepath.Join(stateDir, id, "states", fmt.Sprintf("%d", revision.Version))
			data, err = os.ReadFile(state)
			if err != nil {
				return nil, err
			}
			revision.container = dag.LoadContainerFromID(dagger.ContainerID(data))
		}
		c.container = c.History.Latest().container

		environments[id] = &c
	}
	return environments, nil
}
