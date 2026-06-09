package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type menuItem string

func (m menuItem) Title() string { return string(m) }
func (m menuItem) Description() string {
	switch string(m) {
	case "Dashboard":
		return "Live monitoring of clients and credentials"
	case "System Status":
		return "Core services and network interfaces state"
	case "Traffic Analyzer":
		return "Real-time packet inspection and flow graphs"
	case "AP Profiles":
		return "Wireless configuration profiles"
	case "Plugins & Proxies":
		return "Active modules and traffic interceptors"
	case "Quit":
		return "Exit the AP1 terminal interface"
	default:
		return ""
	}
}
func (m menuItem) FilterValue() string { return string(m) }

type tickMsg time.Time

type model struct {
	list        list.Model
	viewport    viewport.Model
	content     string
	err         error
	ready       bool
	width       int
	height      int
	trafficData []int
	maxTraffic  int
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
	infoStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#9090ff"))
	graphStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff87"))
)

func startTUI() error {
	items := []list.Item{
		menuItem("Dashboard"),
		menuItem("System Status"),
		menuItem("Traffic Analyzer"),
		menuItem("AP Profiles"),
		menuItem("Plugins & Proxies"),
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
		list:        l,
		content:     "AP1 Orchestrator initializing...",
		trafficData: make([]int, 40),
		maxTraffic:  10,
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
		// Update traffic graph data
		m.updateTrafficStats()

		if sel := m.list.SelectedItem(); sel != nil {
			switch sel.FilterValue() {
			case "Dashboard":
				m.content = m.refreshDashboard()
			case "Traffic Analyzer":
				m.content = m.refreshTrafficAnalyzer()
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
			case "Traffic Analyzer":
				m.content = m.refreshTrafficAnalyzer()
			case "AP Profiles":
				m.content = fetchPrettyJSON("/api/profiles")
			case "Plugins & Proxies":
				m.content = m.refreshModules()
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

func (m *model) updateTrafficStats() {
	// Shift data
	copy(m.trafficData, m.trafficData[1:])

	// Get real traffic count from API
	b, err := get("/api/traffic?limit=10")
	newVal := 0
	if err == nil {
		var traffic []interface{}
		json.Unmarshal(b, &traffic)
		newVal = len(traffic)
	} else {
		// Mock data if API fails
		newVal = rand.Intn(5)
	}

	m.trafficData[len(m.trafficData)-1] = newVal
	if newVal > m.maxTraffic {
		m.maxTraffic = newVal
	}
}

func (m model) drawGraph(width int, height int) string {
	if m.maxTraffic == 0 {
		m.maxTraffic = 1
	}
	var out strings.Builder

	for h := height; h > 0; h-- {
		for i := 0; i < len(m.trafficData); i++ {
			val := m.trafficData[i]
			normalized := (val * height) / m.maxTraffic
			if normalized >= h {
				out.WriteString(graphStyle.Render("┃"))
			} else {
				out.WriteString(" ")
			}
		}
		out.WriteString("\n")
	}

	// Bottom line
	out.WriteString(strings.Repeat("━", len(m.trafficData)))
	out.WriteString("\n")
	return out.String()
}

func fetchPrettyJSON(path string) string {
	b, err := get(path)
	if err != nil {
		return fmt.Sprintf("Error fetching %s: %v", path, err)
	}

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, b, "", "  "); err != nil {
		return string(b)
	}

	return pretty.String()
}

func (m model) refreshDashboard() string {
	var out strings.Builder
	out.WriteString(headerStyle.Render(" NETWORK ORCHESTRATOR DASHBOARD "))
	out.WriteString("\n\n")

	// 1. Get Portal Status
	statusB, err := get("/api/status")
	if err == nil {
		var status map[string]interface{}
		json.Unmarshal(statusB, &status)
		cfg, _ := status["config"].(map[string]interface{})
		activeProfile := fmt.Sprint(cfg["active_profile"])
		net, _ := cfg["network"].(map[string]interface{})
		iface := fmt.Sprint(net["default_interface"])

		out.WriteString(fmt.Sprintf("Active Profile: %s\n", subtitleStyle.Render(activeProfile)))
		out.WriteString(fmt.Sprintf("Interface:      %s\n", iface))

		if pid, s := findServiceProcess("hostapd"); s == "running" {
			out.WriteString(fmt.Sprintf("Service AP:     %s\n", successStyle.Render("RUNNING (PID "+pid+")")))
		} else {
			out.WriteString(fmt.Sprintf("Service AP:     %s\n", warnStyle.Render("OFFLINE")))
		}
	}

	out.WriteString("\n")
	out.WriteString(subtitleStyle.Render("LIVE FLOW MONITOR"))
	out.WriteString("\n")
	out.WriteString(m.drawGraph(40, 5))

	// 2. Get Credentials
	credsB, err := get("/api/portal/credentials")
	if err == nil {
		var creds []map[string]interface{}
		json.Unmarshal(credsB, &creds)
		out.WriteString(fmt.Sprintf("\nCaptured Events (%d):\n", len(creds)))
		out.WriteString(strings.Repeat("━", m.width-45))
		out.WriteString("\n")

		start := 0
		if len(creds) > 5 {
			start = len(creds) - 5
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

func (m model) refreshTrafficAnalyzer() string {
	var out strings.Builder
	out.WriteString(headerStyle.Render(" ADVANCED TRAFFIC ANALYZER "))
	out.WriteString("\n\n")

	out.WriteString(subtitleStyle.Render("PACKET THROUGHPUT (PPS)"))
	out.WriteString("\n")
	out.WriteString(m.drawGraph(40, 8))
	out.WriteString("\n")

	b, err := get("/api/traffic?limit=15")
	if err == nil {
		var traffic []map[string]interface{}
		json.Unmarshal(b, &traffic)

		out.WriteString(subtitleStyle.Render("RECENT FLOWS"))
		out.WriteString("\n")
		out.WriteString(fmt.Sprintf("%-20s | %-10s | %s\n", "DESTINATION", "PROTO", "INFO"))
		out.WriteString(strings.Repeat("─", m.width-40))
		out.WriteString("\n")

		for _, t := range traffic {
			dest := fmt.Sprint(t["destination"])
			if len(dest) > 20 {
				dest = dest[:17] + "..."
			}
			proto := fmt.Sprint(t["protocol"])
			info := fmt.Sprint(t["info"])
			if len(info) > 40 {
				info = info[:37] + "..."
			}

			protoColor := successStyle
			if proto == "DNS" {
				protoColor = subtitleStyle
			}

			out.WriteString(fmt.Sprintf("%-20s | %s | %s\n",
				dest,
				protoColor.Render(fmt.Sprintf("%-10s", proto)),
				info))
		}
	}

	return out.String()
}

func (m model) refreshStatus() string {
	var out strings.Builder
	out.WriteString(headerStyle.Render(" SYSTEM & INTERFACES "))
	out.WriteString("\n\n")

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

	out.WriteString("\n")
	out.WriteString(headerStyle.Render(" BACKGROUND JOBS "))
	out.WriteString("\n")
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
	out.WriteString(headerStyle.Render(" ACTIVE PLUGINS "))
	out.WriteString("\n")
	b, _ := get("/api/plugins")
	var plugins []map[string]interface{}
	json.Unmarshal(b, &plugins)
	for _, p := range plugins {
		enabled := warnStyle.Render("NO")
		if p["enabled"].(bool) {
			enabled = successStyle.Render("YES")
		}
		out.WriteString(fmt.Sprintf(" [%s] %-15s | %s\n", enabled, p["name"], p["description"]))
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
