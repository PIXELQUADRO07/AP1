🐙 WiFiPumpkin3 (WP3) — Descrizione tecnica completa

WiFiPumpkin3 è un framework open-source per security auditing delle reti Wi-Fi, progettato per simulare scenari di attacco controllati come rogue access point, evil twin e captive portal phishing, con funzionalità modulari per analisi del traffico e testing di dispositivi client.

È scritto principalmente in Python e si appoggia a strumenti di sistema Linux per la gestione delle interfacce wireless e del traffico di rete.

🧠 Architettura generale

WiFiPumpkin3 è organizzato come un sistema modulare composto da:

1. Core Engine

Gestisce:

inizializzazione delle interfacce Wi-Fi
creazione e gestione degli Access Point
orchestrazione dei moduli attivi
event handling (client connect/disconnect)

👉 È il “motore centrale” del framework.

2. Wireless Layer (AP Manager)

Responsabile della parte radio:

creazione di access point tramite hostapd o equivalente
configurazione SSID, canali, sicurezza (open/WPA2 emulation)
gestione multiple interfacce (AP + monitor mode)

Funzioni chiave:

AP spoofing
beacon flooding (in alcune configurazioni)
clone SSID (evil twin setup)
3. DHCP / DNS / Routing Layer

Gestisce la rete simulata:

DHCP server
assegna IP ai client connessi
DNS spoofing / resolver custom
reindirizza domini verso IP controllati
Routing NAT
instrada traffico verso gateway o proxy
4. Captive Portal System

Uno dei moduli più importanti.

Permette:

intercettazione iniziale della navigazione HTTP/HTTPS
redirezione automatica a pagina di login
gestione template web personalizzati

Funzioni avanzate:

template engine (HTML/CSS/JS)
logging input (solo in contesti autorizzati)
conditional redirect (basato su host richiesto)
5. MITM & Proxy Layer (opzionale)

Modulo per analisi del traffico:

HTTP proxy trasparente
sniffing pacchetti (livello applicativo)
filtraggio richieste
injection controllata (solo lab)

⚠️ In contesti reali moderni HTTPS limita molto questa parte senza tecniche aggiuntive certificate.

6. Recon Module

Modulo di intelligence Wi-Fi:

scansione reti disponibili (SSID/BSSID)
analisi intensità segnale
identificazione canali congestionati
detection dispositivi client
7. Plugin System (chiave del framework)

WiFiPumpkin3 è estensibile tramite plugin:

Esempi tipici:

credential harvester (captive portal input logging)
traffic logger
DNS logger
social engineering pages
automation scripts

👉 Architettura event-driven:

hook su eventi client
hook su richieste DNS/HTTP
hook su connessioni/disconnessioni
8. Web UI / Control Panel

Interfaccia di gestione (se attiva):

gestione AP
monitor client in tempo reale
attivazione/disattivazione moduli
visualizzazione log
configurazione plugin
⚙️ Flusso operativo tipico
Setup interfacce Wi-Fi (monitor + AP)
Avvio rogue AP con SSID scelto
DHCP assegna IP ai client
DNS reindirizza traffico
Captive portal intercetta HTTP request iniziale
Client interagisce con pagina fake (se progettata così)
Logging e analisi eventi
Proxy opzionale intercetta traffico applicativo
🧩 Design pattern utili (per una tua versione migliorata)

Se stai costruendo un’evoluzione, WP3 è tipicamente basato su:

Modular plugin architecture
Event bus interno (observer pattern)
State machine per client session
Layered networking stack abstraction
🚀 Limiti della versione classica (utile per miglioramenti)

WiFiPumpkin3 “base” ha limiti strutturali:

1. Dipendenza forte da tool esterni
hostapd
dnsmasq
iptables

👉 migliorabile con stack interno più autonomo

2. Scarsa gestione HTTPS moderna
quasi tutto il traffico è cifrato
captive portal limitato

👉 possibile miglioramento: detection SNI / metadata analysis (in lab)

3. Scalabilità limitata
gestione multi-client non ottimizzata
logging non sempre strutturato
4. UI datata
web UI basilare
poca telemetria avanzata