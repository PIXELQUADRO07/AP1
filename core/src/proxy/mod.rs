//! Proxy module.
//!
//! Questo modulo ospita i componenti di proxying e forward del traffico.

use std::process::Command;

pub fn init_proxy() {
    if Command::new("which").arg("mitmproxy").output().map(|o| o.status.success()).unwrap_or(false) {
        println!("mitmproxy disponibile: avvio del proxy trasparente richiesto");
    } else if Command::new("which").arg("tinyproxy").output().map(|o| o.status.success()).unwrap_or(false) {
        println!("tinyproxy disponibile: proxy HTTP pronto per essere configurato");
    } else {
        println!("Nessun proxy HTTP/mitm installato. Installare mitmproxy o tinyproxy per il supporto completo.");
    }
}
