AP1 CLI

Usage examples:

Build:

```bash
cd cli
go build -o ap1-cli
```

Run:

```bash
./ap1-cli status
./ap1-cli config
./ap1-cli profiles list
./ap1-cli profiles select default
./ap1-cli plugins list
./ap1-cli plugins toggle captive-portal on
./ap1-cli system hostapd restart
./ap1-cli firewall apply wlan0 192.168.50.1
./ap1-cli firewall clear wlan0
./ap1-cli interface configure wlan0 192.168.50.1 24
```
