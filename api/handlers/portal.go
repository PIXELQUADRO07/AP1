package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/ap1/project/server"
	"github.com/ap1/project/services"
	"net/http"
)

func PortalCredentialsHandler(ps *server.PortalServer, coreURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if coreURL != "" {
			resp, err := getFromCore(coreURL, "/api/portal/credentials")
			if err == nil {
				writeCoreResponse(w, resp)
				return
			}
		}

		credentials := ps.GetCredentials()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(credentials)
	}
}

func PortalStatusHandler(ps *server.PortalServer, coreURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if coreURL != "" {
			resp, err := getFromCore(coreURL, "/api/portal/status")
			if err == nil {
				writeCoreResponse(w, resp)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"running":     ps.IsRunning(),
			"credentials": ps.GetCredentials(),
		})
	}
}

func PortalStartHandler(cfg *services.Config, ps *server.PortalServer, coreURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if coreURL != "" {
			resp, err := postToCore(coreURL, "/api/portal/start", map[string]string{})
			if err == nil {
				writeCoreResponse(w, resp)
				return
			}
		}

		if ps.IsRunning() {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "portal already running"})
			return
		}

		port := cfg.Network.PortalPort
		if port == 0 {
			port = 80
		}
		fallback := cfg.Network.PortalFallbackPort
		if fallback == 0 {
			fallback = 8000
		}

		err := ps.Start(fmt.Sprintf(":%d", port))
		if err != nil {
			err = ps.Start(fmt.Sprintf(":%d", fallback))
		}
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to start portal: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "portal started"})
	}
}

func PortalStopHandler(ps *server.PortalServer, coreURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if coreURL != "" {
			resp, err := postToCore(coreURL, "/api/portal/stop", map[string]string{})
			if err == nil {
				writeCoreResponse(w, resp)
				return
			}
		}

		if !ps.IsRunning() {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "portal already stopped"})
			return
		}

		if err := ps.Stop(); err != nil {
			http.Error(w, fmt.Sprintf("failed to stop portal: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "portal stopped"})
	}
}
