//! Scheletro del plugin loader.
//!
//! Questo file illustra come definire metadati plugin e un loader di base.

use serde::{Deserialize, Serialize};
use std::fs;
use std::path::Path;

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct PluginMeta {
    pub name: String,
    pub description: String,
    #[serde(rename = "type")]
    pub plugin_type: String,
    pub enabled: bool,
    pub command: Option<String>,
    pub args: Option<Vec<String>>,
}

pub fn discover_plugins() -> Vec<PluginMeta> {
    vec![
        PluginMeta {
            name: "default-logger".to_string(),
            description: "Basic runtime logger for AP1 core events".to_string(),
            plugin_type: "core".to_string(),
            enabled: true,
            command: None,
            args: None,
        },
    ]
}

pub fn load_plugins_from_config(path: &Path) -> Result<Vec<PluginMeta>, Box<dyn std::error::Error>> {
    let raw = fs::read_to_string(path)?;
    let config: PluginConfig = serde_yaml::from_str(&raw)?;
    Ok(config.plugins)
}

pub fn load_plugin(plugin: &PluginMeta) {
    println!("Loading plugin: {} (type={}, enabled={})", plugin.name, plugin.plugin_type, plugin.enabled);
    if let Some(command) = &plugin.command {
        println!("plugin command configured: {} {:?}", command, plugin.args);
    }
}

#[derive(Debug, Deserialize)]
struct PluginConfig {
    plugins: Vec<PluginMeta>,
}
