package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ap1/project/server"
	"github.com/ap1/project/services"
	"github.com/ap1/project/system"
)

type profileRequest struct {
	Name        string `json:"name"`
	SSID        string `json:"ssid"`
	Password    string `json:"password"`
	Channel     int    `json:"channel"`
	Mode        string `json:"mode"`
	DHCPEnabled bool   `json:"dhcp_enabled"`
}

type selectProfileRequest struct {
	Profile string `json:"profile"`
}

func ProfilesHandler(cfg *services.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		encoded, err := json.Marshal(cfg.Profiles)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to encode profiles: %v", err), http.StatusInternalServerError)
			return
		}
		w.Write(encoded)
	}
}

func SelectProfileHandler(cfg *services.Config, configPath string, ps *server.PortalServer, coreURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var payload selectProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		if payload.Profile == "" {
			http.Error(w, "profile is required", http.StatusBadRequest)
			return
		}

		if coreURL != "" {
			resp, err := postToCore(coreURL, "/api/profiles/select", payload)
			if err == nil {
				writeCoreResponse(w, resp)
				return
			}
		}

		var selected *services.Profile
		for i := range cfg.Profiles {
			if cfg.Profiles[i].Name == payload.Profile {
				selected = &cfg.Profiles[i]
				break
			}
		}

		if selected == nil {
			http.Error(w, "profile not found", http.StatusNotFound)
			return
		}

		cfg.ActiveProfile = payload.Profile
		if err := services.SaveConfig(configPath, cfg); err != nil {
			http.Error(w, fmt.Sprintf("failed to save config: %v", err), http.StatusInternalServerError)
			return
		}

		runtimeDir := "../system/runtime"
		iface := cfg.Network.DefaultInterface
		if iface == "" {
			iface = "wlan0"
		}

		portalIP := cfg.Network.PortalIP
		if portalIP == "" {
			portalIP = "192.168.50.1"
		}
		subnet := fmt.Sprintf("%d", cfg.Network.Subnet)
		if cfg.Network.Subnet == 0 {
			subnet = "24"
		}

		// Step 1: Configure network interface
		if err := system.ConfigureInterface(iface, portalIP, subnet); err != nil {
			http.Error(w, fmt.Sprintf("failed to configure interface: %v", err), http.StatusInternalServerError)
			return
		}

		// Step 2: Generate and apply AP config (hostapd/dnsmasq)
		output, err := system.ApplyProfileConfig(selected, iface, runtimeDir)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to apply profile: %v", err), http.StatusInternalServerError)
			return
		}

		// Step 3: Apply firewall rules
		if err := system.ApplyFirewallRules(iface, portalIP); err != nil {
			fmt.Fprintf(w, "warning: firewall rules: %v\n", err)
		}

		// Step 4: Stop old portal server if running
		if ps.IsRunning() {
			ps.Stop()
		}

		// Step 5: Start new portal server on configured port
		portalPort := cfg.Network.PortalPort
		if portalPort == 0 {
			portalPort = 80
		}
		fallbackPort := cfg.Network.PortalFallbackPort
		if fallbackPort == 0 {
			fallbackPort = 8000
		}
		if err := ps.Start(fmt.Sprintf(":%d", portalPort)); err != nil {
			if err := ps.Start(fmt.Sprintf(":%d", fallbackPort)); err != nil {
				fmt.Fprintf(w, "warning: failed to start portal server: %v\n", err)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"active_profile": payload.Profile,
			"details":        output,
			"status":         "AP activated, firewall configured, portal started",
		})
	}
}

func CreateProfileHandler(cfg *services.Config, configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var payload profileRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		if payload.Name == "" || payload.SSID == "" {
			http.Error(w, "name and ssid are required", http.StatusBadRequest)
			return
		}

		for _, profile := range cfg.Profiles {
			if profile.Name == payload.Name {
				http.Error(w, "profile already exists", http.StatusConflict)
				return
			}
		}

		cfg.Profiles = append(cfg.Profiles, services.Profile{
			Name:        payload.Name,
			SSID:        payload.SSID,
			Password:    payload.Password,
			Channel:     payload.Channel,
			Mode:        payload.Mode,
			DHCPEnabled: payload.DHCPEnabled,
		})

		if cfg.ActiveProfile == "" {
			cfg.ActiveProfile = payload.Name
		}

		if err := services.SaveConfig(configPath, cfg); err != nil {
			http.Error(w, fmt.Sprintf("failed to save config: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(payload)
	}
}

func UpdateProfileHandler(cfg *services.Config, configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var payload profileRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		if payload.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}

		updated := false
		for i, profile := range cfg.Profiles {
			if profile.Name == payload.Name {
				cfg.Profiles[i] = services.Profile{
					Name:        payload.Name,
					SSID:        payload.SSID,
					Password:    payload.Password,
					Channel:     payload.Channel,
					Mode:        payload.Mode,
					DHCPEnabled: payload.DHCPEnabled,
				}
				updated = true
				break
			}
		}

		if !updated {
			http.Error(w, "profile not found", http.StatusNotFound)
			return
		}

		if err := services.SaveConfig(configPath, cfg); err != nil {
			http.Error(w, fmt.Sprintf("failed to save config: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}
}

func DeleteProfileHandler(cfg *services.Config, configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var payload selectProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		if payload.Profile == "" {
			http.Error(w, "profile is required", http.StatusBadRequest)
			return
		}

		removed := false
		nextProfiles := make([]services.Profile, 0, len(cfg.Profiles))
		for _, profile := range cfg.Profiles {
			if profile.Name == payload.Profile {
				removed = true
				continue
			}
			nextProfiles = append(nextProfiles, profile)
		}

		if !removed {
			http.Error(w, "profile not found", http.StatusNotFound)
			return
		}

		cfg.Profiles = nextProfiles
		if cfg.ActiveProfile == payload.Profile {
			if len(cfg.Profiles) > 0 {
				cfg.ActiveProfile = cfg.Profiles[0].Name
			} else {
				cfg.ActiveProfile = ""
			}
		}

		if err := services.SaveConfig(configPath, cfg); err != nil {
			http.Error(w, fmt.Sprintf("failed to save config: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
