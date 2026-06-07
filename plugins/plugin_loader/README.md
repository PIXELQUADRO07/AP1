# Plugin Loader

Questo modulo contiene uno scheletro per il loader dei plugin AP1.

## Obiettivo

- caricare plugin Rust da `plugins/core_plugins/`
- caricare plugin WASM da `plugins/wasm_plugins/`
- esporre un'interfaccia comune per l'integrazione con il core

## Struttura

- `plugins/core_plugins/` - plugin Rust nativi
- `plugins/wasm_plugins/` - plugin compilati in WebAssembly
- `plugins/plugin_loader/loader.rs` - esempi di loader e metadata plugin
