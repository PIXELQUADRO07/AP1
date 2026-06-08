package handlers

import (
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/ap1/project/services"
)

func ConfigHandler(cfg *services.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		encoded, err := json.Marshal(cfg)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to encode config: %v", err), http.StatusInternalServerError)
			return
		}
		w.Write(encoded)
	}
}

func SetInterfaceHandler(cfg *services.Config, coreURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var payload struct {
			Interface string `json:"interface"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		// 1. Update local Go config
		cfg.Network.DefaultInterface = payload.Interface

		// 2. Update core via proxy
		resp, err := postToCore(coreURL, "/api/config/set_interface", payload)
		if err != nil {
			http.Error(w, fmt.Sprintf("core error: %v", err), http.StatusInternalServerError)
			return
		}
		writeCoreResponse(w, resp)
	}
}

func UpdateConfigHandler(cfg *services.Config, coreURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		// Update core via proxy
		resp, err := postToCore(coreURL, "/api/config/update", payload)
		if err != nil {
			http.Error(w, fmt.Sprintf("core error: %v", err), http.StatusInternalServerError)
			return
		}
		writeCoreResponse(w, resp)
	}
}

func PresetConfigHandler(cfg *services.Config, coreURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var payload struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		// Update core via proxy
		resp, err := postToCore(coreURL, "/api/config/preset", payload)
		if err != nil {
			http.Error(w, fmt.Sprintf("core error: %v", err), http.StatusInternalServerError)
			return
		}
		writeCoreResponse(w, resp)
	}
}
