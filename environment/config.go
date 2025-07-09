package environment

import (
	"encoding/json"
	"os"
	"path"
)

const (
	defaultImage = "ubuntu:24.04"

	// WARNING: for maximum efficiency, please ensure golangImage is always based on this specific alpineImage
	alpineImage = "alpine:3.22.0@sha256:8a1f59ffb675680d47db6337b49d22281a139e9d709335b492be023728e11715"
	golangImage = "golang:1.24.5-alpine@sha256:ddf52008bce1be455fe2b22d780b6693259aaf97b16383b6372f4b22dd33ad66"

	configDir        = ".container-use"
	instructionsFile = "AGENT.md"
	environmentFile  = "environment.json"
	lockFile         = "lock"
)

func DefaultConfig() *EnvironmentConfig {
	return &EnvironmentConfig{
		BaseImage:    defaultImage,
		Instructions: "No instructions found. Please look around the filesystem and update me",
		Workdir:      "/workdir",
	}
}

type EnvironmentConfig struct {
	Instructions  string         `json:"-"`
	Workdir       string         `json:"workdir,omitempty"`
	BaseImage     string         `json:"base_image,omitempty"`
	SetupCommands []string       `json:"setup_commands,omitempty"`
	Env           []string       `json:"env,omitempty"`
	Secrets       []string       `json:"secrets,omitempty"`
	Services      ServiceConfigs `json:"services,omitempty"`
	Locked        bool
}

type ServiceConfig struct {
	Name         string   `json:"name,omitempty"`
	Image        string   `json:"image,omitempty"`
	Command      string   `json:"command,omitempty"`
	ExposedPorts []int    `json:"exposed_ports,omitempty"`
	Env          []string `json:"env,omitempty"`
	Secrets      []string `json:"secrets,omitempty"`
}

type ServiceConfigs []*ServiceConfig

func (sc ServiceConfigs) Get(name string) *ServiceConfig {
	for _, cfg := range sc {
		if cfg.Name == name {
			return cfg
		}
	}
	return nil
}

func (config *EnvironmentConfig) Copy() *EnvironmentConfig {
	copy := *config
	copy.Services = make(ServiceConfigs, len(config.Services))
	for i, svc := range config.Services {
		svcCopy := *svc
		copy.Services[i] = &svcCopy
	}
	return &copy
}

func (config *EnvironmentConfig) Save(baseDir string) error {
	configPath := path.Join(baseDir, configDir)
	if err := os.MkdirAll(configPath, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(path.Join(configPath, instructionsFile), []byte(config.Instructions), 0644); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path.Join(configPath, environmentFile), data, 0644); err != nil {
		return err
	}

	return nil
}

func (config *EnvironmentConfig) Load(baseDir string) error {
	configPath := path.Join(baseDir, configDir)

	instructions, err := os.ReadFile(path.Join(configPath, instructionsFile))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		config.Instructions = string(instructions)
	}

	data, err := os.ReadFile(path.Join(configPath, environmentFile))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		if err := json.Unmarshal(data, config); err != nil {
			return err
		}
	}
	if _, err := os.Stat(path.Join(baseDir, configDir, lockFile)); err == nil {
		config.Locked = true
	}

	return nil
}
