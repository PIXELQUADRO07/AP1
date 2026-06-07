package network

import (
    "fmt"
    "os/exec"
    "regexp"
    "strconv"
    "strings"
)

type WirelessNetwork struct {
    SSID       string `json:"ssid"`
    BSSID      string `json:"bssid"`
    Frequency  string `json:"frequency"`
    Channel    int    `json:"channel"`
    Quality    string `json:"quality"`
    Signal     string `json:"signal"`
    Encryption string `json:"encryption"`
}

func ScanWirelessNetworks(iface string) ([]WirelessNetwork, error) {
    if iface == "" {
        iface = "wlan0"
    }

    cmd := exec.Command("iwlist", iface, "scan")
    out, err := cmd.CombinedOutput()
    if err != nil {
        return nil, fmt.Errorf("failed to scan wireless networks on %s: %w", iface, err)
    }

    return parseIwlistOutput(string(out)), nil
}

func parseIwlistOutput(raw string) []WirelessNetwork {
    var networks []WirelessNetwork
    var current WirelessNetwork
    cellRegex := regexp.MustCompile(`Cell \d+ - Address: ([0-9A-Fa-f:]+)`) 
    essidRegex := regexp.MustCompile(`ESSID:"(.*)"`)
    freqRegex := regexp.MustCompile(`Frequency:([0-9\.]+ GHz) \(Channel (\d+)\)`)
    qualityRegex := regexp.MustCompile(`Quality=([0-9]+/[0-9]+)`) 
    signalRegex := regexp.MustCompile(`Signal level=(-?[0-9]+) dBm`)
    encRegex := regexp.MustCompile(`Encryption key:(on|off)`)
    wpaRegex := regexp.MustCompile(`IE: (WPA|IEEE 802\.11i/WPA2)`)

    lines := strings.Split(raw, "\n")
    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        if trimmed == "" {
            continue
        }

        if m := cellRegex.FindStringSubmatch(trimmed); len(m) == 2 {
            if current.BSSID != "" {
                networks = append(networks, current)
                current = WirelessNetwork{}
            }
            current.BSSID = m[1]
            current.Encryption = "off"
            continue
        }

        if m := essidRegex.FindStringSubmatch(trimmed); len(m) == 2 {
            current.SSID = m[1]
            continue
        }

        if m := freqRegex.FindStringSubmatch(trimmed); len(m) == 3 {
            current.Frequency = m[1]
            if ch, err := strconv.Atoi(m[2]); err == nil {
                current.Channel = ch
            }
            continue
        }

        if m := qualityRegex.FindStringSubmatch(trimmed); len(m) == 2 {
            current.Quality = m[1]
            continue
        }

        if m := signalRegex.FindStringSubmatch(trimmed); len(m) == 2 {
            current.Signal = m[1]
            continue
        }

        if m := encRegex.FindStringSubmatch(trimmed); len(m) == 2 {
            if m[1] == "on" {
                current.Encryption = "on"
            } else {
                current.Encryption = "off"
            }
            continue
        }

        if m := wpaRegex.FindStringSubmatch(trimmed); len(m) == 2 {
            if current.Encryption == "on" {
                current.Encryption = "WPA/WPA2"
            }
            continue
        }
    }

    if current.BSSID != "" {
        networks = append(networks, current)
    }

    return networks
}

func ListInterfaces() ([]string, error) {
    cmd := exec.Command("ip", "-o", "link", "show")
    out, err := cmd.Output()
    if err != nil {
        return nil, err
    }
    lines := strings.Split(string(out), "\n")
    var names []string
    for _, line := range lines {
        if line == "" {
            continue
        }
        parts := strings.Fields(line)
        if len(parts) >= 2 {
            names = append(names, strings.TrimSuffix(parts[1], ":"))
        }
    }
    return names, nil
}
