//! Layered networking stack abstraction.

use crate::system_control::{apply_firewall_rules, clear_firewall_rules};

pub fn setup_routing() {
    if let Err(err) = apply_firewall_rules("wlan0", "192.168.50.1") {
        eprintln!("failed to set up routing: {}", err);
    } else {
        println!("routing setup completed");
    }
}

pub fn teardown_routing() {
    if let Err(err) = clear_firewall_rules("wlan0", "192.168.50.1") {
        eprintln!("failed to tear down routing: {}", err);
    } else {
        println!("routing rules cleared");
    }
}
