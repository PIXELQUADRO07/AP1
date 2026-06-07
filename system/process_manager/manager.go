package process_manager

import (
    "bytes"
    "fmt"
    "os/exec"
)

func RunCommand(command string, args ...string) (string, error) {
    cmd := exec.Command(command, args...)
    var out bytes.Buffer
    var stderr bytes.Buffer
    cmd.Stdout = &out
    cmd.Stderr = &stderr
    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("%v: %s", err, stderr.String())
    }
    return out.String(), nil
}
