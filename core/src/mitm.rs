//! MITM / proxy support for AP1 core.

use crate::packet_capture;
use std::process::Command;

pub fn sniff_traffic(iface: &str) {
    let iface = if iface.is_empty() { "wlan0" } else { iface };
    println!("Starting packet capture on {}", iface);
    if let Err(err) = packet_capture::start(iface, "../system/runtime/packet_capture.log") {
        eprintln!("packet capture failed: {}", err);
    } else {
        println!("packet capture running; logs in ../system/runtime/packet_capture.log");
    }
}

pub fn start_transparent_proxy(iface: &str) {
    let iface = if iface.is_empty() { "wlan0" } else { iface };
    if Command::new("which").arg("mitmproxy").output().map(|o| o.status.success()).unwrap_or(false) {
        println!("mitmproxy available on {}: configure traffic via iptables/nftables to redirect to 8080", iface);
    } else {
        println!("mitmproxy not available, install mitmproxy for MITM support");
    }
}

pub fn stop_transparent_proxy() {
    let _ = Command::new("pkill").arg("-f").arg("mitmproxy").status();
    packet_capture::stop();
    println!("transparent proxy and packet capture stop requested");
}

pub fn inject_payload() {
    println!("injection payload placeholder: advanced MITM attack implementation via packet replay or HTTP injection required");
}

pub fn get_capture_count() -> u64 {
    packet_capture::count()
}
