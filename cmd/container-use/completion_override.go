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
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate the autocompletion script for the specified shell",
		Long: `Generate the autocompletion script for container-use for the specified shell.
See each sub-command's help for details on how to use the generated script.

Use --command-name to generate completions for a different command name (e.g., 'cu').`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateCompletionForBinary(args[0])
		},
	}

	completionCmd.PersistentFlags().StringVar(&commandName, "command-name", "container-use", "Command name to use in completions")

	// Add help subcommands that show usage instructions
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
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
	case "powershell":
		return tempRootCmd.GenPowerShellCompletion(os.Stdout)
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

To load completions for every new session, save the output to your bash completion directory.
Common locations include:
  - Linux: /usr/local/etc/bash_completion.d/{{.command}}
  - macOS: $(brew --prefix)/etc/bash_completion.d/{{.command}}
  - Windows (Git Bash): /usr/share/bash-completion/completions/{{.command}}

Example:
	{{.command}} completion bash > /path/to/bash_completion.d/{{.command}}`,

		"zsh": `Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment,
you will need to enable it. You can execute the following once:
	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:
	source <({{.command}} completion zsh)

To load completions for every new session, save the output to your zsh completion directory.
Common locations include:
  - Linux: /usr/local/share/zsh/site-functions/_{{.command}}
  - macOS: $(brew --prefix)/share/zsh/site-functions/_{{.command}}
  - Custom: Any directory in your $fpath

Example:
	{{.command}} completion zsh > /path/to/zsh/site-functions/_{{.command}}`,

		"fish": `Generate the autocompletion script for the fish shell.

To load completions in your current shell session:
	{{.command}} completion fish | source

To load completions for every new session, save the output to your fish completion directory.
Common locations include:
  - Linux/macOS: ~/.config/fish/completions/{{.command}}.fish
  - Windows: %APPDATA%\fish\completions\{{.command}}.fish

Example:
	{{.command}} completion fish > ~/.config/fish/completions/{{.command}}.fish`,

		"powershell": `Generate the autocompletion script for PowerShell.

To load completions in your current shell session:
	{{.command}} completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output to your PowerShell profile.
Common profile locations include:
  - Windows: $PROFILE (usually %USERPROFILE%\Documents\PowerShell\Microsoft.PowerShell_profile.ps1)
  - Linux/macOS: ~/.config/powershell/Microsoft.PowerShell_profile.ps1

Example:
	{{.command}} completion powershell >> $PROFILE

Note: You may need to create the profile file if it doesn't exist.`,
	}

	template := templates[shell]
	return strings.ReplaceAll(template, "{{.command}}", commandName)
}
