# AP1 API

## Endpoints

### `GET /`
Messaggio base di salute dell'API.

### `GET /health`
Verifica se il server API è attivo.

### `GET /api/status`
Proxy verso il core Rust. Restituisce lo stato del core e la configurazione caricata.

### `GET /api/config`
Restituisce la configurazione globale `config/global.yaml` in formato JSON.

### `GET /api/profiles`
Restituisce la lista dei profili AP definiti in `config/global.yaml`.

### `POST /api/profiles/select`
Seleziona il profilo AP attivo e applica la configurazione runtime per `hostapd` e `dnsmasq`.

Richiesta JSON:

```json
{
  "profile": "default"
}
```

Risposta JSON:

```json
{
  "active_profile": "default",
  "details": "..."
}
```

### `POST /api/profiles/create`
Crea un nuovo profilo AP e salva la configurazione.

Richiesta JSON:

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
Aggiorna un profilo AP esistente.

Richiesta JSON:

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
Elimina un profilo AP.

Richiesta JSON:

```json
{
  "profile": "guest"
}
```

### `GET /api/plugins`
Restituisce la lista dei plugin disponibili.

### `POST /api/plugins/toggle`
Abilita o disabilita un plugin.

Richiesta JSON:

```json
{
  "name": "default-logger",
  "enabled": true
}
```

Risposta JSON:

```json
{
  "name": "default-logger",
  "type": "core",
  "enabled": true,
  "description": "Basic runtime logger for AP1 core events"
}
```

### `POST /api/plugins/start`
Avvia un plugin esterno in background.

Richiesta JSON:

```json
{
  "name": "captive-portal",
  "cmd": "/usr/bin/python3",
  "args": ["/opt/captive.py"]
}
```

### `POST /api/plugins/stop`
Interrompe il plugin eseguito in background.

Richiesta JSON:

```json
{
  "name": "captive-portal"
}
```

### `GET /api/interfaces`
Restituisce l'elenco delle interfacce di rete locali.

### `GET /api/recon/networks`
Esegue la scansione Wi-Fi sull'interfaccia specificata.

Query string:

- `iface` (opzionale, default: `wlan0`)

### `GET /api/portal/credentials`
Restituisce le credenziali catturate dal captive portal.

### `POST /api/system/<service>/<action>`
Controlla i servizi di sistema supportati, ad esempio `hostapd` e `dnsmasq`.

`action` può essere `start`, `stop`, `restart` o `status`.

## Utilizzo

- Avvia il core con `cd core && cargo run`
- Avvia l'API con `cd api && go run .`
- Avvia la CLI con `cd cli && go run . --help`

## Note

La configurazione caricata dall'API viene letta da `../config/global.yaml`.
