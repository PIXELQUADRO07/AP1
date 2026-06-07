//! Plugin system support.

use ap1_plugin_loader::{load_plugins_from_config, PluginMeta};
use std::collections::HashMap;
use std::path::Path;
use std::process::Command;
use std::sync::{Mutex, OnceLock};

static PLUGIN_PROCESSES: OnceLock<Mutex<HashMap<String, std::process::Child>>> = OnceLock::new();

fn process_store() -> &'static Mutex<HashMap<String, std::process::Child>> {
    PLUGIN_PROCESSES.get_or_init(|| Mutex::new(HashMap::new()))
}

pub fn list_plugins(path: &Path) -> Result<Vec<PluginMeta>, String> {
    load_plugins_from_config(path).map_err(|e| format!("failed to load plugin config: {}", e))
}

pub fn load_plugins(path: &Path) -> Result<(), String> {
    let plugins = list_plugins(path)?;
    for plugin in plugins.iter().filter(|p| p.enabled) {
        load_plugin(plugin);
    }
    Ok(())
}

pub fn load_plugin(plugin: &PluginMeta) {
    println!("Loading plugin: {} (type={}, enabled={})", plugin.name, plugin.plugin_type, plugin.enabled);
    if let Some(command) = &plugin.command {
        let mut cmd = Command::new(command);
        if let Some(args) = &plugin.args {
            cmd.args(args);
        }
        cmd.env("AP1_PLUGIN_ENABLED", "1");
        match cmd.spawn() {
            Ok(child) => {
                process_store().lock().unwrap().insert(plugin.name.clone(), child);
                println!("spawned plugin process: {}", plugin.name);
            }
            Err(err) => {
                eprintln!("failed to spawn plugin {}: {}", plugin.name, err);
            }
        }
    }
}

pub fn trigger_hook(hook: &str, config_path: &Path) {
    println!("triggering plugin hook: {}", hook);
    if let Ok(plugins) = list_plugins(config_path) {
        for plugin in plugins.into_iter().filter(|p| p.enabled) {
            println!("plugin {} is enabled and ready for hook {}", plugin.name, hook);
        }
    }
}

pub fn stop_plugins() {
    let mut store = process_store().lock().unwrap();
    for (name, child) in store.iter_mut() {
        let _ = child.kill();
        println!("terminated plugin process: {}", name);
    }
    store.clear();
}
