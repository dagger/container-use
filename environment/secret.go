package environment

import (
	"fmt"
	"strings"
)

// Secrets represents a list of secret specifications in the format NAME=schema://location
type Secrets []string

// AddSecret adds a new secret to the list
func (s *Secrets) AddSecret(name, spec string) error {
	// Validate secret format
	if err := validateSecretSpec(spec); err != nil {
		return err
	}

	// Check if secret already exists
	if s.Get(name) != "" {
		return fmt.Errorf("secret %s already exists", name)
	}

	*s = append(*s, fmt.Sprintf("%s=%s", name, spec))
	return nil
}

// DeleteSecret removes a secret by name
func (s *Secrets) DeleteSecret(name string) error {
	if s.Get(name) == "" {
		return fmt.Errorf("secret %s not found", name)
	}

	newSecrets := make([]string, 0, len(*s))
	for _, secret := range *s {
		secretName, _ := parseSecretSpec(secret)
		if secretName != name {
			newSecrets = append(newSecrets, secret)
		}
	}
	*s = newSecrets
	return nil
}

// Get returns the full secret spec for a given secret name, or empty string if not found
func (s Secrets) Get(name string) string {
	for _, secret := range s {
		secretName, _ := parseSecretSpec(secret)
		if secretName == name {
			return secret
		}
	}
	return ""
}

// List returns all secret names
func (s Secrets) List() []string {
	names := make([]string, 0, len(s))
	for _, secret := range s {
		name, _ := parseSecretSpec(secret)
		names = append(names, name)
	}
	return names
}

// parseSecretSpec splits a secret specification into name and value
func parseSecretSpec(spec string) (name, value string) {
	parts := strings.SplitN(spec, "=", 2)
	if len(parts) != 2 {
		return spec, ""
	}
	return parts[0], parts[1]
}

// validateSecretSpec ensures the secret specification is valid
func validateSecretSpec(value string) error {
	schemaParts := strings.SplitN(value, "://", 2)
	if len(schemaParts) != 2 {
		return fmt.Errorf("invalid secret value format: %s (expected schema://value)", value)
	}

	schema := schemaParts[0]
	switch schema {
	case "file", "env", "op":
		// Valid schemas
	default:
		return fmt.Errorf("unsupported secret schema: %s", schema)
	}

	return nil
}
