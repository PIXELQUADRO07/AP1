//! Hostapd wrapper scaffolding

use std::fs;
use std::io::Write;
use std::path::Path;
use std::process::Command;

fn config_path() -> String {
    std::env::var("AP1_HOSTAPD_CONF").unwrap_or_else(|_| "../system/runtime/hostapd.conf".to_string())
}

pub fn write_hostapd_config(conf: &str, path: &str) -> Result<(), String> {
    if let Some(parent) = Path::new(path).parent() {
        fs::create_dir_all(parent).map_err(|e| format!("failed to create directory {}: {}", parent.display(), e))?;
    }

    let mut file = fs::File::create(path).map_err(|e| format!("failed to create {}: {}", path, e))?;
    file.write_all(conf.as_bytes()).map_err(|e| format!("failed to write {}: {}", path, e))?;
    Ok(())
}

pub fn start_hostapd() -> Result<String, String> {
    let path = config_path();
    if !Path::new(&path).exists() {
        return Err(format!("hostapd config not found at {}", path));
    }

    let output = Command::new("hostapd").arg(&path).output().map_err(|e| e.to_string())?;
    if output.status.success() {
        Ok(format!("hostapd started with {}", path))
    } else {
        Err(String::from_utf8_lossy(&output.stderr).trim().to_string())
    }
}

pub fn stop_hostapd() -> Result<String, String> {
    if let Ok(output) = Command::new("pkill").arg("-f").arg("hostapd").output() {
        if output.status.success() {
            return Ok("hostapd stopped".to_string());
        }
    }
    if let Ok(output) = Command::new("killall").arg("hostapd").output() {
        if output.status.success() {
            return Ok("hostapd stopped".to_string());
        }
    }
    Err("failed to stop hostapd processes".to_string())
}
