package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"dagger.io/dagger"

	"github.com/google/uuid"
)

const (
	AlpineImage = "alpine:3.21.3@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c"

	environmentFile = "container-use.json"
)

type Version int

type Revision struct {
	Version     Version   `json:"version"`
	Name        string    `json:"name"`
	Explanation string    `json:"explanation"`
	Output      string    `json:"output,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	State       string    `json:"state"`

	container *dagger.Container `json:"-"`
}

type History []*Revision

func (h History) Latest() *Revision {
	if len(h) == 0 {
		return nil
	}
	return h[len(h)-1]
}

func (h History) LatestVersion() Version {
	latest := h.Latest()
	if latest == nil {
		return 0
	}
	return latest.Version
}

func (h History) Get(version Version) *Revision {
	for _, revision := range h {
		if revision.Version == version {
			return revision
		}
	}
	return nil
}

type Environment struct {
	ID           string `json:"id"`
	Source       string `json:"-"`
	Dockerfile   string `json:"dockerfile"`
	Instructions string `json:"instructions"`

	Name    string  `json:"name"`
	Image   string  `json:"image"`
	Workdir string  `json:"workdir"`
	History History `json:"history"`

	mu        sync.Mutex
	container *dagger.Container
}

var environments = map[string]*Environment{}

func LoadEnvironments() error {
	env, err := loadState()
	if err != nil {
		return err
	}
	environments = env
	return nil
}

func CreateEnvironment(ctx context.Context, source, name string) (*Environment, error) {
	env := &Environment{
		ID:           uuid.New().String(),
		Source:       source,
		Dockerfile:   "FROM ubuntu:latest",
		Instructions: "No instructions found. Please look around the filesystem and update me",
	}

	container, err := env.build(ctx, env.Dockerfile)
	if err != nil {
		return nil, err
	}

	if err := env.apply(ctx, "Create environment", "Create the environment", "", container); err != nil {
		return nil, err
	}
	environments[env.ID] = env
	return env, nil
}

func OpenEnvironment(ctx context.Context, source, name string) (*Environment, error) {
	def, err := os.ReadFile(path.Join(source, environmentFile))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CreateEnvironment(ctx, source, name)
		}
		return nil, err
	}
	env := &Environment{
		Source: source,
	}
	if err := json.Unmarshal(def, env); err != nil {
		return nil, err
	}
	for _, revision := range env.History {
		revision.container = dag.LoadContainerFromID(dagger.ContainerID(revision.State))
	}
	env.container = env.History.Latest().container

	environments[env.ID] = env
	return env, nil
}

func (s *Environment) build(ctx context.Context, dockerfile string) (*dagger.Container, error) {
	container, err := dag.Directory().WithNewFile("Dockerfile", dockerfile).DockerBuild().Sync(ctx)
	if err != nil {
		return nil, err
	}
	sourceDir := dag.Host().Directory(s.Source)
	container = container.WithWorkdir("/workdir").WithDirectory(".", sourceDir)

	return container, nil
}

func (s *Environment) Update(ctx context.Context, explanation, dockerfile, instructions string) error {
	s.Dockerfile = dockerfile
	s.Instructions = instructions

	container, err := s.build(ctx, s.Dockerfile)
	if err != nil {
		return err
	}

	return s.apply(ctx, "Update environment", explanation, "", container)
}

func GetEnvironment(idOrName string) *Environment {
	if environment, ok := environments[idOrName]; ok {
		return environment
	}
	for _, environment := range environments {
		if environment.Name == idOrName {
			return environment
		}
	}
	return nil
}

func ListEnvironments() []*Environment {
	env := make([]*Environment, 0, len(environments))
	for _, environment := range environments {
		env = append(env, environment)
	}
	return env
}

func (e *Environment) apply(ctx context.Context, name, explanation, output string, newState *dagger.Container) error {
	if _, err := newState.Sync(ctx); err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	revision := &Revision{
		Version:     e.History.LatestVersion() + 1,
		Name:        name,
		Explanation: explanation,
		Output:      output,
		CreatedAt:   time.Now(),
		container:   newState,
	}
	containerID, err := revision.container.ID(ctx)
	if err != nil {
		return err
	}
	revision.State = string(containerID)
	e.container = revision.container
	e.History = append(e.History, revision)

	out, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path.Join(e.Source, environmentFile), out, 0644); err != nil {
		return err
	}

	return nil
}

func (s *Environment) Run(ctx context.Context, explanation, command, shell string, useEntrypoint bool) (string, error) {
	args := []string{}
	if command != "" {
		args = []string{shell, "-c", command}
	}
	newState := s.container.WithExec(args, dagger.ContainerWithExecOpts{
		UseEntrypoint: useEntrypoint,
	})
	stdout, err := newState.Stdout(ctx)
	if err != nil {
		var exitErr *dagger.ExecError
		if errors.As(err, &exitErr) {
			return fmt.Sprintf("command failed with exit code %d.\nstdout: %s\nstderr: %s", exitErr.ExitCode, exitErr.Stdout, exitErr.Stderr), nil
		}
		return "", err
	}
	if err := s.apply(ctx, "Run "+command, explanation, stdout, newState); err != nil {
		return "", err
	}

	return stdout, nil
}

type EndpointMapping struct {
	Internal string `json:"internal"`
	External string `json:"external"`
}

type EndpointMappings map[int]*EndpointMapping

func (s *Environment) RunBackground(ctx context.Context, explanation, command, shell string, ports []int, useEntrypoint bool) (EndpointMappings, error) {
	args := []string{}
	if command != "" {
		args = []string{shell, "-c", command}
	}
	serviceState := s.container

	// Expose ports
	for _, port := range ports {
		serviceState = serviceState.WithExposedPort(port, dagger.ContainerWithExposedPortOpts{
			Protocol:    dagger.NetworkProtocolTcp,
			Description: fmt.Sprintf("Port %d", port),
		})
	}

	// Start the service
	svc, err := serviceState.AsService(dagger.ContainerAsServiceOpts{
		Args:          args,
		UseEntrypoint: useEntrypoint,
	}).Start(context.Background())
	if err != nil {
		var exitErr *dagger.ExecError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("command failed with exit code %d.\nstdout: %s\nstderr: %s", exitErr.ExitCode, exitErr.Stdout, exitErr.Stderr)
		}
		return nil, err
	}

	endpoints := EndpointMappings{}
	hostForwards := []dagger.PortForward{}

	for _, port := range ports {
		endpoints[port] = &EndpointMapping{}
		hostForwards = append(hostForwards, dagger.PortForward{
			Backend:  port,
			Frontend: rand.Intn(1000) + 5000,
			Protocol: dagger.NetworkProtocolTcp,
		})
	}

	// Expose ports on the host
	tunnel, err := dag.Host().Tunnel(svc, dagger.HostTunnelOpts{Ports: hostForwards}).Start(context.Background())
	if err != nil {
		return nil, err
	}

	// Retrieve endpoints
	for _, forward := range hostForwards {
		externalEndpoint, err := tunnel.Endpoint(ctx, dagger.ServiceEndpointOpts{
			Port: forward.Frontend,
		})
		if err != nil {
			return nil, err
		}

		endpoints[forward.Backend].External = externalEndpoint
	}
	for port, endpoint := range endpoints {
		internalEndpoint, err := svc.Endpoint(ctx, dagger.ServiceEndpointOpts{
			Port: port,
		})
		if err != nil {
			return nil, err
		}
		endpoint.Internal = internalEndpoint
	}

	return endpoints, nil
}

func (s *Environment) SetEnv(ctx context.Context, explanation string, envs []string) error {
	state := s.container
	for _, env := range envs {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid environment variable: %s", env)
		}
		state = state.WithEnvVariable(parts[0], parts[1])
	}
	return s.apply(ctx, "Set env "+strings.Join(envs, ", "), explanation, "", state)
}

func (s *Environment) Revert(ctx context.Context, explanation string, version Version) error {
	revision := s.History.Get(version)
	if revision == nil {
		return errors.New("no revisions found")
	}
	if err := s.apply(ctx, "Revert to "+revision.Name, explanation, "", revision.container); err != nil {
		return err
	}
	return nil
}

func (s *Environment) Fork(ctx context.Context, explanation, name string, version *Version) (*Environment, error) {
	revision := s.History.Latest()
	if version != nil {
		revision = s.History.Get(*version)
	}
	if revision == nil {
		return nil, errors.New("version not found")
	}

	forkedEnvironment := &Environment{
		ID:    uuid.New().String(),
		Name:  name,
		Image: s.Image,
	}
	if err := forkedEnvironment.apply(ctx, "Fork from "+s.Name, explanation, "", revision.container); err != nil {
		return nil, err
	}
	environments[forkedEnvironment.ID] = forkedEnvironment
	return forkedEnvironment, nil
}
