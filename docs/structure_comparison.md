# AP1 vs WiFi-Pumpkin3

## WiFi-Pumpkin3 (official repo)

Primary stack:
- Python 3.8+
- `wifipumpkin3` package
- `config/` with captive portal, hostapd, dnsmasq templates
- `setup.py` and `requirements.txt`
- internal Python plugins and modules
- `Dockerfile` for container deployment
- CLI scripts like `wp3`, `captiveflask`, `phishkin3`, `evilqr3`, `sslstrip3`

Detected structure:
- `.github/`
- `config/`
- `wifipumpkin3/`
- `README.md`, `LICENSE.md`, `CHANGELOG.md`

## AP1 (current project)

Primary stack:
- Rust core engine
- Go API server
- native CLI
- plugin system with Rust/WASM support
- OS wrappers for hostapd, dnsmasq, firewall, and process management
- Docker containerization with Docker Compose

Detected structure:
- `core/`
  - `src/main.rs`
  - `src/ap_manager/`
  - `src/traffic_engine/`
  - `src/proxy/`
  - `src/system_control/`
  - `src/utils/`
- `api/`
  - `routes/`
  - `handlers/`
  - `websocket/`
  - `services/`
  - `middleware/`
- `plugins/`
  - `core_plugins/`
  - `wasm_plugins/`
  - `plugin_loader/`
- `system/`
  - `network/`
  - `services/`
  - `firewall/`
  - `process_manager/`
- `config/`
  - `global.yaml`
  - `plugins.yaml`
  - `ap_profiles/`
- `docker/`
  - `Dockerfile.api`
  - `Dockerfile.core`
  - `docker-compose.yml`
  - `scripts/`

## Key differences

- AP1 is a ground-up architectural rewrite, not a direct fork.
- The structure separates core, API, and CLI into distinct services.
- AP1 uses Rust/Go instead of pure Python.
- Configuration is YAML-based with service and container support.

## Current status

- The Rust core is buildable and includes modules such as `ap_manager`, `traffic_engine`, `proxy`, `system_control`, and `utils`.
- The AP1 repository contains the expected structural components.
