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

    let dest_80 = format!("{}:80", portal_ip);
    let dest_443 = format!("{}:80", portal_ip);
    let dest_53 = format!("{}:53", portal_ip);

    let rules: Vec<Vec<&str>> = vec![
        vec!["-t", "nat", "-A", "POSTROUTING", "-o", iface, "-j", "MASQUERADE"],
        vec!["-t", "nat", "-A", "PREROUTING", "-i", iface, "-p", "tcp", "--dport", "80", "-j", "DNAT", "--to-destination", &dest_80],
        vec!["-t", "nat", "-A", "PREROUTING", "-i", iface, "-p", "tcp", "--dport", "443", "-j", "DNAT", "--to-destination", &dest_443],
        vec!["-t", "nat", "-A", "PREROUTING", "-i", iface, "-p", "udp", "--dport", "53", "-j", "DNAT", "--to-destination", &dest_53],
    ];

    for rule in rules.iter() {
        let args: Vec<&str> = rule.iter().copied().collect();
        if let Err(e) = run_command("iptables", &args) {
            eprintln!("warning: iptables rule failed ({}): {}", args.join(" "), e);
        }
    }

    Ok(())
}

pub fn clear_firewall_rules(iface: &str, portal_ip: &str) -> Result<(), String> {
    let iface = if iface.is_empty() { "wlan0" } else { iface };
    let portal_ip = if portal_ip.is_empty() { "192.168.50.1" } else { portal_ip };

    let dest_80 = format!("{}:80", portal_ip);
    let dest_443 = format!("{}:80", portal_ip);
    let dest_53 = format!("{}:53", portal_ip);

    let rules: Vec<Vec<&str>> = vec![
        vec!["-t", "nat", "-D", "POSTROUTING", "-o", iface, "-j", "MASQUERADE"],
        vec!["-t", "nat", "-D", "PREROUTING", "-i", iface, "-p", "tcp", "--dport", "80", "-j", "DNAT", "--to-destination", &dest_80],
        vec!["-t", "nat", "-D", "PREROUTING", "-i", iface, "-p", "tcp", "--dport", "443", "-j", "DNAT", "--to-destination", &dest_443],
        vec!["-t", "nat", "-D", "PREROUTING", "-i", iface, "-p", "udp", "--dport", "53", "-j", "DNAT", "--to-destination", &dest_53],
    ];

    for rule in rules.iter() {
        let args: Vec<&str> = rule.iter().copied().collect();
        if let Err(e) = run_command("iptables", &args) {
            eprintln!("warning: failed to clear rule ({}): {}", args.join(" "), e);
        }
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
