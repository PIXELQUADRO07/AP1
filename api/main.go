package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ap1/project/routes"
	"github.com/ap1/project/server"
	"github.com/ap1/project/services"
)

func findExistingFile(candidates ...string) (string, error) {
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if info, err := os.Stat(abs); err == nil && !info.IsDir() {
			return abs, nil
		}
	}
	return "", fmt.Errorf("no valid path found among candidates: %v", candidates)
}

func defaultConfigPaths() (string, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}
	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)

	configCandidates := []string{
		filepath.Join(cwd, "config", "global.yaml"),
		filepath.Join(cwd, "..", "config", "global.yaml"),
		filepath.Join(exeDir, "config", "global.yaml"),
		filepath.Join(exeDir, "..", "config", "global.yaml"),
		filepath.Join(exeDir, "..", "..", "config", "global.yaml"),
	}
	pluginCandidates := []string{
		filepath.Join(cwd, "config", "plugins.yaml"),
		filepath.Join(cwd, "..", "config", "plugins.yaml"),
		filepath.Join(exeDir, "config", "plugins.yaml"),
		filepath.Join(exeDir, "..", "config", "plugins.yaml"),
		filepath.Join(exeDir, "..", "..", "plugins.yaml"),
	}

	configPath, err := findExistingFile(configCandidates...)
	if err != nil {
		return "", "", err
	}
	pluginConfigPath, err := findExistingFile(pluginCandidates...)
	if err != nil {
		return "", "", err
	}
	return configPath, pluginConfigPath, nil
}

func main() {
	var configPath string
	var pluginConfigPath string
	var addr string
	var tlsCert string
	var tlsKey string

	flag.StringVar(&configPath, "config", "", "Path to global YAML config")
	flag.StringVar(&pluginConfigPath, "plugins", "", "Path to plugin YAML config")
	flag.StringVar(&addr, "addr", "", "Listen address for the API server")
	flag.StringVar(&tlsCert, "tls-cert", "", "Path to TLS certificate file")
	flag.StringVar(&tlsKey, "tls-key", "", "Path to TLS private key file")
	flag.Parse()

	if configPath == "" {
		configPath = os.Getenv("AP1_CONFIG_PATH")
	}
	if pluginConfigPath == "" {
		pluginConfigPath = os.Getenv("AP1_PLUGIN_CONFIG_PATH")
	}

	if configPath == "" || pluginConfigPath == "" {
		defaultConfigPath, defaultPluginPath, err := defaultConfigPaths()
		if err != nil {
			log.Fatalf("failed to resolve config paths: %v", err)
		}
		if configPath == "" {
			configPath = defaultConfigPath
		}
		if pluginConfigPath == "" {
			pluginConfigPath = defaultPluginPath
		}
	}

	if addr == "" {
		addr = os.Getenv("AP1_API_ADDR")
	}
	if addr == "" {
		addr = ":8001"
	}

	services.InitDB("../system/runtime/ap1.db")
	services.StartLogWatcher("../system/runtime/portal_credentials.log")

	cfg, err := services.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	coreURL := os.Getenv("AP1_CORE_URL")
	if coreURL == "" {
		coreURL = cfg.App.CoreURL
	}
	if coreURL == "" {
		coreURL = "http://127.0.0.1:8081"
	}

	configDir := filepath.Dir(configPath)
	templateName := cfg.Network.Template
	if templateName == "" {
		templateName = "DarkLogin"
	}
	templateDir := filepath.Join(configDir, "templates", templateName)

	logPath := cfg.Logging.CredentialsLog
	if logPath == "" {
		logPath = "/var/log/ap1/credentials.json"
	}

	portalIP := cfg.Network.PortalIP
	if portalIP == "" {
		portalIP = "192.168.50.1"
	}

	ps := server.NewPortalServer(templateDir, logPath, portalIP)

	router := routes.NewRouter(coreURL, cfg, configPath, pluginConfigPath, ps)

	log.Printf("Starting AP1 API server on %s", addr)
	log.Printf("Using config: %s", configPath)
	log.Printf("Using plugin config: %s", pluginConfigPath)
	if tlsCert != "" || tlsKey != "" {
		if tlsCert == "" || tlsKey == "" {
			log.Fatal("both --tls-cert and --tls-key are required when using HTTPS")
		}
		log.Printf("Starting AP1 API server with TLS on %s", addr)
		log.Fatal(http.ListenAndServeTLS(addr, tlsCert, tlsKey, router))
	}
	log.Fatal(http.ListenAndServe(addr, router))
}
