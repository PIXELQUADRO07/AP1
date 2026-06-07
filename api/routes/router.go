package routes

import (
	"net/http"

	"github.com/ap1/project/handlers"
	"github.com/ap1/project/server"
	"github.com/ap1/project/services"
)

func NewRouter(coreURL string, cfg *services.Config, configPath, pluginConfigPath string, ps *server.PortalServer) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handlers.RootHandler)
	mux.HandleFunc("/health", handlers.HealthHandler)
	mux.HandleFunc("/api/status", handlers.StatusHandler(coreURL))
	mux.HandleFunc("/api/config", handlers.ConfigHandler(cfg))
	mux.HandleFunc("/api/profiles", handlers.ProfilesHandler(cfg))
	mux.HandleFunc("/api/profiles/create", handlers.CreateProfileHandler(cfg, configPath))
	mux.HandleFunc("/api/profiles/update", handlers.UpdateProfileHandler(cfg, configPath))
	mux.HandleFunc("/api/profiles/delete", handlers.DeleteProfileHandler(cfg, configPath))
	mux.HandleFunc("/api/plugins", handlers.PluginsHandler(pluginConfigPath))
	mux.HandleFunc("/api/plugins/toggle", handlers.TogglePluginHandler(pluginConfigPath))
	mux.HandleFunc("/api/plugins/start", handlers.StartPluginHandler(pluginConfigPath))
	mux.HandleFunc("/api/plugins/stop", handlers.StopPluginHandler(pluginConfigPath))
	mux.HandleFunc("/api/interfaces", handlers.InterfacesHandler())
	mux.HandleFunc("/api/recon/networks", handlers.ReconNetworksHandler())
	mux.HandleFunc("/api/system/hostapd/", handlers.SystemHandler("hostapd"))
	mux.HandleFunc("/api/system/dnsmasq/", handlers.SystemHandler("dnsmasq"))
	mux.HandleFunc("/api/system/firewall/apply", handlers.ApplyFirewallHandler(coreURL))
	mux.HandleFunc("/api/system/firewall/clear", handlers.ClearFirewallHandler(coreURL))
	mux.HandleFunc("/api/system/interface/configure", handlers.ConfigureInterfaceHandler(coreURL))
	mux.HandleFunc("/api/profiles/select", handlers.SelectProfileHandler(cfg, configPath, ps, coreURL))
	mux.HandleFunc("/api/portal/status", handlers.PortalStatusHandler(ps, coreURL))
	mux.HandleFunc("/api/portal/credentials", handlers.PortalCredentialsHandler(ps, coreURL))
	mux.HandleFunc("/api/portal/start", handlers.PortalStartHandler(cfg, ps, coreURL))
	mux.HandleFunc("/api/portal/stop", handlers.PortalStopHandler(ps, coreURL))
	return mux
}
