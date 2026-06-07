//! Utility module.
//!
//! Helper generici riutilizzabili in tutto il core AP1.

pub fn format_status(message: &str) -> String {
    format!("[AP1] {}", message)
}
