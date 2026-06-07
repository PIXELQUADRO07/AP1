package handlers

import (
    "fmt"
    "io"
    "net/http"

    "github.com/ap1/project/services"
)

type StatusResponse struct {
    Service string           `json:"service"`
    Version string           `json:"version"`
    Config  *services.Config `json:"config"`
}

func RootHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprint(w, `{"message":"AP1 API server is running"}`)
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprint(w, `{"status":"ok"}`)
}

func StatusHandler(coreURL string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")

        resp, err := http.Get(coreURL + "/status")
        if err != nil {
            http.Error(w, fmt.Sprintf("failed to reach core: %v", err), http.StatusBadGateway)
            return
        }
        defer resp.Body.Close()

        w.WriteHeader(resp.StatusCode)
        _, _ = io.Copy(w, resp.Body)
    }
}
