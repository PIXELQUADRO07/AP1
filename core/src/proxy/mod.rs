//! Proxy module.
//!
//! This module hosts proxying and traffic forwarding components.

use std::process::Command;

pub fn init_proxy() {
    if Command::new("which").arg("mitmproxy").output().map(|o| o.status.success()).unwrap_or(false) {
        println!("mitmproxy available: transparent proxy startup requested");
    } else if Command::new("which").arg("tinyproxy").output().map(|o| o.status.success()).unwrap_or(false) {
        println!("tinyproxy available: HTTP proxy ready to be configured");
    } else {
        println!("No HTTP/mitm proxy installed. Install mitmproxy or tinyproxy for full support.");
    }
}
