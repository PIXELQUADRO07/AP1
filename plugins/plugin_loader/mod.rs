//! AP1 plugin loader.
//!
//! This module exposes primitives for plugin discovery and loading.

pub mod loader;

pub use loader::{discover_plugins, load_plugin, PluginMeta};
