//! Client session state machine support

use crate::logging;

#[derive(Debug)]
pub enum SessionState {
    Connected,
    CaptiveShown,
    Authenticated,
    Blocked,
}

pub fn transition(state: SessionState, event: &str) {
    let message = format!("transition: {:?} -> {}", state, event);
    logging::log_event(&message);
    println!("{}", message);
}
