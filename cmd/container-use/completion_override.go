package main

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var commandName string

func init() {
	// Override cobra's default completion command
	completionCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish]",
		Short: "Generate the autocompletion script for the specified shell",
		Long: `Generate the autocompletion script for container-use for the specified shell.
See each sub-command's help for details on how to use the generated script.

Use --command-name to generate completions for a different command name (e.g., 'cu').`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateCompletionForBinary(args[0])
		},
	}

	completionCmd.PersistentFlags().StringVar(&commandName, "command-name", "container-use", "Command name to use in completions")

	// Add help subcommands that show usage instructions
	for _, shell := range []string{"bash", "zsh", "fish"} {
		shell := shell // capture loop variable
		helpCmd := &cobra.Command{
			Use:   shell,
			Short: "Generate the autocompletion script for " + shell,
			Long:  generateHelpText(shell),
			RunE: func(cmd *cobra.Command, args []string) error {
				return generateCompletionForBinary(shell)
			},
		}
		completionCmd.AddCommand(helpCmd)
	}

	rootCmd.AddCommand(completionCmd)
}

func generateCompletionForBinary(shell string) error {
	tempRootCmd := &cobra.Command{
		Use:   commandName,
		Short: rootCmd.Short,
		Long:  rootCmd.Long,
	}

	for _, subCmd := range rootCmd.Commands() {
		if subCmd.Name() != "completion" {
			tempRootCmd.AddCommand(subCmd)
		}
	}

	switch shell {
	case "bash":
		return tempRootCmd.GenBashCompletion(os.Stdout)
	case "zsh":
		return tempRootCmd.GenZshCompletion(os.Stdout)
	case "fish":
		return tempRootCmd.GenFishCompletion(os.Stdout, true)
	}
	return nil
}

func generateHelpText(shell string) string {
	// Generate help text dynamically based on the shell
	// This reduces duplication of the static help text
	templates := map[string]string{
		"bash": `Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:
	source <({{.command}} completion bash)

To load completions for every new session, execute once:
	{{.command}} completion bash > /usr/local/etc/bash_completion.d/{{.command}}`,

		"zsh": `Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment,
you will need to enable it. You can execute the following once:
	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:
	source <({{.command}} completion zsh)

To load completions for every new session, execute once:
	{{.command}} completion zsh > /usr/local/share/zsh/site-functions/_{{.command}}`,

		"fish": `Generate the autocompletion script for the fish shell.

To load completions in your current shell session:
	{{.command}} completion fish | source

To load completions for every new session, execute once:
	{{.command}} completion fish > ~/.config/fish/completions/{{.command}}.fish`,
	}

	template := templates[shell]
	return strings.ReplaceAll(template, "{{.command}}", commandName)
}
