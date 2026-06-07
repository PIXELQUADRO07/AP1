package handlers

import (
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/ap1/project/system/network"
)

func InterfacesHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }

        interfaces, err := network.ListInterfaces()
        if err != nil {
            http.Error(w, fmt.Sprintf("failed to list interfaces: %v", err), http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(interfaces)
    }
}

func ReconNetworksHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }

        iface := r.URL.Query().Get("iface")
        if iface == "" {
            iface = "wlan0"
        }

        networks, err := network.ScanWirelessNetworks(iface)
        if err != nil {
            http.Error(w, fmt.Sprintf("failed to scan wireless interfaces: %v", err), http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(networks)
    }
}
