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
}

// Available agents
var agents = []Agent{
	{
		Key:         "claude",
		Name:        "Claude Code",
		Description: "Anthropic's Claude Code",
	},
	{
		Key:         "goose",
		Name:        "Goose",
		Description: "an open source, extensible AI agent that goes beyond code suggestions",
	},
	{
		Key:         "cursor",
		Name:        "Cursor",
		Description: "AI-powered code editor",
	},
	{
		Key:         "codex",
		Name:        "OpenAI Codex",
		Description: "OpenAI's lightweight coding agent that runs in your terminal",
	},
	{
		Key:         "amazonq",
		Name:        "Amazon Q Developer",
		Description: "Amazon's agentic chat experience in your terminal",
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
		case "ctrl+c", "q", "esc":
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
	default:
		return m, nil
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
		Margin(1, 0).
		Bold(true)

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true).
		Margin(1, 0, 0, 0)

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#F25D94")).
		Padding(0, 1).
		Bold(true)

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575")).
		Padding(0, 1)

	descriptionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		Padding(0, 1, 0, 3).
		Italic(true)

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		Margin(1, 0, 0, 0)

	// Build the view
	var s strings.Builder

	// Title
	s.WriteString(titleStyle.Render("🛠️  Container Use Configuration"))
	s.WriteString("\n")

	// Header
	s.WriteString(headerStyle.Render("Select an agent to configure:"))
	s.WriteString("\n\n")

	// Agent list TODO: filter or sort agents based on if they are installed (ConfigurableAgent.isInstalled())
	for i, agent := range agents {
		cursor := "  " // not selected
		if m.cursor == i {
			cursor = "▶ " // selected
		}

		agentLine := fmt.Sprintf("%s%s", cursor, agent.Name)
		if m.cursor == i {
			s.WriteString(selectedStyle.Render(agentLine))
		} else {
			s.WriteString(normalStyle.Render(agentLine))
		}

		s.WriteString("\n")

		// Show description for selected item
		if m.cursor == i {
			s.WriteString(descriptionStyle.Render(agent.Description))
			s.WriteString("\n")
		}
	}

	// Footer
	s.WriteString("\n")
	s.WriteString(footerStyle.Render("Use ↑/↓ or j/k to navigate • Enter/Space to select • q/Ctrl+C/Esc to quit"))

	return s.String()
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
