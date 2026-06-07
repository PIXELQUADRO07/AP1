# System Wrappers

AP1 include wrapper per l’integrazione con strumenti di sistema e rete.

## Cartelle principali

- `system/network/` - gestione interfacce e configurazione IP
- `system/firewall/` - generazione regole firewall
- `system/process_manager/` - esecuzione e monitoraggio processi esterni
- `system/services/` - wrapper per hostapd, dnsmasq e altri servizi di rete

## Obiettivi

- centralizzare l’interazione con strumenti di sistema come `systemctl` o `service`
- generare file runtime per `hostapd` e `dnsmasq`
- mantenere un layer separato dalla logica API e core

## Integrazione

L’API server può usare questi wrapper per:
- creare e applicare file di configurazione
- avviare/fermare servizi di rete
- estrarre informazioni dallo stato di sistema
