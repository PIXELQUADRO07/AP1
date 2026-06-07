# Plugin Loader

This module contains a skeleton for the AP1 plugin loader.

## Purpose

- load Rust plugins from `plugins/core_plugins/`
- load WASM plugins from `plugins/wasm_plugins/`
- expose a common interface for integration with the core

## Structure

- `plugins/core_plugins/` - native Rust plugins
- `plugins/wasm_plugins/` - WebAssembly compiled plugins
- `plugins/plugin_loader/loader.rs` - loader examples and plugin metadata
