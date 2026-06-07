# Plugin System

AP1 supporta un sistema di plugin modulari per estendere il core.

## Architettura

- `plugins/core_plugins/` contiene plugin Rust che possono essere caricati dal core.
- `plugins/wasm_plugins/` contiene plugin compilati per WebAssembly.
- `plugins/plugin_loader/` contiene un loader di plugin e un modello per i plugin.
- `config/plugins.yaml` definisce i plugin disponibili e il loro stato `enabled`.

## Configurazione

Il file `config/plugins.yaml` ha il formato:

```yaml
plugins:
  - name: default-logger
    type: core
    enabled: true
    description: Basic runtime logger for AP1 core events
```

## Endpoints API

- `GET /api/plugins` - lista i plugin disponibili
- `POST /api/plugins/toggle` - abilita o disabilita un plugin

Richiesta JSON:

```json
{
  "name": "default-logger",
  "enabled": true
}
```

## Uso

Nell’API il loader dei plugin può leggere `config/plugins.yaml` e decidere quali plugin caricare.
Nel core, il plugin loader può estendere la pipeline di rete con plugin Rust o WASM.
