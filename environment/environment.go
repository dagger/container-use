package environment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"dagger.io/dagger"

	petname "github.com/dustinkirkland/golang-petname"
)

var dag *dagger.Client

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

func Initialize(client *dagger.Client) error {
	dag = client
	return nil
}

type Environment struct {
	Config *EnvironmentConfig

	ID       string
	Name     string
	Source   string
	Worktree string

	Services []*Service

	History History

	mu        sync.Mutex
	container *dagger.Container
}

func (env *Environment) apply(ctx context.Context, name, explanation, output string, newState *dagger.Container) error {
	if _, err := newState.Sync(ctx); err != nil {
		return err
	}

	env.mu.Lock()
	defer env.mu.Unlock()
	revision := &Revision{
		Version:     env.History.LatestVersion() + 1,
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
	env.container = revision.container
	env.History = append(env.History, revision)

	return nil
}

var environments = map[string]*Environment{}

func Create(ctx context.Context, explanation, source, name string) (*Environment, error) {
	env := &Environment{
		ID:     fmt.Sprintf("%s/%s", name, petname.Generate(2, "-")),
		Name:   name,
		Source: source,
		Config: DefaultConfig(),
	}
	if err := env.Config.Load(source); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	worktreePath, err := env.InitializeWorktree(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("failed intializing worktree: %w", err)
	}
	env.Worktree = worktreePath

	container, err := env.buildBase(ctx)
	if err != nil {
		return nil, err
	}

	slog.Info("Creating environment", "id", env.ID, "name", env.Name, "workdir", env.Config.Workdir)

	if err := env.apply(ctx, "Create environment", "Create the environment", "", container); err != nil {
		return nil, err
	}
	environments[env.ID] = env

	if err := env.propagateToWorktree(ctx, "Init env "+name, explanation); err != nil {
		return nil, fmt.Errorf("failed to propagate to worktree: %w", err)
	}

	return env, nil
}

func Open(ctx context.Context, explanation, source, id string) (*Environment, error) {
	// FIXME(aluzzardi): DO NOT USE THIS FUNCTION. It's broken.

	name, _, _ := strings.Cut(id, "/")
	env := &Environment{
		Name:   name,
		ID:     id,
		Source: source,
		Config: DefaultConfig(),
	}
	worktreePath, err := env.InitializeWorktree(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("failed intializing worktree: %w", err)
	}
	env.Worktree = worktreePath

	if err := env.Config.Load(worktreePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Create(ctx, explanation, source, name)
		}
		return nil, err
	}

	container, err := env.buildBase(ctx)
	if err != nil {
		return nil, err
	}
	if err := env.apply(ctx, "Open environment", "Open the environment", "", container); err != nil {
		return nil, err
	}

	environments[env.ID] = env

	return env, nil

	// FIXME(aluzzardi): BROKEN
	// if err := env.loadStateFromNotes(ctx, worktreePath); err != nil {
	// 	return nil, fmt.Errorf("failed to load state from notes: %w", err)
	// }

	// for _, revision := range env.History {
	// 	revision.container = dag.LoadContainerFromID(dagger.ContainerID(revision.State))
	// }
	// if latest := env.History.Latest(); latest != nil {
	// 	env.container = latest.container
	// }
}

func containerWithEnvAndSecrets(container *dagger.Container, envs, secrets []string) (*dagger.Container, error) {
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

func (env *Environment) buildBase(ctx context.Context) (*dagger.Container, error) {
	sourceDir := dag.Host().Directory(env.Worktree, dagger.HostDirectoryOpts{
		NoCache: true,
	})

	container := dag.
		Container().
		From(env.Config.BaseImage).
		WithWorkdir(env.Config.Workdir)

	container, err := containerWithEnvAndSecrets(container, env.Config.Env, env.Config.Secrets)
	if err != nil {
		return nil, err
	}

	for _, command := range env.Config.SetupCommands {
		var err error

		container = container.WithExec([]string{"sh", "-c", command})

		stdout, err := container.Stdout(ctx)
		if err != nil {
			var exitErr *dagger.ExecError
			if errors.As(err, &exitErr) {
				_ = env.addGitNote(ctx,
					fmt.Sprintf("$ %s\nexit %d\nstdout: %s\nstderr: %s\n\n",
						command,
						exitErr.ExitCode, exitErr.Stdout, exitErr.Stderr,
					),
				)
				return nil, fmt.Errorf("setup command failed with exit code %d.\nstdout: %s\nstderr: %s\n%w\n", exitErr.ExitCode, exitErr.Stdout, exitErr.Stderr, err)
			}

			return nil, fmt.Errorf("failed to execute setup command: %w", err)
		}

		_ = env.addGitNote(ctx, fmt.Sprintf("$ %s\n%s\n\n", command, stdout))
	}

	env.Services, err = env.startServices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start services: %w", err)
	}
	for _, service := range env.Services {
		container = container.WithServiceBinding(service.Config.Name, service.svc)
	}

	container = container.WithDirectory(".", sourceDir)

	return container, nil
}

func (env *Environment) UpdateConfig(ctx context.Context, explanation string, newConfig *EnvironmentConfig) error {
	if env.Config.Locked(env.Source) {
		return fmt.Errorf("Environment is locked, no updates allowed. Try to make do with the current environment or ask a human to remove the lock file (%s)", path.Join(env.Source, configDir, lockFile))
	}

	env.Config = newConfig

	// Re-build the base image from the worktree
	container, err := env.buildBase(ctx)
	if err != nil {
		return err
	}

	if err := env.apply(ctx, "Update environment", explanation, "", container); err != nil {
		return err
	}

	return env.propagateToWorktree(ctx, "Update environment "+env.Name, explanation)
}

func Get(idOrName string) *Environment {
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

func List(ctx context.Context, source string) ([]string, error) {
	if _, err := runGitCommand(ctx, source, "rev-parse", "--is-inside-work-tree"); err != nil {
		return nil, fmt.Errorf("cu list only works within git repository, no repo found (or any of the parent directories): .git")
	}

	branches, err := runGitCommand(ctx, source, "for-each-ref", "refs/remotes/"+containerUseRemote, "--format", "%(refname:short)")
	if err != nil {
		return nil, err
	}

	envs := []string{}
	for _, branch := range strings.Split(branches, "\n") {
		env := strings.TrimPrefix(branch, containerUseRemote+"/")
		if !strings.Contains(env, "/") {
			continue
		}
		envs = append(envs, env)
	}

	return envs, nil
}

func (env *Environment) Run(ctx context.Context, explanation, command, shell string, useEntrypoint bool) (string, error) {
	args := []string{}
	if command != "" {
		args = []string{shell, "-c", command}
	}
	newState := env.container.WithExec(args, dagger.ContainerWithExecOpts{
		UseEntrypoint: useEntrypoint,
	})
	stdout, err := newState.Stdout(ctx)
	if err != nil {
		var exitErr *dagger.ExecError
		if errors.As(err, &exitErr) {
			_ = env.addGitNote(ctx,
				fmt.Sprintf("$ %s\nexit %d\nstdout: %s\nstderr: %s\n\n",
					command,
					exitErr.ExitCode, exitErr.Stdout, exitErr.Stderr,
				),
			)
			return fmt.Sprintf("command failed with exit code %d.\nstdout: %s\nstderr: %s", exitErr.ExitCode, exitErr.Stdout, exitErr.Stderr), nil
		}
		return "", err
	}
	_ = env.addGitNote(ctx, fmt.Sprintf("$ %s\n%s\n\n", command, stdout))
	if err := env.apply(ctx, "Run "+command, explanation, stdout, newState); err != nil {
		return "", err
	}

	if err := env.propagateToWorktree(ctx, "Run "+command, explanation); err != nil {
		return "", fmt.Errorf("failed to propagate to worktree: %w", err)
	}

	return stdout, nil
}

func (env *Environment) RunBackground(ctx context.Context, explanation, command, shell string, ports []int, useEntrypoint bool) (EndpointMappings, error) {
	args := []string{}
	if command != "" {
		args = []string{shell, "-c", command}
	}
	serviceState := env.container

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
	}).Start(ctx)
	if err != nil {
		var exitErr *dagger.ExecError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("command failed with exit code %d.\nstdout: %s\nstderr: %s", exitErr.ExitCode, exitErr.Stdout, exitErr.Stderr)
		}
		return nil, err
	}

	_ = env.addGitNote(ctx,
		fmt.Sprintf("$ %s &\n\n", command),
	)

	endpoints := EndpointMappings{}
	for _, port := range ports {
		endpoint := &EndpointMapping{}
		endpoints[port] = endpoint

		// Expose port on the host
		tunnel, err := dag.Host().Tunnel(svc, dagger.HostTunnelOpts{
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

		externalEndpoint, err := tunnel.Endpoint(ctx, dagger.ServiceEndpointOpts{})
		if err != nil {
			return nil, err
		}
		endpoint.External = externalEndpoint

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

func (env *Environment) SetEnv(ctx context.Context, explanation string, envs []string) error {
	state := env.container
	for _, env := range envs {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid environment variable: %s", env)
		}
		state = state.WithEnvVariable(parts[0], parts[1])
	}
	return env.apply(ctx, "Set env "+strings.Join(envs, ", "), explanation, "", state)
}

func (env *Environment) Revert(ctx context.Context, explanation string, version Version) error {
	revision := env.History.Get(version)
	if revision == nil {
		return errors.New("no revisions found")
	}
	if err := env.apply(ctx, "Revert to "+revision.Name, explanation, "", revision.container); err != nil {
		return err
	}
	return env.propagateToWorktree(ctx, "Revert to "+revision.Name, explanation)
}

func (env *Environment) Fork(ctx context.Context, explanation, name string, version *Version) (*Environment, error) {
	revision := env.History.Latest()
	if version != nil {
		revision = env.History.Get(*version)
	}
	if revision == nil {
		return nil, errors.New("version not found")
	}

	forkedEnvironment := &Environment{
		ID:   fmt.Sprintf("%s/%s", name, petname.Generate(2, "-")),
		Name: name,
	}
	if err := forkedEnvironment.apply(ctx, "Fork from "+env.Name, explanation, "", revision.container); err != nil {
		return nil, err
	}
	environments[forkedEnvironment.ID] = forkedEnvironment
	return forkedEnvironment, nil
}

func (env *Environment) Terminal(ctx context.Context) error {
	container := env.container
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
		Cmd: cmd,
	}).Sync(ctx); err != nil {
		return err
	}
	return nil
}

func (env *Environment) Checkpoint(ctx context.Context, target string) (string, error) {
	return env.container.Publish(ctx, target)
}

func (env *Environment) Delete(ctx context.Context) error {
	env.mu.Lock()
	defer env.mu.Unlock()

	if err := env.DeleteWorktree(); err != nil {
		return err
	}

	if err := env.DeleteLocalRemoteBranch(); err != nil {
		return err
	}

	// Remove from global environments map
	delete(environments, env.ID)

	return nil
}
