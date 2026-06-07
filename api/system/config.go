package system

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ap1/project/services"
)

func formatHwMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "a":
		return "a"
	case "n":
		return "g"
	case "g":
		return "g"
	default:
		return "g"
	}
}

func GenerateHostapdConfig(profile *services.Profile, iface string) string {
	if iface == "" {
		iface = "wlan0"
	}

	hwMode := formatHwMode(profile.Mode)
	password := profile.Password
	if password == "" {
		password = "ap1pass"
	}

	return fmt.Sprintf(`interface=%s
driver=nl80211
ssid=%s
hw_mode=%s
channel=%d
auth_algs=1
wpa=2
wpa_passphrase=%s
wpa_key_mgmt=WPA-PSK
rsn_pairwise=CCMP
`, iface, profile.SSID, hwMode, profile.Channel, password)
}

func GenerateDnsmasqConfig(profile *services.Profile, iface string) string {
	if iface == "" {
		iface = "wlan0"
	}

	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("interface=%s\n", iface))
	builder.WriteString("bind-interfaces\n")
	if profile.DHCPEnabled {
		builder.WriteString("dhcp-range=192.168.50.10,192.168.50.100,12h\n")
	} else {
		builder.WriteString("# DHCP disabled for this profile\n")
	}
	builder.WriteString("server=8.8.8.8\n")
	builder.WriteString("address=/#/192.168.50.1\n")
	return builder.String()
}

func ApplyProfileConfig(profile *services.Profile, iface, runtimeDir string) (string, error) {
	if profile == nil {
		return "", errors.New("profilo non valido")
	}
	if iface == "" {
		iface = "wlan0"
	}
	if runtimeDir == "" {
		runtimeDir = "../system/runtime"
	}

	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		return "", err
	}

	hostapdPath := filepath.Join(runtimeDir, "hostapd.conf")
	dnsmasqPath := filepath.Join(runtimeDir, "dnsmasq.conf")

	hostapdConfig := GenerateHostapdConfig(profile, iface)
	dnsmasqConfig := GenerateDnsmasqConfig(profile, iface)

	if err := os.WriteFile(hostapdPath, []byte(hostapdConfig), 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(dnsmasqPath, []byte(dnsmasqConfig), 0o644); err != nil {
		return "", err
	}

	messages := []string{
		fmt.Sprintf("wrote %s", hostapdPath),
		fmt.Sprintf("wrote %s", dnsmasqPath),
	}

	hostapdOutput, hostapdErr := RunServiceAction("hostapd", "restart")
	if hostapdErr != nil {
		if errors.Is(hostapdErr, ErrServiceManagerUnavailable) {
			messages = append(messages, "service manager non disponibile, configurazione hostapd salvata")
		} else {
			messages = append(messages, fmt.Sprintf("hostapd restart warning: %v", hostapdErr))
		}
	} else {
		messages = append(messages, fmt.Sprintf("hostapd restarted: %s", strings.TrimSpace(hostapdOutput)))
	}

	dnsmasqOutput, dnsmasqErr := RunServiceAction("dnsmasq", "restart")
	if dnsmasqErr != nil {
		if errors.Is(dnsmasqErr, ErrServiceManagerUnavailable) {
			messages = append(messages, "service manager non disponibile, configurazione dnsmasq salvata")
		} else {
			messages = append(messages, fmt.Sprintf("dnsmasq restart warning: %v", dnsmasqErr))
		}
	} else {
		messages = append(messages, fmt.Sprintf("dnsmasq restarted: %s", strings.TrimSpace(dnsmasqOutput)))
	}

	return strings.Join(messages, "\n"), nil
}
