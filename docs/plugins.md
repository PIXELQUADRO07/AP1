# Plugin System

AP1 supports a modular plugin system to extend the core.

## Architecture

- `plugins/core_plugins/` contains Rust plugins that can be loaded by the core.
- `plugins/wasm_plugins/` contains plugins compiled for WebAssembly.
- `plugins/plugin_loader/` contains a plugin loader and a plugin model.
- `config/plugins.yaml` defines available plugins and their `enabled` state.

## Configuration

The file `config/plugins.yaml` uses the format:

```yaml
plugins:
  - name: default-logger
    type: core
    enabled: true
    description: Basic runtime logger for AP1 core events
```

## API Endpoints

- `GET /api/plugins` - list available plugins
- `POST /api/plugins/toggle` - enable or disable a plugin

JSON request:

```json
{
  "name": "default-logger",
  "enabled": true
}
```

## Usage

The API plugin loader can read `config/plugins.yaml` and decide which plugins to load.
The core plugin loader can extend the network pipeline with Rust or WASM plugins.
