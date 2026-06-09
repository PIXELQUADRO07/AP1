//! Traffic engine module.
//!
//! Gestisce la logica di routing interno e proxy dei pacchetti.

use crate::system_control::{apply_firewall_rules, apply_nat_rules, clear_firewall_rules};

pub fn start_traffic_engine(iface: &str, portal_ip: &str) {
    let iface = if iface.is_empty() { "wlan0" } else { iface };
    let portal_ip = if portal_ip.is_empty() { "192.168.50.1" } else { portal_ip };
    if let Err(err) = apply_firewall_rules(iface, portal_ip) {
        eprintln!("failed to start traffic engine: {}", err);
    } else {
        println!("traffic engine started on {} -> {}", iface, portal_ip);
    }
}

pub fn configure_nat(iface: &str) {
    let iface = if iface.is_empty() { "wlan0" } else { iface };
    if let Err(err) = apply_nat_rules(iface) {
        eprintln!("failed to configure NAT: {}", err);
    } else {
        println!("NAT rules applied for {}", iface);
    }
}

pub fn start_nat(iface: &str) {
    configure_nat(iface);
}

pub fn stop_nat(iface: &str) {
    let iface = if iface.is_empty() { "wlan0" } else { iface };
    if let Err(err) = clear_firewall_rules(iface, "192.168.50.1") {
        eprintln!("failed to stop NAT: {}", err);
    } else {
        println!("NAT rules removed from {}", iface);
    }
}

pub fn stop_traffic_engine(iface: &str, portal_ip: &str) {
    let iface = if iface.is_empty() { "wlan0" } else { iface };
    let portal_ip = if portal_ip.is_empty() { "192.168.50.1" } else { portal_ip };
    if let Err(err) = clear_firewall_rules(iface, portal_ip) {
        eprintln!("failed to stop traffic engine: {}", err);
    } else {
        println!("traffic engine stopped on {}", iface);
    }
}
