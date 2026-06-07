//! HTTPS / SNI detection scaffolding

use std::process::Command;

pub fn detect_sni(host: &str) {
    let output = Command::new("openssl").arg("s_client").arg("-connect").arg(format!("{}:443", host)).arg("-servername").arg(host).output();
    match output {
        Ok(output) if output.status.success() => {
            println!("HTTPS negotiation ok for {}", host);
        }
        Ok(output) => {
            eprintln!("HTTPS detection failed: {}", String::from_utf8_lossy(&output.stderr));
        }
        Err(err) => {
            eprintln!("failed to execute openssl: {}", err);
        }
    }
}
