package environment

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultImage    = "ubuntu:24.04"
	alpineImage     = "alpine:3.21.3@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c"
	configDir       = ".container-use"
	environmentFile = "environment.json"
)

func DefaultConfig() *EnvironmentConfig {
	return &EnvironmentConfig{
		BaseImage: defaultImage,
		Workdir:   "/workdir",
	}
}

type EnvironmentConfig struct {
	Workdir         string         `json:"workdir,omitempty"`
	BaseImage       string         `json:"base_image,omitempty"`
	SetupCommands   []string       `json:"setup_commands,omitempty"`
	InstallCommands []string       `json:"install_commands,omitempty"`
	Env             KVList         `json:"env,omitempty"`
	Secrets         KVList         `json:"secrets,omitempty"`
	Services        ServiceConfigs `json:"services,omitempty"`
}

type ServiceConfig struct {
	Name         string   `json:"name,omitempty"`
	Image        string   `json:"image,omitempty"`
	Command      string   `json:"command,omitempty"`
	ExposedPorts []int    `json:"exposed_ports,omitempty"`
	Env          []string `json:"env,omitempty"`
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

// KVList represents a list of key-value pairs in the format KEY=VALUE
type KVList []string

// parseKeyValue parses a key-value string in the format "KEY=VALUE"
func (kv KVList) parseKeyValue(raw string) (key, value string) {
	k, v, _ := strings.Cut(raw, "=")
	return k, v
}

// Set adds or updates a key-value pair
func (kv *KVList) Set(key, value string) {
	// Remove existing key if it exists
	kv.Unset(key)
	// Add new key-value pair
	*kv = append(*kv, fmt.Sprintf("%s=%s", key, value))
}

// Unset removes a key-value pair by key and returns true if the key was found
func (kv *KVList) Unset(key string) bool {
	found := false
	newList := make([]string, 0, len(*kv))
	for _, item := range *kv {
		if itemKey, _ := kv.parseKeyValue(item); itemKey != key {
			newList = append(newList, item)
		} else {
			found = true
		}
	}
	*kv = newList
	return found
}

// Clear removes all key-value pairs
func (kv *KVList) Clear() {
	*kv = []string{}
}

// Keys returns all keys in the list
func (kv KVList) Keys() []string {
	keys := make([]string, 0, len(kv))
	for _, item := range kv {
		if key, _ := kv.parseKeyValue(item); key != "" {
			keys = append(keys, key)
		}
	}
	return keys
}

// Get returns the value for a given key, or empty string if not found
func (kv KVList) Get(key string) string {
	for _, item := range kv {
		if itemKey, value := kv.parseKeyValue(item); itemKey == key {
			return value
		}
	}
	return ""
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
	configPath := filepath.Join(baseDir, configDir)
	if err := os.MkdirAll(configPath, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(configPath, environmentFile), data, 0644); err != nil {
		return err
	}

	return nil
}

func (config *EnvironmentConfig) Load(baseDir string) error {
	configPath := filepath.Join(baseDir, configDir)

	data, err := os.ReadFile(filepath.Join(configPath, environmentFile))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		if err := json.Unmarshal(data, config); err != nil {
			return err
		}
	}

	return nil
}
