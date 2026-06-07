//! Recon module.

use std::process::Command;

pub fn scan_ssids(iface: &str) {
    let iface = if iface.is_empty() { "wlan0" } else { iface };
    let output = Command::new("iwlist").arg(iface).arg("scan").output();
    match output {
        Ok(output) if output.status.success() => {
            let text = String::from_utf8_lossy(&output.stdout);
            for line in text.lines() {
                if line.contains("ESSID:") || line.contains("Quality=") || line.contains("Encryption key:") {
                    println!("{}", line.trim());
                }
            }
        }
        Ok(output) => eprintln!("scan failed: {}", String::from_utf8_lossy(&output.stderr)),
        Err(err) => eprintln!("failed to execute iwlist: {}", err),
    }
}

pub fn analyze_signal(iface: &str) {
    let iface = if iface.is_empty() { "wlan0" } else { iface };
    let output = Command::new("iwconfig").arg(iface).output();
    match output {
        Ok(output) if output.status.success() => {
            let text = String::from_utf8_lossy(&output.stdout);
            for line in text.lines() {
                if line.contains("Signal level") || line.contains("Link Quality") {
                    println!("{}", line.trim());
                }
            }
        }
        Ok(output) => eprintln!("signal analyze failed: {}", String::from_utf8_lossy(&output.stderr)),
        Err(err) => eprintln!("failed to execute iwconfig: {}", err),
    }
}
