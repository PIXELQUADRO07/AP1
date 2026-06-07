use serde::{Deserialize, Serialize};
use std::env;
use std::fs;
use std::io::{Read, Write};
use std::net::{TcpListener, TcpStream};
use std::path::{Path, PathBuf};


mod ap_manager;
mod proxy;
mod system_control;
mod traffic_engine;
mod utils;
mod interfaces;
mod orchestrator;
mod event_bus;
mod hostapd;
mod dhcp;
mod dns;
mod captive_portal;
mod mitm;
mod recon;
mod plugin_system;
mod packet_capture;
mod webui;
mod state_machine;
mod networking;
mod improvements;
mod https_detection;
mod logging;

#[derive(Clone, Debug, Deserialize, Serialize)]
struct AppInfo {
    name: String,
    environment: String,
    api_url: String,
    core_url: String,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
struct NetworkConfig {
    default_interface: String,
    captive_portal: bool,
    template: String,
    portal_ip: String,
    portal_port: Option<u16>,
    portal_fallback_port: Option<u16>,
    dns_ip: String,
    subnet: Option<u8>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
struct LoggingConfig {
    level: String,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
struct Profile {
    name: String,
    ssid: String,
    password: String,
    channel: i32,
    mode: String,
    dhcp_enabled: bool,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
struct AppConfig {
    app: AppInfo,
    network: NetworkConfig,
    logging: LoggingConfig,
    active_profile: Option<String>,
    profiles: Option<Vec<Profile>>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
struct PluginManifest {
    name: String,
    #[serde(rename = "type")]
    plugin_type: String,
    enabled: bool,
    description: String,
}

#[derive(Debug, Serialize)]
struct StatusResponse {
    service: String,
    version: String,
    config: AppConfig,
    plugins: Vec<PluginManifest>,
}

#[derive(Clone, Debug, Deserialize)]
struct PluginConfig {
    plugins: Vec<PluginManifest>,
}

fn find_existing_file(candidates: &[PathBuf]) -> Option<PathBuf> {
    for path in candidates {
        if path.exists() && path.is_file() {
            return Some(path.clone());
        }
    }
    None
}

fn candidate_config_paths() -> Vec<PathBuf> {
    let mut candidates = Vec::new();
    if let Ok(cwd) = env::current_dir() {
        candidates.push(cwd.join("config/global.yaml"));
        candidates.push(cwd.join("../config/global.yaml"));
    }
    if let Ok(exe) = env::current_exe() {
        let exe_dir = exe.parent().unwrap_or_else(|| Path::new(".")).to_path_buf();
        candidates.push(exe_dir.join("config/global.yaml"));
        candidates.push(exe_dir.join("../config/global.yaml"));
        candidates.push(exe_dir.join("../../config/global.yaml"));
    }
    candidates
}

fn candidate_plugin_paths() -> Vec<PathBuf> {
    let mut candidates = Vec::new();
    if let Ok(cwd) = env::current_dir() {
        candidates.push(cwd.join("config/plugins.yaml"));
        candidates.push(cwd.join("../config/plugins.yaml"));
    }
    if let Ok(exe) = env::current_exe() {
        let exe_dir = exe.parent().unwrap_or_else(|| Path::new(".")).to_path_buf();
        candidates.push(exe_dir.join("config/plugins.yaml"));
        candidates.push(exe_dir.join("../config/plugins.yaml"));
        candidates.push(exe_dir.join("../../config/plugins.yaml"));
    }
    candidates
}

fn resolve_config_path() -> Result<PathBuf, String> {
    if let Ok(env_path) = env::var("AP1_CONFIG_PATH") {
        let path = PathBuf::from(env_path);
        if path.exists() {
            return Ok(path);
        }
        return Err(format!("AP1_CONFIG_PATH points to missing file: {}", path.display()));
    }
    find_existing_file(&candidate_config_paths()).ok_or_else(|| {
        "Could not locate config/global.yaml. Set AP1_CONFIG_PATH or run from project root.".into()
    })
}

fn resolve_plugin_path() -> Result<PathBuf, String> {
    if let Ok(env_path) = env::var("AP1_PLUGIN_CONFIG_PATH") {
        let path = PathBuf::from(env_path);
        if path.exists() {
            return Ok(path);
        }
        return Err(format!("AP1_PLUGIN_CONFIG_PATH points to missing file: {}", path.display()));
    }
    find_existing_file(&candidate_plugin_paths()).ok_or_else(|| {
        "Could not locate config/plugins.yaml. Set AP1_PLUGIN_CONFIG_PATH or run from project root.".into()
    })
}

fn load_config(path: &Path) -> Result<AppConfig, String> {
    let raw = fs::read_to_string(path).map_err(|err| format!("failed to read config {}: {}", path.display(), err))?;
    serde_yaml::from_str(&raw).map_err(|err| format!("failed to parse config {}: {}", path.display(), err))
}

fn load_plugin_config(path: &Path) -> Result<Vec<PluginManifest>, String> {
    let raw = fs::read_to_string(path).map_err(|err| format!("failed to read plugin config {}: {}", path.display(), err))?;
    let config: PluginConfig = serde_yaml::from_str(&raw).map_err(|err| format!("failed to parse plugin config {}: {}", path.display(), err))?;
    Ok(config.plugins)
}

fn write_response(stream: &mut TcpStream, status: &str, body: &str, content_type: &str) {
    let response = format!(
        "HTTP/1.1 {}\r\nContent-Type: {}\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{}",
        status,
        content_type,
        body.as_bytes().len(),
        body
    );
    let _ = stream.write_all(response.as_bytes());
}

fn write_json_response(stream: &mut TcpStream, status: &str, body: &str) {
    write_response(stream, status, body, "application/json")
}

fn parse_request_line(request: &str) -> (&str, &str) {
    let line = request.lines().next().unwrap_or_default();
    let mut parts = line.split_whitespace();
    let method = parts.next().unwrap_or_default();
    let path = parts.next().unwrap_or("/");
    (method, path)
}

fn request_body(request: &str) -> &str {
    request.split("\r\n\r\n").nth(1).unwrap_or("")
}

fn profile_into_ap_profile(profile: &Profile) -> ap_manager::ApProfile {
    ap_manager::ApProfile {
        ssid: profile.ssid.clone(),
        password: profile.password.clone(),
        channel: profile.channel as u8,
        mode: profile.mode.clone(),
        dhcp_enabled: profile.dhcp_enabled,
    }
}

fn save_config(path: &Path, cfg: &AppConfig) -> Result<(), String> {
    let raw = serde_yaml::to_string(cfg).map_err(|e| format!("failed to serialize config: {}", e))?;
    fs::write(path, raw).map_err(|e| format!("failed to write config: {}", e))
}

fn portal_interface(config: &AppConfig) -> String {
    if config.network.default_interface.is_empty() {
        "wlan0".to_string()
    } else {
        config.network.default_interface.clone()
    }
}

fn portal_ip(config: &AppConfig) -> String {
    if config.network.portal_ip.is_empty() {
        "192.168.50.1".to_string()
    } else {
        config.network.portal_ip.clone()
    }
}

fn portal_config_from_app(config: &AppConfig) -> captive_portal::PortalConfig {
    captive_portal::PortalConfig {
        template_dir: if config.network.template.is_empty() {
            "../config/templates/DarkLogin".to_string()
        } else {
            format!("../config/templates/{}", config.network.template)
        },
        log_path: "../system/runtime/portal_credentials.log".to_string(),
        portal_ip: portal_ip(config),
        portal_port: config.network.portal_port.unwrap_or(80),
        fallback_port: config.network.portal_fallback_port.unwrap_or(8000),
    }
}

#[derive(Deserialize)]
struct ProfileSelectRequest {
    profile: String,
}

#[derive(Deserialize)]
struct FirewallRequest {
    interface: Option<String>,
    portal_ip: Option<String>,
}

#[derive(Deserialize)]
struct InterfaceRequest {
    interface: Option<String>,
    ip: Option<String>,
    subnet: Option<String>,
}

fn handle_request(stream: &mut TcpStream, config_path: &PathBuf, plugin_config_path: &PathBuf) {
    let mut buffer = [0; 4096];
    let size = match stream.read(&mut buffer) {
        Ok(size) if size > 0 => size,
        _ => return,
    };

    let request = String::from_utf8_lossy(&buffer[..size]);
    let (method, request_path) = parse_request_line(&request);
    let request_path = request_path.split('?').next().unwrap_or(request_path);

    match request_path {
        "/" => {
            if method == "GET" {
                write_response(stream, "200 OK", "AP1 core engine alive", "text/plain");
            } else {
                write_response(stream, "405 Method Not Allowed", "Method not allowed", "text/plain");
            }
        }
        "/health" => {
            if method == "GET" {
                write_json_response(stream, "200 OK", "{\"status\":\"ok\"}");
            } else {
                write_response(stream, "405 Method Not Allowed", "Method not allowed", "text/plain");
            }
        }
        "/status" => {
            if method != "GET" {
                write_response(stream, "405 Method Not Allowed", "Method not allowed", "text/plain");
                return;
            }
            let config = match load_config(config_path) {
                Ok(cfg) => cfg,
                Err(err) => {
                    write_response(stream, "500 Internal Server Error", &format!("{{\"error\":\"{}\"}}", err), "application/json");
                    return;
                }
            };
            let plugins = match load_plugin_config(plugin_config_path) {
                Ok(list) => list,
                Err(_) => Vec::new(),
            };
            let response = StatusResponse {
                service: "ap1_core".to_string(),
                version: env!("CARGO_PKG_VERSION").to_string(),
                config,
                plugins,
            };
            let body = serde_json::to_string(&response).unwrap_or_else(|_| "{}".to_string());
            write_json_response(stream, "200 OK", &body);
        }
        "/config" => {
            if method != "GET" {
                write_response(stream, "405 Method Not Allowed", "Method not allowed", "text/plain");
                return;
            }
            let config = match load_config(config_path) {
                Ok(cfg) => cfg,
                Err(err) => {
                    write_response(stream, "500 Internal Server Error", &format!("{{\"error\":\"{}\"}}", err), "application/json");
                    return;
                }
            };
            let body = serde_json::to_string(&config).unwrap_or_else(|_| "{}".to_string());
            write_json_response(stream, "200 OK", &body);
        }
        "/plugins" => {
            if method != "GET" {
                write_response(stream, "405 Method Not Allowed", "Method not allowed", "text/plain");
                return;
            }
            let plugins = match load_plugin_config(plugin_config_path) {
                Ok(list) => list,
                Err(err) => {
                    write_response(stream, "500 Internal Server Error", &format!("{{\"error\":\"{}\"}}", err), "application/json");
                    return;
                }
            };
            let body = serde_json::to_string(&plugins).unwrap_or_else(|_| "[]".to_string());
            write_json_response(stream, "200 OK", &body);
        }
        "/api/system/firewall/apply" => {
            if method != "POST" {
                write_response(stream, "405 Method Not Allowed", "Method not allowed", "text/plain");
                return;
            }
            let body = request_body(&request);
            let payload: FirewallRequest = match serde_json::from_str(body) {
                Ok(payload) => payload,
                Err(err) => {
                    write_response(stream, "400 Bad Request", &format!("{{\"error\":\"invalid payload: {}\"}}", err), "application/json");
                    return;
                }
            };
            let iface = payload.interface.unwrap_or_else(|| "wlan0".to_string());
            let portal = payload.portal_ip.unwrap_or_else(|| "192.168.50.1".to_string());
            if let Err(err) = system_control::apply_firewall_rules(&iface, &portal) {
                write_response(stream, "500 Internal Server Error", &format!("{{\"error\":\"{}\"}}", err), "application/json");
                return;
            }
            write_json_response(stream, "200 OK", "{\"status\":\"firewall rules applied\"}");
        }
        "/api/system/firewall/clear" => {
            if method != "POST" {
                write_response(stream, "405 Method Not Allowed", "Method not allowed", "text/plain");
                return;
            }
            let body = request_body(&request);
            let payload: FirewallRequest = match serde_json::from_str(body) {
                Ok(payload) => payload,
                Err(err) => {
                    write_response(stream, "400 Bad Request", &format!("{{\"error\":\"invalid payload: {}\"}}", err), "application/json");
                    return;
                }
            };
            let iface = payload.interface.unwrap_or_else(|| "wlan0".to_string());
            let portal = payload.portal_ip.unwrap_or_else(|| "192.168.50.1".to_string());
            if let Err(err) = system_control::clear_firewall_rules(&iface, &portal) {
                write_response(stream, "500 Internal Server Error", &format!("{{\"error\":\"{}\"}}", err), "application/json");
                return;
            }
            write_json_response(stream, "200 OK", "{\"status\":\"firewall rules cleared\"}");
        }
        "/api/system/interface/configure" => {
            if method != "POST" {
                write_response(stream, "405 Method Not Allowed", "Method not allowed", "text/plain");
                return;
            }
            let body = request_body(&request);
            let payload: InterfaceRequest = match serde_json::from_str(body) {
                Ok(payload) => payload,
                Err(err) => {
                    write_response(stream, "400 Bad Request", &format!("{{\"error\":\"invalid payload: {}\"}}", err), "application/json");
                    return;
                }
            };
            let iface = payload.interface.unwrap_or_else(|| "wlan0".to_string());
            let ip = payload.ip.unwrap_or_else(|| "192.168.50.1".to_string());
            let subnet = payload.subnet.unwrap_or_else(|| "24".to_string());
            if let Err(err) = system_control::configure_interface(&iface, &ip, &subnet) {
                write_response(stream, "500 Internal Server Error", &format!("{{\"error\":\"{}\"}}", err), "application/json");
                return;
            }
            write_json_response(stream, "200 OK", &format!("{{\"status\":\"interface {} configured\"}}", iface));
        }
        "/api/portal/start" => {
            if method != "POST" {
                write_response(stream, "405 Method Not Allowed", "Method not allowed", "text/plain");
                return;
            }
            let config = match load_config(config_path) {
                Ok(cfg) => cfg,
                Err(err) => {
                    write_response(stream, "500 Internal Server Error", &format!("{{\"error\":\"{}\"}}", err), "application/json");
                    return;
                }
            };
            let portal_cfg = portal_config_from_app(&config);
            captive_portal::start_portal_with_config(portal_cfg.clone());
            traffic_engine::start_traffic_engine(&portal_interface(&config), &portal_cfg.portal_ip);
            write_json_response(stream, "200 OK", "{\"status\":\"portal started\"}");
        }
        "/api/portal/stop" => {
            if method != "POST" {
                write_response(stream, "405 Method Not Allowed", "Method not allowed", "text/plain");
                return;
            }
            let config = match load_config(config_path) {
                Ok(cfg) => cfg,
                Err(err) => {
                    write_response(stream, "500 Internal Server Error", &format!("{{\"error\":\"{}\"}}", err), "application/json");
                    return;
                }
            };
            let iface = portal_interface(&config);
            let portal_ip = portal_ip(&config);
            captive_portal::stop_portal();
            traffic_engine::stop_traffic_engine(&iface, &portal_ip);
            write_json_response(stream, "200 OK", "{\"status\":\"portal stopped\"}");
        }
        "/api/portal/status" => {
            if method != "GET" {
                write_response(stream, "405 Method Not Allowed", "Method not allowed", "text/plain");
                return;
            }
            let is_running = captive_portal::is_running();
            let body = format!("{{\"running\":{} }}", is_running);
            write_json_response(stream, "200 OK", &body);
        }
        "/api/portal/credentials" => {
            if method != "GET" {
                write_response(stream, "405 Method Not Allowed", "Method not allowed", "text/plain");
                return;
            }
            let credentials = captive_portal::read_credentials();
            let body = serde_json::to_string(&credentials).unwrap_or_else(|_| "[]".to_string());
            write_json_response(stream, "200 OK", &body);
        }
        "/api/profiles/select" => {
            if method != "POST" {
                write_response(stream, "405 Method Not Allowed", "Method not allowed", "text/plain");
                return;
            }
            let body = request_body(&request);
            let payload: ProfileSelectRequest = match serde_json::from_str(body) {
                Ok(payload) => payload,
                Err(err) => {
                    write_response(stream, "400 Bad Request", &format!("{{\"error\":\"invalid payload: {}\"}}", err), "application/json");
                    return;
                }
            };
            if payload.profile.is_empty() {
                write_response(stream, "400 Bad Request", "{\"error\":\"profile is required\"}", "application/json");
                return;
            }
            let mut config = match load_config(config_path) {
                Ok(cfg) => cfg,
                Err(err) => {
                    write_response(stream, "500 Internal Server Error", &format!("{{\"error\":\"{}\"}}", err), "application/json");
                    return;
                }
            };
            let profiles = match &config.profiles {
                Some(list) => list,
                None => {
                    write_response(stream, "404 Not Found", "{\"error\":\"no profiles configured\"}", "application/json");
                    return;
                }
            };
            let selected = profiles.iter().find(|p| p.name == payload.profile);
            if selected.is_none() {
                write_response(stream, "404 Not Found", "{\"error\":\"profile not found\"}", "application/json");
                return;
            }
            config.active_profile = Some(payload.profile.clone());
            if let Err(err) = save_config(config_path, &config) {
                write_response(stream, "500 Internal Server Error", &format!("{{\"error\":\"{}\"}}", err), "application/json");
                return;
            }
            let _iface = if config.network.default_interface.is_empty() { "wlan0" } else { &config.network.default_interface };
            let ap_profile = profile_into_ap_profile(selected.unwrap());
            ap_manager::activate_profile(&ap_profile);
            if config.network.captive_portal {
                let portal_cfg = portal_config_from_app(&config);
                captive_portal::start_portal_with_config(portal_cfg.clone());
                traffic_engine::start_traffic_engine(_iface, &portal_cfg.portal_ip);
            } else {
                let portal_ip = portal_ip(&config);
                traffic_engine::start_traffic_engine(_iface, &portal_ip);
            }
            system_control::restart_system_services();
            write_json_response(stream, "200 OK", &format!("{{\"status\":\"profile {} activated\"}}", payload.profile));
        }
        _ => write_response(stream, "404 Not Found", "Not found", "text/plain"),
    }
}

fn extract_listen_addr(config: &AppConfig) -> String {
    if let Ok(addr) = env::var("AP1_CORE_ADDR") {
        if !addr.is_empty() {
            return addr;
        }
    }

    if !config.app.core_url.is_empty() {
        let url = config.app.core_url.trim();
        if let Some(stripped) = url.strip_prefix("http://") {
            return stripped.to_string();
        }
        if let Some(stripped) = url.strip_prefix("https://") {
            return stripped.to_string();
        }
    }

    "127.0.0.1:8081".to_string()
}

fn main() -> std::io::Result<()> {
    let config_path = resolve_config_path().unwrap_or_else(|err| {
        panic!("Unable to resolve config path: {}", err);
    });
    let plugin_config_path = resolve_plugin_path().unwrap_or_else(|err| {
        panic!("Unable to resolve plugin config path: {}", err);
    });
    let config = load_config(&config_path).unwrap_or_else(|err| {
        panic!("Unable to load config: {}", err);
    });

    let listen_addr = extract_listen_addr(&config);
    let listener = TcpListener::bind(&listen_addr)?;
    println!("{}", utils::format_status(&format!("Starting AP1 core engine on http://{}", listen_addr)));

    logging::init_logging();
    event_bus::subscribe("core.events");
    event_bus::emit(&"core started".to_string());
    webui::start_webui();

    let portal_ip = if config.network.portal_ip.is_empty() {
        "192.168.50.1".to_string()
    } else {
        config.network.portal_ip.clone()
    };
    let portal_port = config.network.portal_port.unwrap_or(80);
    let fallback_port = config.network.portal_fallback_port.unwrap_or(8000);
    let _dns_ip = if config.network.dns_ip.is_empty() {
        portal_ip.clone()
    } else {
        config.network.dns_ip.clone()
    };
    let _subnet = config.network.subnet.unwrap_or(24);

    let network_iface = if config.network.default_interface.is_empty() {
        "wlan0"
    } else {
        &config.network.default_interface
    };

    if config.network.captive_portal {
        let portal_cfg = captive_portal::PortalConfig {
            template_dir: if config.network.template.is_empty() {
                "../config/templates/DarkLogin".to_string()
            } else {
                format!("../config/templates/{}", config.network.template)
            },
            log_path: "../system/runtime/portal_credentials.log".to_string(),
            portal_ip: portal_ip.clone(),
            portal_port,
            fallback_port,
        };
        captive_portal::start_portal_with_config(portal_cfg);
    }

    if let Err(err) = packet_capture::start(network_iface, "../system/runtime/packet_capture.log") {
        eprintln!("packet capture start failed: {}", err);
    }
    mitm::sniff_traffic(network_iface);

    let profile = if let Some(active) = config.active_profile.as_ref() {
        config.profiles.as_ref().and_then(|profiles| {
            profiles.iter().find(|p| &p.name == active).cloned()
        }).unwrap_or_else(|| Profile {
            name: "default".to_string(),
            ssid: "AP1-Test".to_string(),
            password: "password123".to_string(),
            channel: 1,
            mode: "g".to_string(),
            dhcp_enabled: true,
        })
    } else {
        config.profiles.as_ref().and_then(|profiles| profiles.first().cloned()).unwrap_or(Profile {
            name: "default".to_string(),
            ssid: "AP1-Test".to_string(),
            password: "password123".to_string(),
            channel: 1,
            mode: "g".to_string(),
            dhcp_enabled: true,
        })
    };

    if let Err(err) = plugin_system::load_plugins(&plugin_config_path) {
        eprintln!("plugin loader warning: {}", err);
    }
    plugin_system::trigger_hook("pre_ap_start", &plugin_config_path);

    let ap_profile = profile_into_ap_profile(&profile);
    ap_manager::activate_profile(&ap_profile);
    traffic_engine::start_traffic_engine(network_iface, &portal_ip);
    proxy::init_proxy();
    system_control::restart_system_services();
    orchestrator::start_module("core");

    for stream in listener.incoming() {
        if let Ok(mut stream) = stream {
            handle_request(&mut stream, &config_path, &plugin_config_path);
        }
    }
    Ok(())
}
