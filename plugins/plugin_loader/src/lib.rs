//! AP1 Plugin Loader
//!
//! This crate provides discovery and loading of plugins based on YAML configuration.

pub mod loader;

pub use loader::{discover_plugins, load_plugin, load_plugins_from_config, PluginMeta};
