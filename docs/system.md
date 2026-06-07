# System Wrappers

AP1 includes wrappers for interacting with operating system and network tools.

## Main folders

- `system/network/` - interface management and IP configuration
- `system/firewall/` - firewall rule generation
- `system/process_manager/` - external process execution and monitoring
- `system/services/` - wrappers for hostapd, dnsmasq, and other network services

## Goals

- centralize interaction with system tools like `systemctl` or `service`
- generate runtime files for `hostapd` and `dnsmasq`
- keep system integration separate from API and core logic

## Integration

The API server can use these wrappers to:
- create and apply configuration files
- start/stop network services
- query system state
