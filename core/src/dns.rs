//! DNS spoofing/resolver scaffolding

use std::fs;
use std::io::Write;
use std::path::Path;
use std::process::Command;

fn config_path() -> String {
    std::env::var("AP1_DNSMASQ_CONF").unwrap_or_else(|_| "../system/runtime/dnsmasq.conf".to_string())
}

pub fn generate_dnsmasq_config(interface: &str, portal_ip: &str, dns_ip: &str) -> String {
    let iface = if interface.is_empty() { "wlan0" } else { interface };
    let dns_server = if dns_ip.is_empty() { "8.8.8.8" } else { dns_ip };

    format!(
        "interface={}\nbind-interfaces\nserver={}\naddress=/#/{}/\nlog-queries\nlog-dhcp\n",
        iface, dns_server, portal_ip
    )
}

pub fn write_dnsmasq_config(conf: &str, path: &str) -> Result<(), String> {
    if let Some(parent) = Path::new(path).parent() {
        fs::create_dir_all(parent).map_err(|e| format!("failed to create directory {}: {}", parent.display(), e))?;
    }

    let mut file = fs::File::create(path).map_err(|e| format!("failed to create {}: {}", path, e))?;
    file.write_all(conf.as_bytes()).map_err(|e| format!("failed to write {}: {}", path, e))?;
    Ok(())
}

pub fn start_dns() -> Result<String, String> {
    let path = config_path();
    if !Path::new(&path).exists() {
        return Err(format!("dnsmasq config not found at {}", path));
    }

    let child = Command::new("dnsmasq")
        .arg("--conf-file")
        .arg(&path)
        .spawn()
        .map_err(|e| e.to_string())?;

    if child.id() > 0 {
        Ok(format!("dnsmasq started with {}", path))
    } else {
        Err("failed to start dnsmasq".to_string())
    }
}

pub fn stop_dns() -> Result<String, String> {
    if let Ok(output) = Command::new("pkill").arg("-f").arg("dnsmasq").output() {
        if output.status.success() {
            return Ok("dnsmasq stopped".to_string());
        }
    }
    if let Ok(output) = Command::new("killall").arg("dnsmasq").output() {
        if output.status.success() {
            return Ok("dnsmasq stopped".to_string());
        }
    }
    Err("failed to stop dnsmasq processes".to_string())
}

pub fn add_spoof(domain: &str, ip: &str) -> Result<String, String> {
    let config_file = config_path();
    let entry = format!("address=/{}/{}\n", domain, ip);
    fs::OpenOptions::new()
        .create(true)
        .append(true)
        .open(&config_file)
        .and_then(|mut file| file.write_all(entry.as_bytes()))
        .map_err(|e| format!("failed to append spoof entry: {}", e))?;
    Ok(format!("added spoof {} -> {}", domain, ip))
}
