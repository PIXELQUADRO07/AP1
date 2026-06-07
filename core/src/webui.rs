//! Web UI support for AP1 core.

use std::io::{Read, Write};
use std::net::TcpListener;
use std::thread;

fn index_page() -> String {
    let body = r#"<!DOCTYPE html>
<html lang='en'>
<head><meta charset='UTF-8'><title>AP1 Web UI</title></head>
<body>
<h1>AP1 Web UI</h1>
<ul>
<li><a href='/status'>Stato core</a></li>
<li><a href='http://127.0.0.1:8080'>Captive Portal</a></li>
</ul>
</body>
</html>"#;
    format!("HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{}", body.len(), body)
}

fn status_page() -> String {
    let body = r#"{"status":"AP1 core active","webui":"running"}"#;
    format!("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{}", body.len(), body)
}

pub fn start_webui() {
    let listener = TcpListener::bind("127.0.0.1:8082");
    match listener {
        Ok(listener) => {
            thread::spawn(move || {
                println!("Web UI avviato su http://127.0.0.1:8082");
                for stream in listener.incoming() {
                    if let Ok(mut stream) = stream {
                        let mut buffer = [0; 1024];
                        if let Ok(size) = stream.read(&mut buffer) {
                            let request = String::from_utf8_lossy(&buffer[..size]);
                            if request.starts_with("GET /status") {
                                let _ = stream.write_all(status_page().as_bytes());
                            } else {
                                let _ = stream.write_all(index_page().as_bytes());
                            }
                        }
                    }
                }
            });
        }
        Err(err) => {
            eprintln!("impossibile avviare Web UI: {}", err);
        }
    }
}

pub fn stop_webui() {
    println!("Web UI stop richiesto (non supportato nel demo corrente)");
}

pub fn ui_status() {
    println!("Web UI status: running");
}
