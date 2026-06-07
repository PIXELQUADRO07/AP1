# Setup AP1

## Prerequisites

- Rust 1.71+
- Go 1.21+
- Docker (optional)

## Start the core

```bash
cd core
cargo run
```

If you want to pass explicit config paths:

```bash
AP1_CONFIG_PATH=../config/global.yaml AP1_PLUGIN_CONFIG_PATH=../config/plugins.yaml cargo run
```

## Start the API

```bash
cd api
go run .
```

You can also use flags or environment variables:

```bash
go run . -config ../config/global.yaml -plugins ../config/plugins.yaml -addr :8080
```

## Start the CLI

```bash
cd cli
go run . --help
```

## Full setup

Run the bootstrap script:

```bash
./install.sh
```

## Quick start with a single command

From the project root you can start the core and API in the background and open the CLI with:

```bash
./ap1
```

The first time it will try to install `ap1` into a PATH directory (`/usr/local/bin` or `~/.local/bin`).

Or explicitly:

```bash
./ap1 start
```

## Available API routes

- `GET /health` - API server health
- `GET /api/status` - core status and configuration
- `GET /api/config` - global config JSON
- `GET /api/profiles` - AP profiles list
- `POST /api/profiles/select` - select the active AP profile and apply runtime config
- `GET /api/plugins` - available plugins
- `POST /api/plugins/toggle` - enable/disable a plugin
- `POST /api/system/hostapd/<action>` - control hostapd
- `POST /api/system/dnsmasq/<action>` - control dnsmasq

## Configuration

Edit `config/global.yaml` to change API, core, and networking options.

## Import templates from another template repo

If you have a local copy of a compatible template repository, you can import `config/templates` from the project root:

```bash
make import-templates
# or
sh docker/scripts/import_templates.sh /path/to/source/config/templates $(pwd)/config/templates
```

This copies templates into the project `config/templates` folder.

## Additional notes

See `docs/api.md` for API route documentation.
