# 🔧 AP1 - Rogue AP Orchestrator

A powerful, modular orchestrator for managing captive portals and access points. Built with a high-performance **Rust core**, **Go API server**, and intuitive **CLI interface** for seamless AP management.

---

## ✨ Features

- 🚀 **Modular Architecture** - Rust core, Go API, and CLI client working in harmony
- 🎯 **Captive Portal Runtime** - Advanced packet capture and portal management
- 🔌 **Plugin System** - Rust/WASM plugin support for extensibility
- 🌐 **REST API** - Full-featured API for programmatic control
- 🎛️ **Interactive CLI** - User-friendly command-line interface
- 🔒 **Wi-Fi Management** - hostapd and dnsmasq integration
- 📊 **Credential Capture** - Portal credential tracking and management
- 🔥 **Firewall Control** - Dynamic firewall rules for captive portal
- 🐳 **Docker Support** - Containerization for easy deployment

---

## 📁 Repository Structure

```
AP1/
├── core/                 # Rust engine - captive portal runtime, packet capture, plugin management
├── api/                  # Go REST API server - session management and core coordination
├── cli/                  # Command-line client - profile and service management
├── plugins/              # Plugin system with Rust/WASM support
├── system/               # OS wrappers and network integrations
├── config/               # YAML configurations and portal templates
├── docker/               # Docker setup and deployment scripts
├── docs/                 # Architecture and setup documentation
├── Makefile              # Build automation
├── install.sh            # Dependency installation script
├── start.sh              # Service startup script
└── ap1                   # Interactive orchestrator wrapper
```

---

## 🚀 Quick Start

### Prerequisites

- Rust 1.70+
- Go 1.20+
- Linux (tested on Ubuntu 20.04+)
- Root/sudo access (for network operations)

### Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/PIXELQUADRO07/AP1.git
   cd AP1
   ```

2. **Install dependencies:**
   ```bash
   ./install.sh
   ```

3. **Build all components:**
   ```bash
   make core
   make api
   ```

### Running AP1

**Option 1: Interactive Orchestrator (Recommended)**
```bash
./ap1 start
```

**Option 2: Individual Components**

Start the Rust core:
```bash
cd core
cargo run
# With custom config:
AP1_CONFIG_PATH=../config/global.yaml AP1_PLUGIN_CONFIG_PATH=../config/plugins.yaml cargo run
```

Start the Go API server:
```bash
cd ../api
go run .
# With custom flags:
go run . -config ../config/global.yaml -plugins ../config/plugins.yaml -addr :8080
```

Start the CLI client:
```bash
cd cli
go run . --help
```

---

## 📋 CLI Usage Examples

### Status and Configuration
```bash
./ap1 status          # Check overall system status
./ap1 config          # View global configuration
```

### Profile Management
```bash
./ap1 profiles list                    # List available profiles
./ap1 profiles select <profile-name>   # Activate a profile
```

### Plugin Management
```bash
./ap1 plugins list                          # List available plugins
./ap1 plugins toggle <plugin-name> on       # Enable a plugin
./ap1 plugins toggle <plugin-name> off      # Disable a plugin
./ap1 plugins start <plugin-name>           # Start an external plugin
./ap1 plugins stop <plugin-name>            # Stop a running plugin
```

### System Management
```bash
./ap1 system hostapd restart                              # Restart hostapd
./ap1 system dnsmasq restart                              # Restart dnsmasq
./ap1 firewall apply <interface> <gateway-ip>             # Apply captive portal rules
./ap1 firewall clear <interface>                          # Clear firewall rules
./ap1 interface configure <interface> <ip> <subnet-mask>  # Configure network interface
```

### Reconnaissance
```bash
./ap1 scan <interface>  # Scan for available Wi-Fi networks
```

---

## 🔌 REST API Endpoints

### Core Status
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | API server health check |
| `GET` | `/api/status` | Core status and configuration |
| `GET` | `/status` | Simplified core status |

### Configuration
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/config` | Global configuration (JSON) |

### AP Profiles
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/profiles` | List all AP profiles |
| `POST` | `/api/profiles/select` | Select and apply active AP profile |

### Plugins
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/plugins` | List available plugins |
| `POST` | `/api/plugins/toggle` | Enable/disable a plugin |
| `POST` | `/api/plugins/start` | Start external plugin |
| `POST` | `/api/plugins/stop` | Stop running plugin |

### Network Interfaces
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/interfaces` | List local network interfaces |
| `POST` | `/api/system/interface/configure` | Configure IP and subnet |

### Reconnaissance
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/recon/networks?iface=<iface>` | Perform Wi-Fi scan on interface |

### Captive Portal
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/portal/status` | Captive portal status |
| `GET` | `/api/portal/credentials` | Retrieved portal credentials |

### System Services
| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/system/hostapd/<action>` | Manage hostapd (start/stop/restart) |
| `POST` | `/api/system/dnsmasq/<action>` | Manage dnsmasq (start/stop/restart) |
| `POST` | `/api/system/firewall/apply` | Apply captive portal firewall rules |
| `POST` | `/api/system/firewall/clear` | Clear firewall rules |

---

## 🐳 Docker Deployment

```bash
# Build Docker image
docker build -f docker/Dockerfile -t ap1:latest .

# Run container
docker run -it --privileged -v /etc/ap1:/config ap1:latest
```

---

## 📚 Documentation

- **[Architecture Guide](./docs/ARCHITECTURE.md)** - System design and component interaction
- **[Configuration Guide](./docs/CONFIGURATION.md)** - Setup and customization
- **[Plugin Development](./docs/PLUGINS.md)** - Building custom plugins
- **[API Reference](./docs/API.md)** - Detailed endpoint documentation
- **[Fake Connection Mode](./docs/fake_connection.md)** - Evil twin / captive portal behavior

---

## 🛠️ Development

### Building from Source

```bash
# Build everything
make all

# Build specific components
make core      # Build Rust core
make api       # Build Go API
make cli       # Build CLI

# Clean build artifacts
make clean
```

### Testing

```bash
cd core
cargo test

cd ../api
go test ./...

cd ../cli
go test ./...
```

---

## 📝 Configuration

Edit `config/global.yaml` to customize:
- AP profiles and settings
- Portal templates
- Plugin configurations
- System parameters

Example:
```yaml
profiles:
  - name: default
    interface: wlan0
    gateway: 192.168.50.1
    subnet: 24
    
plugins:
  - name: captive-portal
    enabled: true
```

---

## 🔐 Security Notice

This tool is designed for authorized testing and educational purposes only. Unauthorized access to networks is illegal. Always ensure you have proper authorization before using AP1.

---

## 📄 License

This project is provided as-is for educational and authorized testing purposes.

---

## 🤝 Contributing

Contributions are welcome! Please ensure your code:
- Follows Rust/Go best practices
- Includes comprehensive tests
- Updates documentation accordingly
- Has clear commit messages

---

## 📞 Support

For issues, questions, or suggestions:
- Open an [Issue](https://github.com/PIXELQUADRO07/AP1/issues)
- Check existing [Documentation](./docs/)
- Review [Architecture](./docs/ARCHITECTURE.md)

---

## 🎯 Roadmap

- [ ] Enhanced plugin marketplace
- [ ] Web-based dashboard
- [ ] Advanced analytics and logging
- [ ] Multi-AP orchestration
- [ ] Cloud integration
- [ ] Extended plugin library

---

**AP1** - Powerful AP Management Made Simple 🚀
