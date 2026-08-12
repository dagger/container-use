package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/dagger/container-use/rules"
	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v3"
)

type ConfigureGoose struct {
	Name        string
	Description string
}

func NewConfigureGoose() *ConfigureGoose {
	return &ConfigureGoose{
		Name:        "Goose",
		Description: "an open source, extensible AI agent that goes beyond code suggestions",
	}
}

func (a *ConfigureGoose) name() string {
	return a.Name
}

func (a *ConfigureGoose) description() string {
	return a.Description
}

func gooseConfigPath() (string, error) {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		return filepath.Join(appData, "Block", "goose", "config", "config.yaml"), nil
	}
	return homedir.Expand(filepath.Join("~", ".config", "goose", "config.yaml"))
}

func (a *ConfigureGoose) editMcpConfig() error {
	configPath, err := gooseConfigPath()
	if err != nil {
		return err
	}

	return writeMcpConfig(configPath, func(data []byte, cfg *map[string]any) error {
		return yaml.Unmarshal(data, cfg)
	}, a.updateGooseConfig)
}

func (a *ConfigureGoose) updateGooseConfig(config map[string]any) ([]byte, error) {
	if config == nil {
		config = make(map[string]any)
	}

	var extensions map[string]any
	if ext, ok := config["extensions"]; ok {
		extensions = ext.(map[string]any)
	} else {
		extensions = make(map[string]any)
		config["extensions"] = extensions
	}

	extensions["container-use"] = map[string]any{
		"name":    "container-use",
		"type":    "stdio",
		"enabled": true,
		"cmd":     ContainerUseBinary,
		"args":    []any{"stdio"},
		"envs":    map[string]any{},
	}

	data, err := yaml.Marshal(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	return data, nil
}

func (a *ConfigureGoose) editRules() error {
	return saveRulesFile(".goosehints", rules.AgentRules)
}

func (a *ConfigureGoose) isInstalled() bool {
	_, err := exec.LookPath("goose")
	return err == nil
}
