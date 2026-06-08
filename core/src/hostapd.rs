//! Hostapd wrapper scaffolding

use std::fs;
use std::io::Write;
use std::path::Path;
use std::process::Command;

use crate::config::Profile;

fn config_path() -> String {
    std::env::var("AP1_HOSTAPD_CONF").unwrap_or_else(|_| "../system/runtime/hostapd.conf".to_string())
}

pub fn generate_hostapd_config(profile: &Profile, iface: &str) -> String {
    let hw_mode = match profile.mode.to_lowercase().as_str() {
        "a" => "a",
        "ac" => "a",
        "n" => "g",
        "g" => "g",
        _ => "g",
    };

    let mut config = format!(
        "interface={}\ndriver=nl80211\nssid={}\nhw_mode={}\nchannel={}\nauth_algs=1\nwmm_enabled=1\n",
        iface, profile.ssid, hw_mode, profile.channel
    );

    if profile.mode.to_lowercase() == "n" || profile.mode.to_lowercase() == "ac" {
        config.push_str("ieee80211n=1\n");
    }
    if profile.mode.to_lowercase() == "ac" {
        config.push_str("ieee80211ac=1\n");
    }

    let security = profile.security.as_deref().unwrap_or("wpa2").to_lowercase();
    match security.as_str() {
        "open" => {
            // No security lines needed
        },
        "wep" => {
            config.push_str(&format!("wep_default_key=0\nwep_key0=\"{}\"\n", profile.password));
        },
        "wpa" => {
            config.push_str(&format!("wpa=1\nwpa_passphrase={}\nwpa_key_mgmt=WPA-PSK\nwpa_pairwise=TKIP\n", profile.password));
        },
        "wpa2" | _ => {
            let mut password = if profile.password.is_empty() { "ap1password" } else { &profile.password };
            if password.len() < 8 {
                password = "ap1password";
            }
            config.push_str(&format!("wpa=2\nwpa_passphrase={}\nwpa_key_mgmt=WPA-PSK\nrsn_pairwise=CCMP\n", password));
        }
    }

    config
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

    let log_path = "../system/runtime/logs/hostapd.log";
    if let Some(parent) = Path::new(log_path).parent() {
        let _ = fs::create_dir_all(parent);
    }
    let log_file = fs::File::create(log_path).map_err(|e| e.to_string())?;

    // Use spawn instead of output to avoid blocking the main thread
    let child = Command::new("hostapd")
        .arg(&path)
        .stdout(log_file.try_clone().map_err(|e| e.to_string())?)
        .stderr(log_file)
        .spawn()
        .map_err(|e| format!("failed to spawn hostapd: {}", e))?;

    // Check if it's still running after a short delay
    std::thread::sleep(std::time::Duration::from_millis(800));

    // Check if process is still alive
    match Command::new("kill").arg("-0").arg(child.id().to_string()).status() {
        Ok(status) if status.success() => Ok(format!("hostapd spawned with PID {}", child.id())),
        _ => {
            let error_msg = fs::read_to_string("../system/runtime/logs/hostapd.log").unwrap_or_default();
            Err(format!("hostapd failed to start. Log: {}", error_msg))
        }
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
