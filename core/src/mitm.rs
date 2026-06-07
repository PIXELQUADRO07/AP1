//! MITM / proxy support for AP1 core.

use crate::packet_capture;
use std::process::Command;

pub fn sniff_traffic(iface: &str) {
    let iface = if iface.is_empty() { "wlan0" } else { iface };
    println!("Avvio packet capture su {}", iface);
    if let Err(err) = packet_capture::start(iface, "../system/runtime/packet_capture.log") {
        eprintln!("packet capture failed: {}", err);
    } else {
        println!("packet capture running; logs in ../system/runtime/packet_capture.log");
    }
}

pub fn start_transparent_proxy(iface: &str) {
    let iface = if iface.is_empty() { "wlan0" } else { iface };
    if Command::new("which").arg("mitmproxy").output().map(|o| o.status.success()).unwrap_or(false) {
        println!("mitmproxy disponibile su {}: configura il traffico tramite iptables/nftables per reindirizzare su 8080", iface);
    } else {
        println!("mitmproxy non disponibile, installare mitmproxy per la funzione MITM");
    }
}

pub fn stop_transparent_proxy() {
    let _ = Command::new("pkill").arg("-f").arg("mitmproxy").status();
    packet_capture::stop();
    println!("arresto proxy trasparente e packet capture richiesti");
}

pub fn inject_payload() {
    println!("injection payload placeholder: implementazione di attacco MITM avanzato tramite packet replay o HTTP injection necessaria");
}

pub fn get_capture_count() -> u64 {
    packet_capture::count()
}
