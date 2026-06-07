# Setup AP1

## Prerequisiti

- Rust 1.71+
- Go 1.21+
- Docker (opzionale per deploy)

## Avvio del core

```bash
cd core
cargo run
```

Se vuoi passare percorsi di config espliciti:

```bash
AP1_CONFIG_PATH=../config/global.yaml AP1_PLUGIN_CONFIG_PATH=../config/plugins.yaml cargo run
```

## Avvio dell'API

```bash
cd api
go run .
```

Puoi anche usare flag o variabili d'ambiente:

```bash
go run . -config ../config/global.yaml -plugins ../config/plugins.yaml -addr :8080
```

## Avvio della CLI

```bash
cd cli
go run . --help
```

## Setup completo

Esegui il bootstrap da root:

```bash
./install.sh
```

## Avvio rapido con un solo comando

Dalla root del progetto puoi avviare core e API in background e aprire la CLI con:

```bash
./ap1
```

Oppure in modo esplicito:

```bash
./ap1 start
```

## API disponibili

- `GET /health` - stato del server API
- `GET /api/status` - stato del core e configurazione
- `GET /api/config` - configurazione globale in JSON
- `GET /api/profiles` - lista dei profili AP definiti
- `POST /api/profiles/select` - seleziona il profilo AP attivo e applica la configurazione
- `GET /api/plugins` - lista i plugin disponibili
- `POST /api/plugins/toggle` - abilita/disabilita un plugin
- `POST /api/system/hostapd/<action>` - controlla hostapd
- `POST /api/system/dnsmasq/<action>` - controlla dnsmasq

## Configurazione

Modifica `config/global.yaml` per cambiare API, core e parametri di rete.

## Import templates da un repository di template

Se hai una copia locale di un repository di template compatibile con il formato di AP1,
puoi importare le cartelle `config/templates` eseguendo dalla root del progetto:

```bash
make import-templates
# oppure
sh docker/scripts/import_templates.sh /path/to/source/config/templates $(pwd)/config/templates
```

Questo copierà i template nel percorso `config/templates` del progetto AP1.

## Note aggiuntive

Puoi consultare anche `docs/api.md` per la documentazione dei percorsi API.
