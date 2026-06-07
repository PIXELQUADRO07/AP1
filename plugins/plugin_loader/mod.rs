//! Loader di plugin AP1.
//!
//! Questo modulo espone le primitive per la scoperta e il caricamento dei plugin.

pub mod loader;

pub use loader::{discover_plugins, load_plugin, PluginMeta};
