//! Beacon Flooding module for creating multiple fake SSIDs.

use std::process::Command;
use std::thread;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use crate::system_control;

pub struct BeaconFlood {
    running: Arc<AtomicBool>,
}

impl BeaconFlood {
    pub fn new() -> Self {
        Self {
            running: Arc::new(AtomicBool::new(false)),
        }
    }

    pub fn start(&self, iface: &str, ssids: Vec<String>) {
        if self.running.load(Ordering::SeqCst) {
            return;
        }

        let effective_iface = system_control::ensure_monitor_mode(iface).unwrap_or(iface.to_string());

        self.running.store(true, Ordering::SeqCst);
        let running = self.running.clone();
        let iface_str = effective_iface.clone();

        thread::spawn(move || {
            println!("Beacon Flooding started on {} with {} SSIDs", iface_str, ssids.len());

            let ssids_str = ssids.join("\n");
            let ssid_file = "../system/runtime/fake_ssids.txt";
            let _ = std::fs::write(ssid_file, ssids_str);

            let child = Command::new("mdk4")
                .args([&iface_str, "b", "-f", ssid_file])
                .spawn();

            while running.load(Ordering::SeqCst) {
                thread::sleep(std::time::Duration::from_secs(1));
            }

            if let Ok(mut c) = child {
                let _ = c.kill();
            }
            println!("Beacon Flooding stopped.");
        });
    }

    pub fn stop(&self) {
        self.running.store(false, Ordering::SeqCst);
    }
}
