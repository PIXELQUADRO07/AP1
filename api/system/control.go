package system

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var ErrServiceManagerUnavailable = errors.New("service manager unavailable")

func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%v: %s", err, stderr.String())
	}
	return out.String(), nil
}

func RunServiceAction(service, action string) (string, error) {
	if action != "start" && action != "stop" && action != "restart" && action != "status" {
		return "", errors.New("action not supported")
	}

	if path, err := exec.LookPath("systemctl"); err == nil {
		return runCommand(path, action, service)
	}

	if path, err := exec.LookPath("service"); err == nil {
		return runCommand(path, service, action)
	}

	return "", ErrServiceManagerUnavailable
}

// ConfigureInterface assigns IP address to network interface
func ConfigureInterface(iface, ip, subnet string) error {
	if iface == "" {
		iface = "wlan0"
	}
	if ip == "" {
		ip = "192.168.50.1"
	}
	if subnet == "" {
		subnet = "24"
	}

	// Flush existing addresses
	if _, err := runCommand("ip", "addr", "flush", "dev", iface); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to flush %s: %v\n", iface, err)
	}

	// Assign new address
	if _, err := runCommand("ip", "addr", "add", fmt.Sprintf("%s/%s", ip, subnet), "dev", iface); err != nil {
		return fmt.Errorf("failed to assign IP %s to %s: %w", ip, iface, err)
	}

	// Bring interface up
	if _, err := runCommand("ip", "link", "set", iface, "up"); err != nil {
		return fmt.Errorf("failed to bring up %s: %w", iface, err)
	}

	return nil
}

// ApplyFirewallRules sets up iptables rules for captive portal
func ApplyFirewallRules(iface, portalIP string) error {
	if iface == "" {
		iface = "wlan0"
	}
	if portalIP == "" {
		portalIP = "192.168.50.1"
	}

	// Enable IP forwarding
	if err := os.WriteFile("/proc/sys/net/ipv4/ip_forward", []byte("1"), 0o644); err != nil {
		return fmt.Errorf("failed to enable IP forwarding: %w", err)
	}

	rules := [][]string{
		{"iptables", "-t", "nat", "-A", "POSTROUTING", "-o", iface, "-j", "MASQUERADE"},
		{"iptables", "-t", "nat", "-A", "PREROUTING", "-p", "tcp", "--dport", "80", "-j", "DNAT", "--to-destination", fmt.Sprintf("%s:80", portalIP)},
		{"iptables", "-t", "nat", "-A", "PREROUTING", "-p", "tcp", "--dport", "80", "-j", "REDIRECT", "--to-port", "8080"},
		{"iptables", "-t", "nat", "-A", "PREROUTING", "-p", "tcp", "--dport", "443", "-j", "DNAT", "--to-destination", fmt.Sprintf("%s:80", portalIP)},
		{"iptables", "-t", "nat", "-A", "PREROUTING", "-p", "udp", "--dport", "53", "-j", "DNAT", "--to-destination", fmt.Sprintf("%s:53", portalIP)},
	}

	for _, rule := range rules {
		if _, err := runCommand(rule[0], rule[1:]...); err != nil {
			fmt.Fprintf(os.Stderr, "warning: iptables rule failed (%v): %v\n", strings.Join(rule, " "), err)
		}
	}

	return nil
}

// ClearFirewallRules removes iptables rules for captive portal
func ClearFirewallRules(iface string) error {
	if iface == "" {
		iface = "wlan0"
	}

	rules := [][]string{
		{"iptables", "-t", "nat", "-D", "POSTROUTING", "-o", iface, "-j", "MASQUERADE"},
		{"iptables", "-t", "nat", "-F", "PREROUTING"},
		{"iptables", "-t", "nat", "-F", "OUTPUT"},
	}

	for _, rule := range rules {
		if _, err := runCommand(rule[0], rule[1:]...); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to clear rule (%v): %v\n", strings.Join(rule, " "), err)
		}
	}

	return nil
}
