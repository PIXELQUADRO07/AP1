//! System control module.
//!
//! Wrapper for system commands and network service management.

use std::fs;
use std::path::Path;
use std::process::Command;

pub fn run_command(cmd: &str, args: &[&str]) -> Result<String, String> {
    let output = Command::new(cmd).args(args).output().map_err(|e| e.to_string())?;
    if output.status.success() {
        Ok(String::from_utf8_lossy(&output.stdout).to_string())
    } else {
        let mut s = String::new();
        s.push_str(&String::from_utf8_lossy(&output.stdout));
        s.push_str(&String::from_utf8_lossy(&output.stderr));
        Err(s)
    }
}

pub fn command_available(cmd: &str) -> bool {
    Command::new("sh")
        .arg("-c")
        .arg(format!("command -v {} >/dev/null 2>&1", cmd))
        .status()
        .map(|s| s.success())
        .unwrap_or(false)
}

pub fn run_service_action(service: &str, action: &str) -> Result<String, String> {
    if !["start", "stop", "restart", "status"].contains(&action) {
        return Err("action not supported".into());
    }

    if command_available("systemctl") {
        return run_command("systemctl", &[action, service]);
    }
    if command_available("service") {
        return run_command("service", &[service, action]);
    }

    Err("service manager unavailable".into())
}

pub fn configure_interface(iface: &str, ip: &str, subnet: &str) -> Result<(), String> {
    let iface = if iface.is_empty() { "wlan0" } else { iface };
    let ip = if ip.is_empty() { "192.168.50.1" } else { ip };
    let subnet = if subnet.is_empty() { "24" } else { subnet };

    let _ = run_command("ip", &["addr", "flush", "dev", iface]);

    run_command("ip", &["addr", "add", &format!("{}/{}", ip, subnet), "dev", iface])
        .map_err(|e| format!("failed to assign IP: {}", e))?;

    run_command("ip", &["link", "set", iface, "up"]) .map_err(|e| format!("failed to bring up iface: {}", e))?;

    Ok(())
}

pub fn apply_firewall_rules(iface: &str, portal_ip: &str) -> Result<(), String> {
    let iface = if iface.is_empty() { "wlan0" } else { iface };
    let portal_ip = if portal_ip.is_empty() { "192.168.50.1" } else { portal_ip };

    fs::write("/proc/sys/net/ipv4/ip_forward", "1").map_err(|e| format!("failed to enable ip_forward: {}", e))?;

    // Find the real internet interface (WAN)
    let wan_iface = run_command("sh", &["-c", "ip route | grep default | grep -v " + iface + " | awk '{print $5}' | head -n 1"])
        .unwrap_or_default()
        .trim()
        .to_string();

    let dest_80 = format!("{}:80", portal_ip);
    let dest_443 = format!("{}:80", portal_ip);
    let dest_53 = format!("{}:53", portal_ip);

    // Clean up old rules by flushing our custom chains if they exist, or creating them
    let _ = run_command("iptables", &["-t", "nat", "-F", "AP1_NAT"]);
    let _ = run_command("iptables", &["-t", "nat", "-X", "AP1_NAT"]);
    let _ = run_command("iptables", &["-t", "nat", "-N", "AP1_NAT"]);

    // Jump to our chain from PREROUTING
    let _ = run_command("iptables", &["-t", "nat", "-D", "PREROUTING", "-j", "AP1_NAT"]);
    let _ = run_command("iptables", &["-t", "nat", "-I", "PREROUTING", "1", "-j", "AP1_NAT"]);

    let mut rules: Vec<Vec<&str>> = vec![
        vec!["-t", "nat", "-A", "AP1_NAT", "-i", iface, "-p", "tcp", "--dport", "80", "-j", "DNAT", "--to-destination", &dest_80],
        vec!["-t", "nat", "-A", "AP1_NAT", "-i", iface, "-p", "tcp", "--dport", "443", "-j", "DNAT", "--to-destination", &dest_443],
        vec!["-t", "nat", "-A", "AP1_NAT", "-i", iface, "-p", "udp", "--dport", "53", "-j", "DNAT", "--to-destination", &dest_53],
    ];

    if !wan_iface.is_empty() && wan_iface != iface {
        let _ = run_command("iptables", &["-t", "nat", "-D", "POSTROUTING", "-o", &wan_iface, "-j", "MASQUERADE"]);
        let _ = run_command("iptables", &["-t", "nat", "-I", "POSTROUTING", "1", "-o", &wan_iface, "-j", "MASQUERADE"]);
        println!("[*] NAT enabled: {} -> {}", iface, wan_iface);
    }

    for rule in rules.iter() {
        let args: Vec<&str> = rule.iter().copied().collect();
        if let Err(e) = run_command("iptables", &args) {
            eprintln!("warning: iptables rule failed ({}): {}", args.join(" "), e);
        }
    }

    Ok(())
}

pub fn clear_firewall_rules(iface: &str, _portal_ip: &str) -> Result<(), String> {
    let _ = run_command("iptables", &["-t", "nat", "-D", "PREROUTING", "-j", "AP1_NAT"]);
    let _ = run_command("iptables", &["-t", "nat", "-F", "AP1_NAT"]);
    let _ = run_command("iptables", &["-t", "nat", "-X", "AP1_NAT"]);

    let wan_iface = run_command("sh", &["-c", "ip route | grep default | grep -v " + iface + " | awk '{print $5}' | head -n 1"])
        .unwrap_or_default()
        .trim()
        .to_string();

    if !wan_iface.is_empty() {
        let _ = run_command("iptables", &["-t", "nat", "-D", "POSTROUTING", "-o", &wan_iface, "-j", "MASQUERADE"]);
    }

    Ok(())
}

pub fn write_profile_configs(hostapd_conf: &str, dnsmasq_conf: &str, runtime_dir: &str) -> Result<(), String> {
    let runtime = if runtime_dir.is_empty() { "../system/runtime" } else { runtime_dir };
    let path = Path::new(runtime);
    fs::create_dir_all(path).map_err(|e| format!("failed to create runtime dir: {}", e))?;

    fs::write(path.join("hostapd.conf"), hostapd_conf).map_err(|e| format!("failed to write hostapd.conf: {}", e))?;
    fs::write(path.join("dnsmasq.conf"), dnsmasq_conf).map_err(|e| format!("failed to write dnsmasq.conf: {}", e))?;

    Ok(())
}

pub fn restart_system_services() {
    println!("Restarting AP1 system services");

    match run_service_action("hostapd", "restart") {
        Ok(o) => println!("hostapd restarted: {}", o.trim()),
        Err(e) => eprintln!("hostapd restart warning: {}", e),
    }

    match run_service_action("dnsmasq", "restart") {
        Ok(o) => println!("dnsmasq restarted: {}", o.trim()),
        Err(e) => eprintln!("dnsmasq restart warning: {}", e),
    }
}

pub fn prepare_system_for_ap(iface: &str) -> Result<(), String> {
    println!("Preparing system for Rogue AP on {}", iface);

    // 1. Kill conflicting processes gracefully if possible, then forcefully
    if command_available("systemctl") {
        let _ = run_command("systemctl", &["stop", "wpa_supplicant"]);
        let _ = run_command("systemctl", &["stop", "hostapd"]);
        let _ = run_command("systemctl", &["stop", "dnsmasq"]);
    }

    let _ = run_command("pkill", &["-9", "hostapd"]);
    let _ = run_command("pkill", &["-9", "dnsmasq"]);
    let _ = run_command("pkill", &["-9", "wpa_supplicant"]);

    // 2. NetworkManager check
    if command_available("nmcli") {
        let _ = run_command("nmcli", &["device", "set", iface, "managed", "no"]);
        println!("NetworkManager: {} set to unmanaged", iface);
    }

    // 3. RFKill unblock
    if command_available("rfkill") {
        let _ = run_command("rfkill", &["unblock", "wifi"]);
    }

    // 4. Ensure interface is UP
    let _ = run_command("ip", &["link", "set", iface, "up"]);

    Ok(())
}

pub fn ensure_monitor_mode(iface: &str) -> Result<String, String> {
    // If the interface name ends with 'mon', assume it's already monitor
    if iface.ends_with("mon") {
        return Ok(iface.to_string());
    }

    // Check if a monitor interface already exists
    let output = run_command("iw", &["dev"]);
    if let Ok(out) = output {
        if out.contains(&format!("{}mon", iface)) {
            return Ok(format!("{}mon", iface));
        }
    }

    println!("Attempting to create virtual monitor interface for {}", iface);

    // Try to create a virtual monitor interface (supported by many modern drivers)
    let mon_iface = format!("{}mon", iface);
    let create_res = run_command("iw", &["dev", iface, "interface", "add", &mon_iface, "type", "monitor"]);

    if create_res.is_ok() {
        let _ = run_command("ip", &["link", "set", &mon_iface, "up"]);
        println!("Created virtual monitor interface: {}", mon_iface);
        return Ok(mon_iface);
    }

    // Fallback: put the interface itself in monitor mode
    println!("Virtual interface failed, putting {} in monitor mode directly", iface);
    let _ = run_command("ip", &["link", "set", iface, "down"]);
    let _ = run_command("iw", &["dev", iface, "set", "type", "monitor"]);
    let _ = run_command("ip", &["link", "set", iface, "up"]);

    Ok(iface.to_string())
}

pub fn restore_system_after_ap(iface: &str) -> Result<(), String> {
    println!("Restoring system settings for {}", iface);

    if command_available("nmcli") {
        let _ = run_command("nmcli", &["device", "set", iface, "managed", "yes"]);
    }

    Ok(())
}
