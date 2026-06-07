# AP1

AP1 è un orchestratore modulare per captive portal e gestione AP, basato su core Rust, API Go e una CLI nativa.

Struttura iniziale:

- `core/` - engine principale in Rust con runtime captive portal, packet capture C++/FFI e plugin management
- `api/` - server API in Go
- `cli/` - client da riga di comando per gestire profili, plugin e servizi
- `plugins/` - sistema plugin Rust/WASM
- `system/` - wrapper OS e integrazioni di rete
- `config/` - configurazioni YAML/TOML
- `docker/` - containerizzazione e script di deploy
- `docs/` - documentazione di architettura e setup

## Passi successivi consigliati

1. Avviare il core Rust:
   ```bash
   cd core
   cargo run
   ```
   Oppure impostare percorsi di configurazione personalizzati:
   ```bash
   AP1_CONFIG_PATH=../config/global.yaml AP1_PLUGIN_CONFIG_PATH=../config/plugins.yaml cargo run
   ```
2. Avviare l'API server Go:
   ```bash
   cd ../api
   go run .
   ```
   Oppure passare i percorsi con flag:
   ```bash
   go run . -config ../config/global.yaml -plugins ../config/plugins.yaml -addr :8080
   ```
3. Avviare la CLI AP1:
   ```bash
   cd cli
   go run . --help
   ```
4. Oppure utilizzare il Makefile:
   ```bash
   make core
   make api
   ```
5. Per installare le dipendenze e verificare il progetto:
   ```bash
   ./install.sh
   ```
6. Per avviare tutti i componenti dalla root con un singolo comando e aprire la CLI interattiva:
   ```bash
   ./ap1
   ```
   Oppure, per un avvio esplicito:
   ```bash
   ./ap1 start
   ```

## Endpoint disponibili

- `GET /health` - stato server API
- `GET /api/status` - stato del core e configurazione
- `GET /api/config` - configurazione globale in JSON
- `GET /api/profiles` - lista dei profili AP definiti
- `POST /api/profiles/select` - seleziona e applica il profilo AP attivo, generando configurazioni `hostapd`/`dnsmasq`
- `GET /api/plugins` - lista i plugin disponibili
- `POST /api/plugins/toggle` - abilita o disabilita un plugin
- `POST /api/plugins/start` - avvia un plugin esterno con nome e comando
- `POST /api/plugins/stop` - interrompe un plugin avviato in background
- `GET /api/interfaces` - elenco delle interfacce di rete del sistema
- `GET /api/recon/networks?iface=<iface>` - scansione reti Wi-Fi su un'interfaccia
- `GET /api/portal/status` - stato del captive portal
- `GET /api/portal/credentials` - elenco credenziali catturate dal portal
- `POST /api/system/hostapd/<action>` - gestisce hostapd (`start`, `stop`, `restart`, `status`)
- `POST /api/system/dnsmasq/<action>` - gestisce dnsmasq (`start`, `stop`, `restart`, `status`)
- `POST /api/system/firewall/apply` - applica regole captive portal su un'interfaccia
- `POST /api/system/firewall/clear` - cancella le regole firewall del captive portal
- `POST /api/system/interface/configure` - assegna IP/subnet a un'interfaccia
- `GET /status` - stato del core

## Stato attuale

Questa base MVP include una CLI che gestisce il server API e il core Rust, con supporto profili AP, plugin e controllo servizio `hostapd`/`dnsmasq`.
