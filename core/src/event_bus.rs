//! Event bus - observer pattern support for AP1 core.

use std::sync::{Mutex, OnceLock};

pub type Event = String;

static EVENT_BUS: OnceLock<Mutex<Vec<Event>>> = OnceLock::new();

fn event_store() -> &'static Mutex<Vec<Event>> {
    EVENT_BUS.get_or_init(|| Mutex::new(Vec::new()))
}

pub fn emit(event: &Event) {
    if let Ok(mut store) = event_store().lock() {
        store.push(event.clone());
    }
    println!("event emitted: {}", event);
}

pub fn subscribe(topic: &str) {
    println!("subscribed to topic: {}", topic);
}
