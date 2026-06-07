package process_manager

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strconv"
    "syscall"
)

// StartProcess starts a command in background and records its PID to runtime/plugins/<name>.pid
func StartProcess(name string, command string, args ...string) (int, error) {
    runtimeDir := filepath.Join("..", "system", "runtime", "plugins")
    if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
        return 0, err
    }
    cmd := exec.Command(command, args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    if err := cmd.Start(); err != nil {
        return 0, err
    }
    pid := cmd.Process.Pid
    pidFile := filepath.Join(runtimeDir, fmt.Sprintf("%s.pid", name))
    if err := os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0o644); err != nil {
        return pid, err
    }
    return pid, nil
}

// StopProcess stops a process by reading PID from runtime/plugins/<name>.pid and killing it
func StopProcess(name string) error {
    runtimeDir := filepath.Join("..", "system", "runtime", "plugins")
    pidFile := filepath.Join(runtimeDir, fmt.Sprintf("%s.pid", name))
    data, err := os.ReadFile(pidFile)
    if err != nil {
        return err
    }
    pid, err := strconv.Atoi(string(data))
    if err != nil {
        return err
    }
    proc, err := os.FindProcess(pid)
    if err != nil {
        return err
    }
    if err := proc.Signal(syscall.SIGTERM); err != nil {
        return err
    }
    // remove pid file
    _ = os.Remove(pidFile)
    return nil
}
