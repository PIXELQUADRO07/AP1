package services

import (
    "bytes"
    "errors"
    "fmt"
    "os/exec"
    "strings"
)

// ServiceManager gestisce l'avvio e l'arresto dei servizi di sistema.

type ServiceState string

const (
    ServiceStarted ServiceState = "started"
    ServiceStopped ServiceState = "stopped"
)

type ServiceManager struct {
    Name string
}

func NewServiceManager(name string) *ServiceManager {
    return &ServiceManager{Name: name}
}

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

func (s *ServiceManager) runAction(action string) (string, error) {
    if action != "start" && action != "stop" && action != "restart" && action != "status" {
        return "", errors.New("azione non supportata")
    }

    if path, err := exec.LookPath("systemctl"); err == nil {
        return runCommand(path, action, s.Name)
    }
    if path, err := exec.LookPath("service"); err == nil {
        return runCommand(path, s.Name, action)
    }
    return "", errors.New("gestore servizi non disponibile")
}

func (s *ServiceManager) Start() error {
    _, err := s.runAction("start")
    return err
}

func (s *ServiceManager) Stop() error {
    _, err := s.runAction("stop")
    return err
}

func (s *ServiceManager) Status() ServiceState {
    output, err := s.runAction("status")
    if err != nil {
        if path, err2 := exec.LookPath("systemctl"); err2 == nil {
            active, err3 := runCommand(path, "is-active", s.Name)
            if err3 == nil && strings.TrimSpace(active) == "active" {
                return ServiceStarted
            }
        }
        return ServiceStopped
    }

    if strings.Contains(output, "active (running)") || strings.Contains(output, "running") {
        return ServiceStarted
    }
    return ServiceStopped
}
