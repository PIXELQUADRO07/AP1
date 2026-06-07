//! Interfaces module.
//! Gestione delle interfacce di rete.

use std::process::Command;

pub fn list_interfaces() {
    match Command::new("ip").arg("-o").arg("link").arg("show").output() {
        Ok(output) if output.status.success() => {
            let text = String::from_utf8_lossy(&output.stdout);
            for line in text.lines() {
                if let Some(colon) = line.find(':') {
                    let iface = line[colon + 2..].split(':').next().unwrap_or_default();
                    println!("{}", iface.trim());
                }
            }
        }
        Ok(output) => {
            eprintln!("failed to list interfaces: {}", String::from_utf8_lossy(&output.stderr));
        }
        Err(err) => {
            eprintln!("failed to execute ip: {}", err);
        }
    }
}

pub fn set_monitor_mode(iface: &str, enable: bool) {
    let mode = if enable { "monitor" } else { "managed" };
    let _ = Command::new("ip").arg("link").arg("set").arg(iface).arg("down").status();
    let status = Command::new("iw").arg("dev").arg(iface).arg("set").arg("type").arg(mode).status();
    if status.map(|s| s.success()).unwrap_or(false) {
        let _ = Command::new("ip").arg("link").arg("set").arg(iface).arg("up").status();
        println!("set {} to {} mode", iface, mode);
    } else {
        eprintln!("failed to change {} to {} mode", iface, mode);
    }
}

pub fn bring_up(iface: &str) {
    match Command::new("ip").arg("link").arg("set").arg(iface).arg("up").status() {
        Ok(status) if status.success() => println!("interface {} is up", iface),
        Ok(_) => eprintln!("failed to bring up interface {}", iface),
        Err(err) => eprintln!("failed to execute ip: {}", err),
    }
}
