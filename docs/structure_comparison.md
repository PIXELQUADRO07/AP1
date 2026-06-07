# Confronto tra AP1 e WiFi-Pumpkin3

## WiFi-Pumpkin3 (repo ufficiale)

Stack principale:
- Python 3.8+
- Package `wifipumpkin3`
- `config/` con template captive portal, hostapd, dnsmasq, ecc.
- `setup.py` e `requirements.txt`
- Plugin e moduli Python interni
- `Dockerfile` per deploy container
- Interfaccia CLI con script `wp3`, `captiveflask`, `phishkin3`, `evilqr3`, `sslstrip3`

Struttura rilevata:
- `.github/`
- `config/`
- `wifipumpkin3/`
- `README.md`, `LICENSE.md`, `CHANGELOG.md`

## AP1 (progetto corrente)

Stack principale:
- Core engine in Rust
- API server in Go
- Interfaccia CLI nativa
- Sistema plugin con supporto Rust/WASM
- Wrapper OS per hostapd, dnsmasq, firewall, process management
- Containerizzazione Docker con Docker Compose

Struttura rilevata:
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

## Differenze chiave

- AP1 è una riscrittura architetturale completa, non un fork diretto.
- La struttura è più modulare e separa core, API e CLI in servizi distinti.
- AP1 usa Rust/Go invece di Python puro.
- L'approccio di configurazione è basato su YAML e servizi containerizzati.

## Stato corrente

- Il core Rust è compilabile e ora include i moduli `ap_manager`, `traffic_engine`, `proxy`, `system_control`, `utils` nel percorso `core/src/`.
- Il progetto AP1 ha tutti i blocchi di struttura richiesti presenti nel repository.
