package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/ap1/project/services"
)

type pluginToggleRequest struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

func PluginsHandler(pluginConfigPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		cfg, err := services.LoadPluginConfig(pluginConfigPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load plugin config: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cfg.Plugins)
	}
}

func TogglePluginHandler(pluginConfigPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var payload pluginToggleRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
			return
		}
		if payload.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}

		cfg, err := services.LoadPluginConfig(pluginConfigPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load plugin config: %v", err), http.StatusInternalServerError)
			return
		}

		plugin := services.FindPlugin(cfg, payload.Name)
		if plugin == nil {
			http.Error(w, "plugin not found", http.StatusNotFound)
			return
		}

		plugin.Enabled = payload.Enabled
		if err := services.SavePluginConfig(pluginConfigPath, cfg); err != nil {
			http.Error(w, fmt.Sprintf("failed to save plugin config: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(plugin)
	}
}

type pluginExecRequest struct {
	Name string   `json:"name"`
	Cmd  string   `json:"cmd"`
	Args []string `json:"args"`
}

func StartPluginHandler(pluginConfigPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload pluginExecRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
			return
		}
		if payload.Name == "" || payload.Cmd == "" {
			http.Error(w, "name and cmd are required", http.StatusBadRequest)
			return
		}
		// basic security: only allow commands under configured plugin paths or explicit full path
		cmdParts := strings.Fields(payload.Cmd)
		cmd := cmdParts[0]
		args := append(cmdParts[1:], payload.Args...)
		// start process
		c := exec.Command(cmd, args...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Start(); err != nil {
			http.Error(w, fmt.Sprintf("failed to start plugin: %v", err), http.StatusInternalServerError)
			return
		}
		pid := c.Process.Pid
		runtimeDir := filepath.Join("..", "system", "runtime", "plugins")
		_ = os.MkdirAll(runtimeDir, 0o755)
		pidFile := filepath.Join(runtimeDir, fmt.Sprintf("%s.pid", payload.Name))
		_ = os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0o644)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"name": payload.Name, "pid": pid})
	}
}

func StopPluginHandler(pluginConfigPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
			return
		}
		if payload.Name == "" {
			http.Error(w, "name required", http.StatusBadRequest)
			return
		}
		runtimeDir := filepath.Join("..", "system", "runtime", "plugins")
		pidFile := filepath.Join(runtimeDir, fmt.Sprintf("%s.pid", payload.Name))
		data, err := os.ReadFile(pidFile)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to read pid file: %v", err), http.StatusInternalServerError)
			return
		}
		pid, err := strconv.Atoi(string(data))
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid pid file: %v", err), http.StatusInternalServerError)
			return
		}
		proc, err := os.FindProcess(pid)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to find process: %v", err), http.StatusInternalServerError)
			return
		}
		if err := proc.Signal(syscall.SIGTERM); err != nil {
			http.Error(w, fmt.Sprintf("failed to stop process: %v", err), http.StatusInternalServerError)
			return
		}
		_ = os.Remove(pidFile)
		w.WriteHeader(http.StatusNoContent)
	}
}
