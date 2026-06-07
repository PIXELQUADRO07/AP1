package system

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ap1/project/services"
)

func TestGenerateHostapdConfig(t *testing.T) {
	profile := &services.Profile{
		Name:     "test",
		SSID:     "AP1-Test",
		Password: "testpass",
		Channel:  6,
		Mode:     "g",
	}

	config := GenerateHostapdConfig(profile, "wlan0")
	if !strings.Contains(config, "ssid=AP1-Test") {
		t.Fatal("expected hostapd config to contain SSID")
	}
	if !strings.Contains(config, "wpa_passphrase=testpass") {
		t.Fatal("expected hostapd config to contain passphrase")
	}
}

func TestGenerateDnsmasqConfig(t *testing.T) {
	profile := &services.Profile{
		DHCPEnabled: true,
	}

	config := GenerateDnsmasqConfig(profile, "wlan0")
	if !strings.Contains(config, "interface=wlan0") {
		t.Fatal("expected dnsmasq config to contain interface")
	}
	if !strings.Contains(config, "dhcp-range=") {
		t.Fatal("expected dnsmasq config to contain dhcp-range when DHCP is enabled")
	}
}

func TestApplyProfileConfigWritesFiles(t *testing.T) {
	tmp := t.TempDir()
	profile := &services.Profile{
		Name:        "default",
		SSID:        "AP1-Default",
		Password:    "ap1pass",
		Channel:     6,
		Mode:        "g",
		DHCPEnabled: true,
	}

	output, err := ApplyProfileConfig(profile, "wlan0", tmp)
	if err != nil {
		t.Fatalf("ApplyProfileConfig failed: %v", err)
	}
	if !strings.Contains(output, "wrote") {
		t.Fatal("expected output to include write confirmation")
	}

	hostapdPath := filepath.Join(tmp, "hostapd.conf")
	dnsmasqPath := filepath.Join(tmp, "dnsmasq.conf")
	if _, err := os.Stat(hostapdPath); err != nil {
		t.Fatalf("expected hostapd.conf to exist: %v", err)
	}
	if _, err := os.Stat(dnsmasqPath); err != nil {
		t.Fatalf("expected dnsmasq.conf to exist: %v", err)
	}
}
