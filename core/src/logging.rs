//! Structured logging support for AP1 core.

use std::fs;
use std::fs::OpenOptions;
use std::io::Write;
use std::path::Path;

fn log_path() -> String {
    std::env::var("AP1_LOG_PATH").unwrap_or_else(|_| "../system/runtime/ap1.log".to_string())
}

pub fn init_logging() {
    let path = log_path();
    if let Some(parent) = Path::new(&path).parent() {
        if let Err(err) = fs::create_dir_all(parent) {
            eprintln!("failed to create log directory {}: {}", parent.display(), err);
            return;
        }
    }
    let _ = OpenOptions::new().create(true).append(true).open(&path);
    println!("AP1 logging inizializzato in {}", path);
}

pub fn log_event(event: &str) {
    let path = log_path();
    if let Ok(mut file) = OpenOptions::new().create(true).append(true).open(&path) {
        let _ = writeln!(file, "{}", event);
    }
}
