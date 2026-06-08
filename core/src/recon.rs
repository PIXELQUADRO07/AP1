//! Recon module for scanning and target identification.

use std::process::Command;
use crate::config::Profile;

#[derive(Debug, Clone)]
pub struct WiFiNetwork {
    pub ssid: String,
    pub bssid: String,
    pub channel: i32,
    pub signal: String,
}

pub fn scan_targets(iface: &str) -> Vec<WiFiNetwork> {
    let iface = if iface.is_empty() { "wlan0" } else { iface };

    // Attempt nmcli first
    let mut networks = scan_with_nmcli(iface);

    // If nmcli fails or returns nothing, fallback to iwlist
    if networks.is_empty() {
        println!("[!] nmcli scan failed or empty, falling back to iwlist...");
        networks = scan_with_iwlist(iface);
    }

    networks
}

fn scan_with_nmcli(iface: &str) -> Vec<WiFiNetwork> {
    let mut networks = Vec::new();
    let output = Command::new("nmcli")
        .args(["-t", "-f", "SSID,BSSID,CHAN,SIGNAL", "dev", "wifi", "list", "ifname", iface])
        .output();

    if let Ok(out) = output {
        let text = String::from_utf8_lossy(&out.stdout);
        for line in text.lines() {
            let line_unjailed = line.replace("\\:", "____COLON____");
            let parts: Vec<&str> = line_unjailed.split(':').collect();
            if parts.len() >= 4 {
                let ssid = parts[0].replace("____COLON____", ":");
                let bssid = parts[1].replace("____COLON____", ":");
                let channel = parts[2].parse().unwrap_or(1);
                let signal = parts[3].to_string();

                networks.push(WiFiNetwork { ssid, bssid, channel, signal });
            }
        }
    }
    networks
}

fn scan_with_iwlist(iface: &str) -> Vec<WiFiNetwork> {
    let mut networks = Vec::new();
    let output = Command::new("iwlist").args([iface, "scan"]).output();

    if let Ok(out) = output {
        let text = String::from_utf8_lossy(&out.stdout);
        let mut current_net = WiFiNetwork { ssid: "".into(), bssid: "".into(), channel: 1, signal: "".into() };

        for line in text.lines() {
            let line = line.trim();
            if line.contains("Cell") {
                if !current_net.bssid.is_empty() {
                    networks.push(current_net.clone());
                }
                if let Some(pos) = line.find("Address: ") {
                    current_net.bssid = line[pos + 9..].trim().to_string();
                }
            } else if line.contains("ESSID:") {
                current_net.ssid = line.split(':').nth(1).unwrap_or("").trim_matches('"').to_string();
            } else if line.contains("Channel:") {
                if let Some(pos) = line.find("Channel:") {
                     current_net.channel = line[pos+8..].trim_matches(|c: char| !c.is_digit(10)).parse().unwrap_or(1);
                }
            } else if line.contains("Signal level=") {
                if let Some(pos) = line.find("Signal level=") {
                    current_net.signal = line[pos+13..].split_whitespace().next().unwrap_or("").to_string();
                }
            }
        }
        if !current_net.bssid.is_empty() {
            networks.push(current_net);
        }
    }
    networks
}

pub fn profile_from_network(net: &WiFiNetwork) -> Profile {
    Profile {
        name: format!("cloned_{}", net.ssid),
        ssid: net.ssid.clone(),
        password: "".to_string(), // Evil twins are usually Open to attract victims
        channel: net.channel,
        mode: "g".to_string(),
        dhcp_enabled: true,
        security: Some("open".to_string()),
    }
}

pub fn analyze_congestion(iface: &str) {
    let networks = scan_targets(iface);
    let mut channel_counts = std::collections::HashMap::new();

    for net in networks {
        *channel_counts.entry(net.channel).or_insert(0) += 1;
    }

    println!("\n[*] Channel Congestion Analysis:");
    println!("-------------------------------");
    for ch in 1..=13 {
        let count = channel_counts.get(&ch).unwrap_or(&0);
        let bar = "#".repeat(*count as usize);
        println!("CH {:2} | {:2} networks {}", ch, count, bar);
    }
}
