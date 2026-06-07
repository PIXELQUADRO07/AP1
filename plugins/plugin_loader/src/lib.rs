//! AP1 Plugin Loader
//!
//! Questo crate fornisce la scoperta e il caricamento dei plugin basati sulla configurazione YAML.

pub mod loader;

pub use loader::{discover_plugins, load_plugin, load_plugins_from_config, PluginMeta};
