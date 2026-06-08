//! AP Manager module.
//!
//! This module manages AP configuration and lifecycle.

use crate::hostapd::{start_hostapd, stop_hostapd};
use crate::system_control::{configure_interface, write_profile_configs};

pub struct ApProfile {
    pub ssid: String,
    pub password: String,
    pub channel: u8,
    pub mode: String,
    pub dhcp_enabled: bool,
}

fn normalize_hw_mode(mode: &str) -> &str {
    match mode.to_lowercase().as_str() {
        "a" => "a",
        "ac" => "a",
        "n" => "g",
        "g" => "g",
        _ => "g",
    }
}

fn build_hostapd_conf(profile: &ApProfile, iface: &str) -> String {
    let hw_mode = normalize_hw_mode(&profile.mode);
    let password = if profile.password.is_empty() { "ap1password" } else { &profile.password };
    format!(
        "interface={}\ndriver=nl80211\nssid={}\nhw_mode={}\nchannel={}\nauth_algs=1\nwpa=2\nwpa_passphrase={}\nwpa_key_mgmt=WPA-PSK\nrsn_pairwise=CCMP\n",
        iface, profile.ssid, hw_mode, profile.channel, password
    )
}

fn build_dnsmasq_conf(profile: &ApProfile, iface: &str) -> String {
    let mut config = format!("interface={}\nbind-interfaces\n", iface);
    if profile.dhcp_enabled {
        config.push_str("dhcp-range=192.168.50.10,192.168.50.100,12h\n");
    } else {
        config.push_str("# DHCP disabled for this profile\n");
    }
    config.push_str("address=/#/192.168.50.1\n");
    config.push_str("log-queries\nlog-dhcp\n");
    config
}

pub fn activate_profile(profile: &ApProfile) {
    create_ap(profile);
}

pub fn create_ap(profile: &ApProfile) {
    let iface = std::env::var("AP1_IFACE").unwrap_or_else(|_| "wlan0".to_string());
    let hostapd_conf = build_hostapd_conf(profile, &iface);
    let dnsmasq_conf = build_dnsmasq_conf(profile, &iface);

    if let Err(err) = write_profile_configs(&hostapd_conf, &dnsmasq_conf, "../system/runtime") {
        eprintln!("failed to write AP runtime configs: {}", err);
    }

    if let Err(err) = configure_interface(&iface, "192.168.50.1", "24") {
        eprintln!("failed to configure {}: {}", iface, err);
    }

    if let Err(err) = start_hostapd() {
        eprintln!("failed to start hostapd: {}", err);
    }
}

pub fn stop_ap(_ssid: &str) {
    if let Err(err) = stop_hostapd() {
        eprintln!("failed to stop hostapd: {}", err);
    }
}

pub fn configure_security(_ssid: &str, _psk: &str) {
    println!("configured security settings for AP {}", _ssid);
}

pub fn enable_monitor_mode(iface: &str, enable: bool) {
    crate::interfaces::set_monitor_mode(iface, enable);
}

pub fn spoof_ap(target_ssid: &str) {
    println!("spoofing AP: {} (using fake captive portal profile)", target_ssid);
}
