package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ap1/project/system"
)

type systemResponse struct {
	Service string `json:"service"`
	Action  string `json:"action"`
	Output  string `json:"output"`
}

type firewallRequest struct {
	Interface string `json:"interface"`
	PortalIP  string `json:"portal_ip"`
	DNSIP     string `json:"dns_ip"`
}

type interfaceRequest struct {
	Interface string `json:"interface"`
	IP        string `json:"ip"`
	Subnet    string `json:"subnet"`
}

func SystemHandler(serviceName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		action := strings.TrimPrefix(r.URL.Path, "/api/system/"+serviceName+"/")
		if action == "" {
			http.Error(w, "Azione richiesta mancante", http.StatusBadRequest)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		output, err := system.RunServiceAction(serviceName, action)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to %s %s: %v", action, serviceName, err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		resp := systemResponse{
			Service: serviceName,
			Action:  action,
			Output:  output,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func ApplyFirewallHandler(coreURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var payload firewallRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		if payload.Interface == "" {
			payload.Interface = "wlan0"
		}
		if payload.PortalIP == "" {
			payload.PortalIP = "192.168.50.1"
		}

		if coreURL != "" {
			resp, err := postToCore(coreURL, "/api/system/firewall/apply", payload)
			if err == nil {
				writeCoreResponse(w, resp)
				return
			}
		}

		if err := system.ApplyFirewallRules(payload.Interface, payload.PortalIP); err != nil {
			http.Error(w, fmt.Sprintf("failed to apply firewall rules: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "firewall rules applied",
		})
	}
}

func ClearFirewallHandler(coreURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var payload firewallRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		if payload.Interface == "" {
			payload.Interface = "wlan0"
		}

		if coreURL != "" {
			resp, err := postToCore(coreURL, "/api/system/firewall/clear", payload)
			if err == nil {
				writeCoreResponse(w, resp)
				return
			}
		}

		if err := system.ClearFirewallRules(payload.Interface); err != nil {
			http.Error(w, fmt.Sprintf("failed to clear firewall rules: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "firewall rules cleared",
		})
	}
}

func ConfigureInterfaceHandler(coreURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var payload interfaceRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		if payload.Interface == "" {
			payload.Interface = "wlan0"
		}
		if payload.IP == "" {
			payload.IP = "192.168.50.1"
		}
		if payload.Subnet == "" {
			payload.Subnet = "24"
		}

		if coreURL != "" {
			resp, err := postToCore(coreURL, "/api/system/interface/configure", payload)
			if err == nil {
				writeCoreResponse(w, resp)
				return
			}
		}

		if err := system.ConfigureInterface(payload.Interface, payload.IP, payload.Subnet); err != nil {
			http.Error(w, fmt.Sprintf("failed to configure interface: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": fmt.Sprintf("interface %s configured with %s/%s", payload.Interface, payload.IP, payload.Subnet),
		})
	}
}

