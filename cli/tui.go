package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type menuItem string

func (m menuItem) Title() string { return string(m) }
func (m menuItem) Description() string {
	switch string(m) {
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

type model struct {
	list    list.Model
	content string
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00ff00"))
	subtitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ffff")).Bold(true)
	infoStyle     = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#7f7f7f"))
	paneStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#5f5f5f")).Padding(1, 1).Width(70)
	sideStyle     = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#5f5f5f")).Width(30).Padding(1, 1)
	footerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#b0b0b0")).Italic(true)
)

func startTUI() error {
	items := []list.Item{menuItem("Status"), menuItem("Profiles"), menuItem("Plugins"), menuItem("Templates"), menuItem("Quit")}
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles = list.NewDefaultItemStyles()
	delegate.Styles.NormalTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("#a0e0ff"))
	delegate.Styles.NormalDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("#909090"))
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff")).Background(lipgloss.Color("#005f87"))
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("#d0d0d0"))
	delegate.Styles.DimmedTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("#5f7f8f"))
	delegate.Styles.FilterMatch = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffdf00"))

	l := list.New(items, delegate, 30, 16)
	l.Title = "AP1 Console"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)

	m := model{list: l, content: "Use ↑/↓ to navigate, Enter to open, q to quit."}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			sel := m.list.SelectedItem()
			if sel == nil {
				return m, nil
			}
			switch strings.ToLower(sel.FilterValue()) {
			case "status":
				m.content = fetchPrettyJSON("/api/status")
			case "profiles":
				m.content = fetchPrettyJSON("/api/profiles")
			case "plugins":
				m.content = fetchPrettyJSON("/api/plugins")
			case "templates":
				m.content = listTemplates()
			case "quit":
				return m, tea.Quit
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
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
	header := titleStyle.Render("AP1 TUI") + " " + subtitleStyle.Render("[WiFi Pumpkin style terminal]")
	footer := footerStyle.Render("↑/↓ Navigate  •  Enter Open  •  q Quit")
	left := sideStyle.Render(m.list.View())
	right := paneStyle.Render(m.content)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}
