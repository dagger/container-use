package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Agent represents an agent configuration option
type Agent struct {
	Key         string
	Name        string
	Description string
	Icon        string
}

// Available agents
var agents = []Agent{
	{
		Key:         "claude",
		Name:        "Claude Code",
		Description: "Anthropic's Claude Code with MCP support",
		Icon:        "🤖",
	},
	{
		Key:         "goose",
		Name:        "Goose",
		Description: "AI coding assistant with extension support",
		Icon:        "🪿",
	},
	{
		Key:         "cursor",
		Name:        "Cursor",
		Description: "AI-powered code editor",
		Icon:        "⚡",
	},
	{
		Key:         "codex",
		Name:        "OpenAI Codex",
		Description: "OpenAI's code generation model",
		Icon:        "🧠",
	},
	{
		Key:         "amazonq",
		Name:        "Amazon Q Developer",
		Description: "Amazon's AI coding assistant",
		Icon:        "🌟",
	},
}

// AgentSelectorModel represents the bubbletea model for agent selection
type AgentSelectorModel struct {
	cursor   int
	selected string
	quit     bool
}

// InitialModel creates the initial model for agent selection
func InitialModel() AgentSelectorModel {
	return AgentSelectorModel{}
}

// Init initializes the model
func (m AgentSelectorModel) Init() tea.Cmd {
	return nil
}

// Update handles incoming messages and updates the model
func (m AgentSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quit = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(agents)-1 {
				m.cursor++
			}
		case "enter", " ":
			m.selected = agents[m.cursor].Key
			m.quit = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the interface
func (m AgentSelectorModel) View() string {
	if m.quit {
		return ""
	}

	// Styles
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Bold(true)

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#F25D94")).
		Padding(0, 1).
		Bold(true)

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#F25D94")).
		Padding(0, 1).
		Bold(true)

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Padding(0, 1)

	descriptionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		Padding(0, 1)

	// Header
	s := titleStyle.Render("🛠️  Container Use Configuration")
	s += "\n\n"
	s += headerStyle.Render("Select an agent to configure:")
	s += "\n\n"

	// Agent list
	for i, agent := range agents {
		cursor := " " // not selected
		if m.cursor == i {
			cursor = "▶" // selected
		}

		agentLine := fmt.Sprintf("%s %s %s", cursor, agent.Icon, agent.Name)
		if m.cursor == i {
			s += selectedStyle.Render(agentLine)
		} else {
			s += normalStyle.Render(agentLine)
		}

		s += "\n"
		
		// Show description for selected item
		if m.cursor == i {
			s += descriptionStyle.Render(fmt.Sprintf("   %s", agent.Description))
			s += "\n"
		}
	}

	// Footer
	s += "\n"
	s += descriptionStyle.Render("Use ↑/↓ or j/k to navigate, Enter/Space to select, q to quit")

	return s
}

// RunAgentSelector runs the interactive agent selector and returns the selected agent key
func RunAgentSelector() (string, error) {
	p := tea.NewProgram(InitialModel())
	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("error running agent selector: %w", err)
	}

	m := finalModel.(AgentSelectorModel)
	if m.selected == "" {
		return "", fmt.Errorf("no agent selected")
	}

	return m.selected, nil
}
