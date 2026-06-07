# AP1 Architecture

## Overview

AP1 è un framework modulare per la gestione di access point, proxy e strumenti di rete.

Componenti principali:

- `core/` - engine Rust per gestione AP, routing, stato e servizi di basso livello
- `api/` - server Go per esporre REST, orchestrazione e logica di integrazione
- `plugins/` - sistema plugin estendibile in Rust/WASM
- `system/` - wrapper OS per hostapd, dnsmasq, nftables e gestione processi
- `config/` - configurazione YAML globale, profili AP e plugin

## Component Interaction

1. La CLI invia richieste all'API server su `http://localhost:8080`.
2. L'API server legge la configurazione da `config/global.yaml` e gestisce endpoint CRUD.
3. Alcuni endpoint API fanno proxy verso il core Rust su `http://127.0.0.1:8081`.
4. Il core Rust espone `/status` e carica la configurazione al volo.
5. La CLI mostra lo stato del core, la configurazione e i profili AP.

## API Endpoints

- `GET /health` - stato del server API
- `GET /api/status` - stato del core e configurazione
- `GET /api/config` - configurazione globale in JSON
- `GET /api/profiles` - profili AP disponibili
- `POST /api/profiles/select` - seleziona il profilo AP attivo

## Deployment

- `make setup` installa e verifica dipendenze
- `make core` avvia il core Rust
- `make api` avvia l'API Go
- `make docker` avvia i servizi con Docker Compose
- `docker compose -f docker/docker-compose.yml up --build` avvia `core` e `api` in container Docker
