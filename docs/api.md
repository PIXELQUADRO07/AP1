# AP1 API

## Endpoints

### `GET /`
A simple health message from the API.

### `GET /health`
Checks whether the API server is running.

### `GET /api/status`
Proxies to the Rust core and returns core status plus loaded configuration.

### `GET /api/config`
Returns the global configuration from `config/global.yaml` as JSON.

### `GET /api/profiles`
Returns the list of AP profiles defined in `config/global.yaml`.

### `POST /api/profiles/select`
Selects the active AP profile and applies runtime configuration for `hostapd` and `dnsmasq`.

Request JSON:

```json
{
  "profile": "default"
}
```

Response JSON:

```json
{
  "active_profile": "default",
  "details": "..."
}
```

### `POST /api/profiles/create`
Creates a new AP profile and saves configuration.

Request JSON:

```json
{
  "name": "guest",
  "ssid": "AP1-Guest",
  "password": "guestpass",
  "channel": 11,
  "mode": "n",
  "dhcp_enabled": true
}
```

### `PUT /api/profiles/update`
Updates an existing AP profile.

Request JSON:

```json
{
  "name": "guest",
  "ssid": "AP1-Guest",
  "password": "newpass",
  "channel": 6,
  "mode": "g",
  "dhcp_enabled": true
}
```

### `DELETE /api/profiles/delete`
Deletes an AP profile.

Request JSON:

```json
{
  "profile": "guest"
}
```

### `GET /api/plugins`
Returns the list of available plugins.

### `POST /api/plugins/toggle`
Enables or disables a plugin.

Request JSON:

```json
{
  "name": "default-logger",
  "enabled": true
}
```

### `POST /api/plugins/start`
Starts an external plugin in the background.

Request JSON:

```json
{
  "name": "captive-portal",
  "cmd": "/usr/bin/python3",
  "args": ["/opt/captive.py"]
}
```

### `POST /api/plugins/stop`
Stops a running plugin.

Request JSON:

```json
{
  "name": "captive-portal"
}
```

### `GET /api/interfaces`
Returns the list of local network interfaces.

### `GET /api/recon/networks`
Performs a Wi-Fi scan on the specified interface.

Query string:

- `iface` (optional, default: `wlan0`)

### `GET /api/portal/credentials`
Returns credentials captured by the captive portal.

### `POST /api/system/<service>/<action>`
Controls supported system services such as `hostapd` and `dnsmasq`.

`action` can be `start`, `stop`, `restart`, or `status`.

## Usage

- Start the core with `cd core && cargo run`
- Start the API with `cd api && go run .`
- Start the CLI with `cd cli && go run . --help`

## Notes

The API reads configuration from `../config/global.yaml`.
