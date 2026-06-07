# AP1

AP1 is a modular orchestrator for captive portals and AP management, built with a Rust core, Go API, and native CLI.

Repo structure:

- `core/` - primary engine in Rust with captive portal runtime, packet capture, and plugin management
- `api/` - Go API server
- `cli/` - command-line client for managing profiles, plugins, and services
- `plugins/` - plugin system with Rust/WASM support
- `system/` - OS wrappers and network integrations
- `config/` - YAML configuration and portal templates
- `docker/` - containerization and deployment scripts
- `docs/` - architecture and setup documentation

## Recommended steps

1. Start the Rust core:
   ```bash
   cd core
   cargo run
   ```
   Or use explicit config paths:
   ```bash
   AP1_CONFIG_PATH=../config/global.yaml AP1_PLUGIN_CONFIG_PATH=../config/plugins.yaml cargo run
   ```
2. Start the Go API server:
   ```bash
   cd ../api
   go run .
   ```
   Or pass flags:
   ```bash
   go run . -config ../config/global.yaml -plugins ../config/plugins.yaml -addr :8080
   ```
3. Start the AP1 CLI:
   ```bash
   cd cli
   go run . --help
   ```
4. Or use the Makefile:
   ```bash
   make core
   make api
   ```
5. To install dependencies and bootstrap the project:
   ```bash
   ./install.sh
   ```
6. To start all components from the repo root with one command and open the interactive CLI:
   ```bash
   ./ap1
   ```
   On first run, the command attempts to install `ap1` into your PATH so you can run it from any directory.
   Alternatively:
   ```bash
   ./ap1 start
   ```

## Available endpoints

- `GET /health` - API server health
- `GET /api/status` - core status and configuration
- `GET /api/config` - global configuration JSON
- `GET /api/profiles` - AP profile list
- `POST /api/profiles/select` - select active AP profile and apply runtime config for `hostapd`/`dnsmasq`
- `GET /api/plugins` - available plugins
- `POST /api/plugins/toggle` - enable or disable a plugin
- `POST /api/plugins/start` - start an external plugin
- `POST /api/plugins/stop` - stop a running plugin
- `GET /api/interfaces` - local network interfaces
- `GET /api/recon/networks?iface=<iface>` - Wi-Fi scan on an interface
- `GET /api/portal/status` - captive portal status
- `GET /api/portal/credentials` - captured portal credentials
- `POST /api/system/hostapd/<action>` - manage hostapd
- `POST /api/system/dnsmasq/<action>` - manage dnsmasq
- `POST /api/system/firewall/apply` - apply captive portal firewall rules
- `POST /api/system/firewall/clear` - clear firewall rules
- `POST /api/system/interface/configure` - assign IP/subnet to an interface
- `GET /status` - core status

## Current status

This MVP includes a CLI that manages the API server and Rust core, with AP profile support, plugin control, and hostapd/dnsmasq service management.
