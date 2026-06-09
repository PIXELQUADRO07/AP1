# Fake Connection Mode / Evil Twin Behavior

AP1 supports a fake connection workflow with captive portal and evil twin capabilities.

## Standard Access Point

A normal access point behaves like a legitimate Internet gateway:

- Creates a Wi-Fi network with an SSID and optional password
- Assigns local IP addresses via DHCP
- Routes traffic to a real WAN interface using NAT
- Uses upstream DNS servers like 8.8.8.8 or the ISP's DNS
- Devices connect and can access real Internet resources

## Fake Connection

AP1 can also operate as a deceptive captive portal / evil twin:

1. **Clone a target network**
   - AP1 can scan nearby SSIDs and start a rogue AP with the same name.
   - This is a captive evil twin mode.

2. **Optional deauthentication attack**
   - AP1 supports sending deauth frames to force clients off a legitimate AP.
   - This encourages victims to reconnect to the fake AP instead of the real network.

3. **Isolate traffic and spoof DNS**
   - The fake AP does not forward traffic to the real Internet by default.
   - AP1 can run `hostapd` + `dnsmasq` with DNS spoofing rules.
   - `dnsmasq` can redirect known captive-check domains and all other queries to the local portal IP.

4. **Serve a phishing/captive portal page**
   - AP1 runs a local captive portal web server.
   - Users who browse the network are redirected to a login page or fake landing page.
   - Submitted credentials are captured and logged.

5. **Optional HTTPS interception**
   - True SSL stripping is difficult on modern clients, but AP1 can still redirect HTTPS traffic to the portal.
   - This creates a captive portal experience for many mobile devices.

## How AP1 differs from a normal AP

| Feature | Normal AP | AP1 Fake Connection |
|---|---|---|
| Internet access | Yes | No by default |
| DNS provider | Upstream DNS | Local spoofing |
| NAT to WAN | Yes | Only when captive portal disabled |
| Deauth attack | No | Optional via `deauth` |
| SSID cloning | No | Yes via `eviltwin` |
| Captive portal | Optional | Yes |

## AP1 capabilities for fake AP mode

- `core/src/recon.rs` scans for nearby SSIDs and produces cloned AP profiles.
- `core/src/orchestrator.rs` starts `hostapd` and `dnsmasq` for captive portal mode.
- `core/src/system_control/mod.rs` applies NAT/iptables rules to route port 80/443/53 traffic to the portal.
- `core/src/captive_portal.rs` hosts a local portal page and captures submitted credentials.
- API/CLI commands such as `/api/eviltwin/start` and `eviltwin <iface> <ssid>` automate the process.

## Recommended workflow

1. Use `eviltwin` to clone a target SSID.
2. Use `deauth` to disconnect clients from the real network.
3. Enable captive portal mode in AP1.
4. Connect a victim device and capture credentials or phishing traffic.

## Notes

- AP1 is designed as a research/proof-of-concept tool.
- The fake connection mode is intentionally isolated from real Internet traffic.
- This is a different threat model than a normal managed AP that provides genuine connectivity.
