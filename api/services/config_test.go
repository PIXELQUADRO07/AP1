package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigValidates(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "global.yaml")
	configContents := `app:
  name: AP1
  environment: production
  api_url: http://127.0.0.1:8001
  core_url: http://127.0.0.1:8081
network:
  default_interface: wlan0
  captive_portal: true
  portal_ip: 192.168.50.1
  portal_port: 80
  portal_fallback_port: 8000
  dns_ip: 192.168.50.1
  subnet: 24
  template: DarkLogin
logging:
  level: info
  credentials_log: /tmp/credentials.log
active_profile: default
profiles:
  - name: default
    ssid: FreeWifi
    password: ap1password
    channel: 1
    mode: g
    dhcp_enabled: true
    security: open
`
	if err := os.WriteFile(configPath, []byte(configContents), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ActiveProfile != "default" {
		t.Fatalf("expected active profile default, got %q", cfg.ActiveProfile)
	}
}
