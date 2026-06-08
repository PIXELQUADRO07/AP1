//! DHCP server scaffolding

use std::fs;
use std::io::Write;
use std::path::Path;
use std::process::Command;

fn config_path() -> String {
    std::env::var("AP1_DHCP_CONF").unwrap_or_else(|_| "../system/runtime/dhcp.conf".to_string())
}

pub fn write_dhcp_config(conf: &str, path: &str) -> Result<(), String> {
    if let Some(parent) = Path::new(path).parent() {
        fs::create_dir_all(parent).map_err(|e| format!("failed to create directory {}: {}", parent.display(), e))?;
    }

    let mut file = fs::File::create(path).map_err(|e| format!("failed to create {}: {}", path, e))?;
    file.write_all(conf.as_bytes()).map_err(|e| format!("failed to write {}: {}", path, e))?;
    Ok(())
}

pub fn generate_dhcp_config(interface: &str, _network: &str, range_start: &str, range_end: &str, lease_time: &str) -> String {
    let iface = if interface.is_empty() { "wlan0" } else { interface };
    let lease = if lease_time.is_empty() { "12h" } else { lease_time };
    format!(
        "interface={}\ndhcp-range={},{},{}\ndhcp-option=3,192.168.50.1\ndhcp-option=6,192.168.50.1\nlog-dhcp\n",
        iface, range_start, range_end, lease
    )
}

pub fn parse_dhcp_logs(log_path: &str) -> Vec<String> {
    // In a real scenario, we would parse dnsmasq logs to find DHCPACK
    // and extract MAC, IP, and Hostname
    if let Ok(content) = fs::read_to_string(log_path) {
        content.lines()
            .filter(|l| l.contains("DHCPACK"))
            .map(|s| s.to_string())
            .collect()
    } else {
        Vec::new()
    }
}

pub fn start_dhcp() -> Result<String, String> {
    let path = config_path();
    if !Path::new(&path).exists() {
        return Err(format!("dhcp config not found at {}", path));
    }

    let child = Command::new("dnsmasq")
        .arg("--conf-file")
        .arg(&path)
        .spawn()
        .map_err(|e| e.to_string())?;

    if child.id() > 0 {
        Ok(format!("dhcp server started with {}", path))
    } else {
        Err("failed to start dhcp server".to_string())
    }
}

pub fn stop_dhcp() -> Result<String, String> {
    if let Ok(output) = Command::new("pkill").arg("-f").arg("dnsmasq").output() {
        if output.status.success() {
            return Ok("dhcp stopped".to_string());
        }
    }
    if let Ok(output) = Command::new("killall").arg("dnsmasq").output() {
        if output.status.success() {
            return Ok("dhcp stopped".to_string());
        }
    }
    Err("failed to stop dhcp server".to_string())
}
