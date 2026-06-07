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
	"strconv"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

const (
	defaultAPIBase = "http://127.0.0.1:8080"
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
}

func printSection(title string) {
	fmt.Println(colorText(ansiCyan, title))
	fmt.Println(strings.Repeat("=", len(title)))
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

var banners = []string{
	`
  ____   ____  _   _ _____
 |  _ \ / ___|| | | | ____|
 | |_) | |  _ | | | |  _|
 |  __/| |_| || |_| | |___
 |_|    \____| \___/|_____|
`,
	`
    ___   ____  ____  ______
   / _ \ / ___||  _ \|  ____|
  | | | | |  _ | |_) |  _|
  | |_| | |_| ||  _ <| |___
   \___/ \____||_| \_\_____|
`,
	`
   ___    ____  _   _ _____
  / _ \  / ___|| | | | ____|
 | | | | \___ \| | | |  _|
 | |_| |  ___) | |_| | |___
  \___/  |____/ \___/|_____|
`,
	`
  ____   ____  _____  _   _
 |  _ \ / ___|| ____|| \ | |
 | |_) | |  _ |  _|  |  \| |
 |  _ <| |_| || |___ | |\  |
 |_| \_\\____||_____||_| \_|
`,
}
var startBanners = []string{
	`
   _____ _   _  _____ _____ _   _ ____  
  |_   _| | | |/ ____|_   _| \ | |  _ \ 
    | | | | | | (___   | | |  \| | |_) |
    | | | | |\___ \  | | | .  |  _ < 
   _| |_| |_| |____) |_| |_| |\  | |_) |
  |_____|\___/|_____/|_____|_| \_|____/ 
`,
	`
   _____ _   _ _____ _____  ______ _   _ 
  / ____| \ | |_   _/ ____|/ ____| \ | |
 | (___ |  \| | | || |    | |    |  \| |
  \___ \| .  | | || |    | |    | .  |
  ____) | |\  |_| || |____| |____| |\  |
 |_____/|_| \_|_____|
\_____|\_____|_| \_|
`,
}
var interactiveBanners = []string{
	`
  _____ _   _ _____ _____  _____ ___ _____ 
 |_   _| \ | |_   _/ ____|/ ____|__ \_   _|
   | | |  \| | | || |  __| |       ) || |  
   | | | .  | | || | |_ | |      / / | |  
  _| |_| |\  |_| || |__| | |____ / /_ _| |_ 
 |_____|_| \_|_____|
\_____|_____|_____|_____|
`,
	`
   _____ ___  _   _ _____ _____ _____ _____ 
  / ____/ _ \| \ | |_   _/ ____|_   _/ ____|
 | |   | | | |  \| | | || |      | || |     
 | |   | | | | .  | | || |      | || |     
 | |___| |_| | |\  |_| || |____ _| || |____ 
  \_____\___/|_| \_|_____|
\_____|
\_____|_____|
`,
}
var bannerTaglines = []string{
	"edge-aware captive portal orchestrator",
	"network trickery with a friendly face",
	"AP management for modern pentesting",
	"control APs, portals and payloads",
}

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

func cmdClients(args []string) {
	fmt.Println("clients: feature not implemented yet. Use AP1 server client tracking when available.")
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
	fmt.Printf("  service: %v\n", status["service"])
	fmt.Printf("  version: %v\n", status["version"])
	if cfg, ok := status["config"].(map[string]interface{}); ok {
		fmt.Println("  config:")
		fmt.Printf("    name: %v\n", cfgValue(cfg, []string{"app", "name"}))
		fmt.Printf("    environment: %v\n", cfgValue(cfg, []string{"app", "environment"}))
		fmt.Printf("    api_url: %v\n", cfgValue(cfg, []string{"app", "api_url"}))
		fmt.Printf("    core_url: %v\n", cfgValue(cfg, []string{"app", "core_url"}))
		fmt.Printf("    interface: %v\n", cfgValue(cfg, []string{"network", "default_interface"}))
		fmt.Printf("    portal_ip: %v\n", cfgValue(cfg, []string{"network", "portal_ip"}))
		fmt.Printf("    dns_ip: %v\n", cfgValue(cfg, []string{"network", "dns_ip"}))
		fmt.Printf("    captive_portal: %v\n", cfgValue(cfg, []string{"network", "captive_portal"}))
		fmt.Printf("    active_profile: %v\n", cfgValue(cfg, []string{"active_profile"}))
	}
	if plugins, ok := status["plugins"].([]interface{}); ok {
		fmt.Printf("  plugins: %d enabled\n", len(plugins))
	}
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
	showBanner("start")
	fmt.Println("starting AP and captive portal...")
	if _, err := post("/api/system/hostapd/start", nil); err != nil {
		fmt.Fprintln(os.Stderr, "hostapd start failed:", err)
	}
	if _, err := post("/api/system/dnsmasq/start", nil); err != nil {
		fmt.Fprintln(os.Stderr, "dnsmasq start failed:", err)
	}
	if _, err := post("/api/portal/start", nil); err != nil {
		fmt.Fprintln(os.Stderr, "portal start failed:", err)
	}
}

func cmdStop(args []string) {
	fmt.Println("stopping AP and captive portal...")
	if _, err := post("/api/portal/stop", nil); err != nil {
		fmt.Fprintln(os.Stderr, "portal stop failed:", err)
	}
	if _, err := post("/api/system/hostapd/stop", nil); err != nil {
		fmt.Fprintln(os.Stderr, "hostapd stop failed:", err)
	}
	if _, err := post("/api/system/dnsmasq/stop", nil); err != nil {
		fmt.Fprintln(os.Stderr, "dnsmasq stop failed:", err)
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
	fmt.Println("jobs: background job listing is not implemented yet.")
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
	fmt.Println("dhcpconf: command not implemented yet.")
}

func cmdDhcpmode(args []string) {
	fmt.Println("dhcpmode: command not implemented yet.")
}

func cmdUpdate(args []string) {
	fmt.Println("update: command deprecated.")
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
		for key, value := range status {
			fmt.Printf("%s: %v\n", strings.Title(key), value)
		}
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
	fmt.Println(colorText(ansiCyan, banner))
	var tagline string
	if mode == "start" {
		tagline = "starting AP1..."
	} else if mode == "interactive" {
		tagline = "interactive AP1 console"
	} else {
		tagline = randomTagline()
	}
	fmt.Println(colorText(ansiGreen, ansiBold+"AP1 - "+tagline+ansiReset))
}

func usage() {
	showBanner("")
	fmt.Println(colorText(ansiYellow, "Usage:"))
	fmt.Println("  ap1-cli [--api URL] <command> [args...]")
	fmt.Println()
	fmt.Println(colorText(ansiYellow, "Core commands:"))
	fmt.Println("  help                         Show this help")
	fmt.Println("  status                       Show API/core status")
	fmt.Println("  health                       Check API health endpoint")
	fmt.Println("  config                       Dump current loaded config")
	fmt.Println("  version                      Show CLI version")
	fmt.Println("  banner                       Show a random AP1 banner")
	fmt.Println("  clear                        Clear the terminal and show a new banner")
	fmt.Println("  clients                      Show connected clients")
	fmt.Println()
	fmt.Println(colorText(ansiYellow, "AP management:"))
	fmt.Println("  ap [status|start|stop]       Manage the access point")
	fmt.Println("  start                        Start AP, services and portal")
	fmt.Println("  stop                         Stop AP, services and portal")
	fmt.Println("  profiles list                List all AP profiles")
	fmt.Println("  profiles select <name>       Activate a profile")
	fmt.Println("  profiles create <name> <ssid> <password> <channel> <mode> <dhcp>")
	fmt.Println("  profiles update <name> <ssid> <password> <channel> <mode> <dhcp>")
	fmt.Println("  profiles delete <name>       Remove a profile")
	fmt.Println()
	fmt.Println(colorText(ansiYellow, "Proxy / plugin / session:"))
	fmt.Println("  set <key> <value>            Set runtime configuration")
	fmt.Println("  unset <key>                  Unset runtime configuration")
	fmt.Println("  ignore <component>           Ignore log output for a component")
	fmt.Println("  restore <component>          Restore log output for a component")
	fmt.Println("  info <proxy|plugin>          Show proxy/plugin info")
	fmt.Println("  jobs                         Show background jobs")
	fmt.Println("  mode [set <static|docker>]   Show or set wireless mode")
	fmt.Println("  plugins list                 List plugins")
	fmt.Println("  plugins toggle <name> <on|off>")
	fmt.Println("  plugins start <name> <cmd> [args...]")
	fmt.Println("  plugins stop <name>")
	fmt.Println("  proxies [list]               List supported proxy backends")
	fmt.Println("  show <modules|plugins|proxies|commands>")
	fmt.Println("  search <term>                Search available CLI commands")
	fmt.Println("  use <module>                 Select a module")
	fmt.Println("  dump credentials             Dump captured portal credentials")
	fmt.Println("  banner                       Show a random AP1 banner")
	fmt.Println("  clear                        Clear the terminal and show a new banner")
	fmt.Println("  dhcpconf                     DHCP server configuration helpers")
	fmt.Println("  dhcpmode                     DHCP mode helpers")
	fmt.Println("  update                       Deprecated update command")
	fmt.Println()
	fmt.Println(colorText(ansiYellow, "Network / portal:"))
	fmt.Println("  portal status                Show captive portal status")
	fmt.Println("  portal credentials           Show captured portal credentials")
	fmt.Println("  portal start                 Start the captive portal server")
	fmt.Println("  portal stop                  Stop the captive portal server")
	fmt.Println("  templates list               List available portal templates")
	fmt.Println("  templates import <src_dir>   Import templates from another repo")
	fmt.Println("  firewall apply [iface] [portal_ip]  Apply captive portal firewall rules")
	fmt.Println("  firewall clear [iface]            Clear captive portal firewall rules")
	fmt.Println("  interface configure <iface> <ip> <subnet>  Configure a network interface")
	fmt.Println("  system <service> <action>    Manage hostapd/dnsmasq")
	fmt.Println("  recon scan [iface]           Scan Wi-Fi networks on iface")
	fmt.Println()
	fmt.Println(colorText(ansiYellow, "Environment:"))
	fmt.Println("  AP1_API_URL                 API server URL")
	fmt.Println("  AP1_USE_DOCKER              Use docker compose exec for requests")
}

func main() {
	var dockerFlag bool
	var apiURL string
	flag.BoolVar(&dockerFlag, "docker", false, "Use docker compose exec to reach services")
	flag.StringVar(&apiURL, "api", "", "URL of the AP1 API server")
	flag.Usage = usage
	flag.Parse()

	if apiURL != "" {
		apiBase = apiURL
	} else if envURL := os.Getenv("AP1_API_URL"); envURL != "" {
		apiBase = envURL
	}
	dockerMode = dockerFlag || (isDockerComposeAvailable() && os.Getenv("AP1_USE_DOCKER") == "1")

	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "ERROR: AP1 CLI must be run as root.")
		fmt.Fprintln(os.Stderr, "Please run with sudo: sudo ap1-cli <command>")
		os.Exit(1)
	}

	if flag.NArg() == 0 {
		usage()
		os.Exit(0)
	}

	cmd := flag.Arg(0)
	args := flag.Args()[1:]
	switch cmd {
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
	case "templates":
		cmdTemplates(args)
	case "version":
		cmdVersion()
	case "interactive":
		showBanner("interactive")
		startREPL()
	case "tui":
		if err := startTUI(); err != nil {
			fmt.Fprintln(os.Stderr, "tui error:", err)
			os.Exit(1)
		}
	case "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		usage()
		os.Exit(1)
	}
}
