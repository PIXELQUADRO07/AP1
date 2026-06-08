package routes

import (
	"net/http"
	"os"

	"github.com/ap1/project/handlers"
	"github.com/ap1/project/middleware"
	"github.com/ap1/project/server"
	"github.com/ap1/project/services"
	"github.com/ap1/project/websocket"
)

func NewRouter(coreURL string, cfg *services.Config, configPath, pluginConfigPath string, ps *server.PortalServer) http.Handler {
	token := os.Getenv("AP1_API_TOKEN")
	mux := http.NewServeMux()
	mux.HandleFunc("/", handlers.RootHandler)
	mux.HandleFunc("/health", handlers.HealthHandler)
	mux.HandleFunc("/ws/credentials", websocket.CredentialStream)
	mux.HandleFunc("/api/status", handlers.StatusHandler(coreURL))
	mux.HandleFunc("/api/config", handlers.ConfigHandler(cfg))
	mux.HandleFunc("/api/config/set_interface", middleware.TokenAuth(token, handlers.SetInterfaceHandler(cfg, coreURL)))
	mux.HandleFunc("/api/config/update", middleware.TokenAuth(token, handlers.UpdateConfigHandler(cfg, coreURL)))
	mux.HandleFunc("/api/config/preset", middleware.TokenAuth(token, handlers.PresetConfigHandler(cfg, coreURL)))
	mux.HandleFunc("/api/profiles", handlers.ProfilesHandler(cfg))
	mux.HandleFunc("/api/profiles/create", middleware.TokenAuth(token, handlers.CreateProfileHandler(cfg, configPath)))
	mux.HandleFunc("/api/profiles/update", middleware.TokenAuth(token, handlers.UpdateProfileHandler(cfg, configPath)))
	mux.HandleFunc("/api/profiles/delete", middleware.TokenAuth(token, handlers.DeleteProfileHandler(cfg, configPath)))
	mux.HandleFunc("/api/plugins", handlers.PluginsHandler(pluginConfigPath))
	mux.HandleFunc("/api/plugins/toggle", middleware.TokenAuth(token, handlers.TogglePluginHandler(pluginConfigPath)))
	mux.HandleFunc("/api/plugins/start", middleware.TokenAuth(token, handlers.StartPluginHandler(pluginConfigPath)))
	mux.HandleFunc("/api/plugins/stop", middleware.TokenAuth(token, handlers.StopPluginHandler(pluginConfigPath)))
	mux.HandleFunc("/api/interfaces", handlers.InterfacesHandler())
	mux.HandleFunc("/api/recon/networks", handlers.ReconNetworksHandler())
	mux.HandleFunc("/api/system/hostapd/", handlers.SystemHandler("hostapd"))
	mux.HandleFunc("/api/system/dnsmasq/", handlers.SystemHandler("dnsmasq"))
	mux.HandleFunc("/api/system/firewall/apply", middleware.TokenAuth(token, handlers.ApplyFirewallHandler(coreURL)))
	mux.HandleFunc("/api/system/firewall/clear", middleware.TokenAuth(token, handlers.ClearFirewallHandler(coreURL)))
	mux.HandleFunc("/api/system/interface/configure", middleware.TokenAuth(token, handlers.ConfigureInterfaceHandler(coreURL)))
	mux.HandleFunc("/api/profiles/select", middleware.TokenAuth(token, handlers.SelectProfileHandler(cfg, configPath, ps, coreURL)))
	mux.HandleFunc("/api/portal/status", handlers.PortalStatusHandler(ps, coreURL))
	mux.HandleFunc("/api/portal/credentials", handlers.PortalCredentialsHandler(ps, coreURL))
	mux.HandleFunc("/api/portal/start", middleware.TokenAuth(token, handlers.PortalStartHandler(cfg, ps, coreURL)))
	mux.HandleFunc("/api/portal/stop", middleware.TokenAuth(token, handlers.PortalStopHandler(ps, coreURL)))
	return mux
}
