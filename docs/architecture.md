# AP1 Architecture

## Overview

AP1 is a modular framework for managing access points, proxies, and network tooling.

Main components:

- `core/` - Rust engine for AP management, routing, state, and low-level services
- `api/` - Go API server for REST orchestration and integration logic
- `plugins/` - extensible plugin system with Rust/WASM support
- `system/` - OS wrappers for hostapd, dnsmasq, nftables, and process management
- `config/` - global YAML config, AP profiles, and plugin definitions

## Component interaction

1. The CLI sends requests to the API server at `http://localhost:8080`.
2. The API server reads configuration from `config/global.yaml` and manages CRUD endpoints.
3. Some API endpoints proxy to the Rust core at `http://127.0.0.1:8081`.
4. The Rust core exposes `/status` and loads configuration on demand.
5. The CLI displays core status, configuration, and AP profiles.

## API endpoints

- `GET /health` - API server health
- `GET /api/status` - core status and configuration
- `GET /api/config` - global config JSON
- `GET /api/profiles` - available AP profiles
- `POST /api/profiles/select` - select active AP profile

## Deployment

- `make setup` installs and verifies dependencies
- `make core` starts the Rust core
- `make api` starts the Go API
- `make docker` starts services with Docker Compose
- `docker compose -f docker/docker-compose.yml up --build` starts `core` and `api` in Docker containers
