//! Orchestrator module - start/stop modules

use std::collections::HashMap;
use std::sync::{Mutex, OnceLock};

static MODULES: OnceLock<Mutex<HashMap<String, bool>>> = OnceLock::new();

fn module_store() -> &'static Mutex<HashMap<String, bool>> {
    MODULES.get_or_init(|| Mutex::new(HashMap::new()))
}

pub fn start_module(name: &str) {
    let mut modules = module_store().lock().unwrap();
    modules.insert(name.to_string(), true);
    println!("module started: {}", name);
}

pub fn stop_module(name: &str) {
    let mut modules = module_store().lock().unwrap();
    modules.insert(name.to_string(), false);
    println!("module stopped: {}", name);
}

pub fn list_modules() {
    let modules = module_store().lock().unwrap();
    for (name, active) in modules.iter() {
        println!("{}: {}", name, if *active { "running" } else { "stopped" });
    }
}
