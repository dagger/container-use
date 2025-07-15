package environment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"dagger.io/dagger"
)

// EnvironmentInfo contains basic metadata about an environment
// without requiring dagger operations
type EnvironmentInfo struct {
	State *State `json:"state,omitempty"`

	ID string `json:"id,omitempty"`
}

type Environment struct {
	*EnvironmentInfo

	dag *dagger.Client

	Services []*Service
	Notes    Notes

	mu sync.RWMutex
}

func New(ctx context.Context, dag *dagger.Client, id, title string, config *EnvironmentConfig, initialSourceDir *dagger.Directory) (*Environment, error) {
	env := &Environment{
		EnvironmentInfo: &EnvironmentInfo{
			ID: id,
			State: &State{
				Config:    config,
				Title:     title,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
		dag: dag,
	}

	container, err := env.buildBase(ctx, initialSourceDir)
	if err != nil {
		return nil, err
	}

	slog.Info("Creating environment", "id", env.ID, "workdir", env.State.Config.Workdir)

	if err := env.apply(ctx, container); err != nil {
		return nil, err
	}

	return env, nil
}

func (env *Environment) Workdir() *dagger.Directory {
	return env.container().Directory(env.State.Config.Workdir)
}

func (env *Environment) container() *dagger.Container {
	env.mu.RLock()
	defer env.mu.RUnlock()

	return env.dag.LoadContainerFromID(dagger.ContainerID(env.State.Container))
}

func Load(ctx context.Context, dag *dagger.Client, id string, state []byte, worktree string) (*Environment, error) {
	envInfo, err := LoadInfo(ctx, id, state, worktree)
	if err != nil {
		return nil, err
	}
	env := &Environment{
		EnvironmentInfo: envInfo,
		dag:             dag,
		// Services: ?
	}

	return env, nil
}

// LoadInfo loads basic environment metadata without requiring dagger operations.
// This is useful for operations that only need access to configuration and state
// information without the overhead of initializing container operations.
func LoadInfo(ctx context.Context, id string, state []byte, worktree string) (*EnvironmentInfo, error) {
	envInfo := &EnvironmentInfo{
		ID:    id,
		State: &State{},
	}

	if err := envInfo.State.Unmarshal(state); err != nil {
		return nil, err
	}

	// Backward compatibility: if there's no config in the state, load it from the worktree
	if envInfo.State.Config == nil {
		config := DefaultConfig()
		if err := config.Load(worktree); err != nil {
			return nil, err
		}
		envInfo.State.Config = config
	}

	return envInfo, nil
}

func (env *Environment) apply(ctx context.Context, newState *dagger.Container) error {
	// TODO(braa): is this sync redundant with newState.ID?
	if _, err := newState.Sync(ctx); err != nil {
		return err
	}

	containerID, err := newState.ID(ctx)
	if err != nil {
		return err
	}

	env.mu.Lock()
	defer env.mu.Unlock()
	env.State.UpdatedAt = time.Now()
	env.State.Container = string(containerID)

	return nil
}

func containerWithEnvAndSecrets(dag *dagger.Client, container *dagger.Container, envs, secrets []string) (*dagger.Container, error) {
	for _, env := range envs {
		k, v, found := strings.Cut(env, "=")
		if !found {
			return nil, fmt.Errorf("invalid env variable: %s", env)
		}
		if !found {
			return nil, fmt.Errorf("invalid environment variable: %s", env)
		}
		container = container.WithEnvVariable(k, v)
	}

	for _, secret := range secrets {
		k, v, found := strings.Cut(secret, "=")
		if !found {
			return nil, fmt.Errorf("invalid secret: %s", secret)
		}
		container = container.WithSecretVariable(k, dag.Secret(v))
	}

	return container, nil
}

func (env *Environment) buildBase(ctx context.Context, baseSourceDir *dagger.Directory) (*dagger.Container, error) {
	container := env.dag.
		Container().
		From(env.State.Config.BaseImage).
		WithWorkdir(env.State.Config.Workdir)

	container, err := containerWithEnvAndSecrets(env.dag, container, env.State.Config.Env, env.State.Config.Secrets)
	if err != nil {
		return nil, err
	}

	runCommands := func(commands []string) error {
		for _, command := range commands {
			var err error

			container = container.WithExec([]string{"sh", "-c", command})

			exitCode, err := container.ExitCode(ctx)
			if err != nil {
				var exitErr *dagger.ExecError
				if errors.As(err, &exitErr) {
					env.Notes.AddCommand(command, exitErr.ExitCode, exitErr.Stdout, exitErr.Stderr)
					return fmt.Errorf("exit code %d.\nstdout: %s\nstderr: %s\n%w", exitErr.ExitCode, exitErr.Stdout, exitErr.Stderr, err)
				}

				return err
			}
			stdout, err := container.Stdout(ctx)
			if err != nil {
				return fmt.Errorf("failed to get stdout: %w", err)
			}

			stderr, err := container.Stderr(ctx)
			if err != nil {
				return fmt.Errorf("failed to get stderr: %w", err)
			}

			env.Notes.AddCommand(command, exitCode, stdout, stderr)
		}

		return nil
	}

	// Run setup commands without the source directory for caching purposes
	if err := runCommands(env.State.Config.SetupCommands); err != nil {
		return nil, fmt.Errorf("setup command failed: %w", err)
	}

	env.Services, err = env.startServices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start services: %w", err)
	}
	for _, service := range env.Services {
		container = container.WithServiceBinding(service.Config.Name, service.svc)
	}

	container = container.WithDirectory(".", baseSourceDir)

	// Run the install commands after the source directory is set up
	if err := runCommands(env.State.Config.InstallCommands); err != nil {
		return nil, fmt.Errorf("install command failed: %w", err)
	}

	return container, nil
}

func (env *Environment) UpdateConfig(ctx context.Context, newConfig *EnvironmentConfig) error {
	env.State.Config = newConfig

	// Re-build the base image with the new config
	container, err := env.buildBase(ctx, env.Workdir())
	if err != nil {
		return err
	}

	if err := env.apply(ctx, container); err != nil {
		return err
	}

	return nil
}

func (env *Environment) Run(ctx context.Context, command, shell string, useEntrypoint bool) (string, error) {
	args := []string{}
	if command != "" {
		args = []string{shell, "-c", command}
	}
	newState := env.container().WithExec(args, dagger.ContainerWithExecOpts{
		UseEntrypoint:                 useEntrypoint,
		Expect:                        dagger.ReturnTypeAny, // Don't treat non-zero exit as error
		ExperimentalPrivilegedNesting: true,
	})

	exitCode, err := newState.ExitCode(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get exit code: %w", err)
	}

	stdout, err := newState.Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get stdout: %w", err)
	}

	stderr, err := newState.Stderr(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get stderr: %w", err)
	}

	// Log the command execution with all details
	env.Notes.AddCommand(command, exitCode, stdout, stderr)

	// Always apply the container state (preserving changes even on non-zero exit)
	if err := env.apply(ctx, newState); err != nil {
		return stdout, fmt.Errorf("failed to apply container state: %w", err)
	}

	// Return combined output (stdout + stderr if there was stderr)
	combinedOutput := stdout
	if stderr != "" {
		if stdout != "" {
			combinedOutput += "\n"
		}
		combinedOutput += "stderr: " + stderr
	}
	return combinedOutput, nil
}

func (env *Environment) RunBackground(ctx context.Context, command, shell string, ports []int, useEntrypoint bool) (EndpointMappings, error) {
	args := []string{}
	if command != "" {
		args = []string{shell, "-c", command}
	}
	displayCommand := command + " &"
	serviceState := env.container()

	// Expose ports
	for _, port := range ports {
		serviceState = serviceState.WithExposedPort(port, dagger.ContainerWithExposedPortOpts{
			Protocol:    dagger.NetworkProtocolTcp,
			Description: fmt.Sprintf("Port %d", port),
		})
	}

	// Start the service
	startCtx, cancel := context.WithTimeout(ctx, serviceStartTimeout)
	defer cancel()
	svc, err := serviceState.AsService(dagger.ContainerAsServiceOpts{
		Args:          args,
		UseEntrypoint: useEntrypoint,
	}).Start(startCtx)
	if err != nil {
		var exitErr *dagger.ExecError
		if errors.As(err, &exitErr) {
			env.Notes.AddCommand(displayCommand, exitErr.ExitCode, exitErr.Stdout, exitErr.Stderr)
			return nil, fmt.Errorf("command failed with exit code %d.\nstdout: %s\nstderr: %s", exitErr.ExitCode, exitErr.Stdout, exitErr.Stderr)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("service failed to start within %s timeout", serviceStartTimeout)
			env.Notes.AddCommand(displayCommand, 137, "", err.Error())
			return nil, err
		}
		return nil, err
	}

	env.Notes.AddCommand(displayCommand, 0, "", "")

	endpoints := EndpointMappings{}
	for _, port := range ports {
		endpoint := &EndpointMapping{}
		endpoints[port] = endpoint

		// Expose port on the host
		tunnel, err := env.dag.Host().Tunnel(svc, dagger.HostTunnelOpts{
			Ports: []dagger.PortForward{
				{
					Backend:  port,
					Protocol: dagger.NetworkProtocolTcp,
				},
			},
		}).Start(ctx)
		if err != nil {
			return nil, err
		}

		externalEndpoint, err := tunnel.Endpoint(ctx, dagger.ServiceEndpointOpts{
			Scheme: "tcp",
		})
		if err != nil {
			return nil, err
		}
		endpoint.HostExternal = externalEndpoint

		internalEndpoint, err := svc.Endpoint(ctx, dagger.ServiceEndpointOpts{
			Port:   port,
			Scheme: "tcp",
		})
		if err != nil {
			return nil, err
		}
		endpoint.EnvironmentInternal = internalEndpoint
	}

	return endpoints, nil
}

func (env *Environment) Terminal(ctx context.Context) error {
	container := env.container()
	var cmd []string
	var sourceRC string
	if shells, err := container.File("/etc/shells").Contents(ctx); err == nil {
		for shell := range strings.Lines(shells) {
			if shell[0] == '#' {
				continue
			}
			shell = strings.TrimRight(shell, "\n")
			if strings.HasSuffix(shell, "/bash") {
				sourceRC = fmt.Sprintf("[ -f ~/.bashrc ] && . ~/.bashrc; %q --version | head -4; ", shell)
				cmd = []string{shell, "--rcfile", "/cu/rc.sh", "-i"}
				break
			}
		}
	}
	// Try to show the same pretty PS1 as for the default /bin/sh terminal in dagger
	container = container.WithNewFile("/cu/rc.sh", sourceRC+`export PS1="\033[33mcu\033[0m \033[02m\$(pwd | sed \"s|^\$HOME|~|\")\033[0m \$ "`+"\n")
	if cmd == nil {
		// If bash not available, assume POSIX shell
		container = container.WithEnvVariable("ENV", "/cu/rc.sh")
		cmd = []string{"sh"}
	}
	if _, err := container.Terminal(dagger.ContainerTerminalOpts{
		ExperimentalPrivilegedNesting: true,
		Cmd:                           cmd,
	}).Sync(ctx); err != nil {
		return err
	}
	return nil
}

func (env *Environment) Checkpoint(ctx context.Context, target string) (string, error) {
	return env.container().Publish(ctx, target)
}
