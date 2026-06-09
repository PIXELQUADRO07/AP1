use serde::{Deserialize, Serialize};

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct AppInfo {
    pub name: String,
    pub environment: String,
    pub api_url: String,
    pub core_url: String,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct NetworkConfig {
    pub default_interface: String,
    pub captive_portal: bool,
    pub template: String,
    pub portal_ip: String,
    pub portal_port: Option<u16>,
    pub portal_fallback_port: Option<u16>,
    pub dns_ip: String,
    pub subnet: Option<u8>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct LoggingConfig {
    pub level: String,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct Profile {
    pub name: String,
    pub ssid: String,
    pub bssid: Option<String>,
    pub password: String,
    pub channel: i32,
    pub mode: String,
    pub dhcp_enabled: bool,
    pub security: Option<String>, // "open", "wep", "wpa", "wpa2"
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct AppConfig {
    pub app: AppInfo,
    pub network: NetworkConfig,
    pub logging: LoggingConfig,
    pub active_profile: Option<String>,
    pub profiles: Option<Vec<Profile>>,
}
