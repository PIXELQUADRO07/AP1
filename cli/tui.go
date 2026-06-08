package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type menuItem string

func (m menuItem) Title() string       { return string(m) }
func (m menuItem) Description() string {
	switch string(m) {
	case "Dashboard":
		return "Live monitoring of clients and credentials"
	case "Status":
		return "Show API and core status"
	case "Profiles":
		return "Display configured access point profiles"
	case "Plugins":
		return "Inspect available plugin modules"
	case "Templates":
		return "List available captive portal templates"
	case "Quit":
		return "Exit the AP1 terminal interface"
	default:
		return ""
	}
}
func (m menuItem) FilterValue() string { return string(m) }

type tickMsg time.Time

type model struct {
	list     list.Model
	viewport viewport.Model
	content  string
	err      error
	ready    bool
	width    int
	height   int
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00ff00"))
	subtitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ffff")).Bold(true)
	infoStyle     = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#7f7f7f"))
	paneStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#5f5f5f")).Padding(1, 1)
	sideStyle     = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#5f5f5f")).Width(30).Padding(1, 1)
	footerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#b0b0b0")).Italic(true)
)

func startTUI() error {
	items := []list.Item{
		menuItem("Dashboard"),
		menuItem("Status"),
		menuItem("Profiles"),
		menuItem("Plugins"),
		menuItem("Templates"),
		menuItem("Quit"),
	}
	delegate := list.NewDefaultDelegate()
	delegate.Styles.NormalTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("#a0e0ff"))
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff")).Background(lipgloss.Color("#005f87"))

	l := list.New(items, delegate, 30, 20)
	l.Title = "AP1 Menu"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	m := model{
		list:    l,
		content: "Select Dashboard to begin live monitoring...",
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return tick()
}

func tick() tea.Cmd {
	return tea.Every(time.Second*2, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(30, m.height-6)
		if !m.ready {
			m.viewport = viewport.New(m.width-38, m.height-8)
			m.viewport.SetContent(m.content)
			m.ready = true
		} else {
			m.viewport.Width = m.width - 38
			m.viewport.Height = m.height - 8
		}

	case tickMsg:
		if sel := m.list.SelectedItem(); sel != nil && sel.FilterValue() == "Dashboard" {
			m.content = m.refreshDashboard()
			m.viewport.SetContent(m.content)
		}
		return m, tick()

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			sel := m.list.SelectedItem()
			if sel == nil {
				return m, nil
			}
			switch sel.FilterValue() {
			case "Dashboard":
				m.content = m.refreshDashboard()
			case "Status":
				m.content = fetchPrettyJSON("/api/status")
			case "Profiles":
				m.content = fetchPrettyJSON("/api/profiles")
			case "Plugins":
				m.content = fetchPrettyJSON("/api/plugins")
			case "Templates":
				m.content = listTemplates()
			case "Quit":
				return m, tea.Quit
			}
			m.viewport.SetContent(m.content)
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) refreshDashboard() string {
	var out strings.Builder
	out.WriteString(subtitleStyle.Render("LIVE DASHBOARD") + "\n\n")

	// 1. Get Portal Status
	statusB, err := get("/api/portal/status")
	if err == nil {
		var status map[string]interface{}
		json.Unmarshal(statusB, &status)
		running := "STOPPED"
		if r, ok := status["running"].(bool); ok && r {
			running = "RUNNING"
		}
		out.WriteString(fmt.Sprintf("Portal Status: %s\n", running))
	}

	// 2. Get Credentials
	credsB, err := get("/api/portal/credentials")
	if err == nil {
		var creds []map[string]interface{}
		json.Unmarshal(credsB, &creds)
		out.WriteString(fmt.Sprintf("\nCaptured Credentials (%d):\n", len(creds)))
		out.WriteString(strings.Repeat("-", m.width-45) + "\n")
		// Show last 10
		start := 0
		if len(creds) > 10 {
			start = len(creds) - 10
		}
		for i := len(creds) - 1; i >= start; i-- {
			c := creds[i]
			out.WriteString(fmt.Sprintf("[%s] %-10v | %-10v | %v\n",
				c["timestamp"], c["login"], c["password"], c["ip"]))
		}
	}

	return out.String()
}

func fetchPrettyJSON(path string) string {
	b, err := get(path)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	var data interface{}
	if err := json.Unmarshal(b, &data); err != nil {
		return string(b)
	}
	t, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return string(b)
	}
	return string(t)
}

func listTemplates() string {
	candidates := []string{"../config/templates", "./config/templates", "config/templates"}
	var dir string
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			dir = c
			break
		}
	}
	if dir == "" {
		return "config/templates not found"
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	var out strings.Builder
	out.WriteString("Available templates:\n")
	for _, f := range files {
		if f.IsDir() {
			out.WriteString("- " + f.Name() + "\n")
		}
	}
	return out.String()
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	header := titleStyle.Render("AP1 TUI") + " " + subtitleStyle.Render("[WiFi Pumpkin style terminal]")
	footer := footerStyle.Render("↑/↓ Navigate  •  Enter Select  •  q Quit")

	left := sideStyle.Render(m.list.View())
	right := paneStyle.
		Width(m.width - 38).
		Height(m.height - 8).
		Render(m.viewport.View())

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}
