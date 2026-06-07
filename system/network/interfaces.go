package network

import (
    "os/exec"
    "strings"
)

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
            name := strings.TrimSuffix(parts[1], ":")
            names = append(names, name)
        }
    }
    return names, nil
}

func SetInterfaceUp(iface string) error {
    cmd := exec.Command("ip", "link", "set", iface, "up")
    return cmd.Run()
}

func SetInterfaceDown(iface string) error {
    cmd := exec.Command("ip", "link", "set", iface, "down")
    return cmd.Run()
}
