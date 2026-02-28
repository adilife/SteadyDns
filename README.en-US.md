# SteadyDNS

[![Version](https://img.shields.io/badge/version-0.9.0--beta.1-blue.svg)](https://github.com/adilife/SteadyDns/releases/tag/v0.9.0-beta.1)
[![License](https://img.shields.io/badge/license-AGPLv3-green.svg)](LICENSE)
[![Language](https://img.shields.io/badge/Language-Golang%20%7C%20JavaScript-blue)](https://github.com/adilife/SteadyDns)

A lightweight DNS solution tailored for small to medium-sized environments. Built with Golang, it delivers extreme concurrent processing capabilities and minimal resource consumption while balancing ease of use and stability.

## 🚀 Beta Testing (v0.9.0-beta.1 [Release Note](https://github.com/adilife/SteadyDns/releases/tag/v0.9.0-beta.1))
> The beta version is now available! Download, test, and provide your feedback.

> [Changelog](https://github.com/adilife/SteadyDns/blob/main/CHANGELOG.md)

### Core Highlights of This Version
- **Ultra-Simple Deployment**: Single binary file with no additional dependencies, supporting Linux x86_64/arm64 architectures
- **High-Performance Resolution**: Based on Go's native concurrency model, single node supports 10,000+ DNS requests per second
- **Visual Management**: React-based Web panel for one-click DNS rule configuration and real-time status monitoring
- **High Availability Design**: Intelligent upstream DNS switching, local caching, and automatic blocking of unavailable upstreams to ensure resolution stability

### Environment Requirements
- Operating System: Linux (Mainstream distributions like CentOS/Ubuntu/Debian are all supported)
- Architecture Support: x86_64, arm64 (e.g., Raspberry Pi, Kunpeng servers, etc.)
- Port Requirements: TCP/UDP 53 (default DNS service port) and 8080 (Web panel, customizable) need to be open

### Quick Download & Startup (Recommended)
Download the precompiled binary package directly (no compilation required, ready to use out of the box):

#### 1. Download the version for your architecture

> [Linux x86-64 download](https://github.com/adilife/SteadyDNS/releases/download/v0.9.0-beta.1/steadydns-v0.9.0-beta.1-linux-amd64.tar.gz)
```bash
# Linux x86_64 architecture (mainstream x86 servers/virtual machines)
wget https://github.com/adilife/SteadyDNS/releases/download/v0.9.0-beta.1/steadydns-v0.9.0-beta.1-linux-amd64.tar.gz
```
> [Linux arm-64 download](https://github.com/adilife/SteadyDNS/releases/download/v0.9.0-beta.1/steadydns-v0.9.0-beta.1-linux-arm64.tar.gz)
```bash
# Linux arm64 architecture (e.g., Raspberry Pi, Kunpeng, AWS Graviton, etc.)
wget https://github.com/adilife/SteadyDNS/releases/download/v0.9.0-beta.1/steadydns-v0.9.0-beta.1-linux-arm64.tar.gz
```

#### 2. Unzip and Start (Basic Testing)

```bash
# Unzip the downloaded package
tar -zxvf steadydns-v0.9.0-beta.1-linux-*.tar.gz

# Enter the unzipped directory
cd steadydns-v0.9.0-beta.1-linux-*

# Grant execution permission
chmod +x steadydnsd

# Start the service (foreground run, for testing)
./steadydnsd start

# View command line help
./steadydnsd --help
```

#### 3. Access the Web Panel
After startup, access http://Server-IP:8080 in your browser to enter the visual management panel (default username: admin, password: admin123).
> ⚠️ Security Note: Change the default password immediately after first login!

>For complete installation & deployment, auto-start on boot, custom configuration, etc., please refer to the [Deployment Guide](https://github.com/adilife/SteadyDns/blob/main/DEPLOYMENT.md)

### Testing Feedback
* Encountered an issue? 👉 [Submit an Issue](https://github.com/adilife/SteadyDNS/issues/new?labels=beta-test&title=%5Bv0.9.0-beta.1%20Test%20Feedback%5D)
* Have a feature suggestion? 👉 [Discuss in the Discussion Area](https://github.com/adilife/SteadyDNS/discussions/categories/beta-test)

## Project Introduction

SteadyDNS consists of two core sub-projects with a front-end and back-end separation architecture:

| Sub-project | Tech Stack | Description |
|--------|--------|------|
| [steadydnsd](./steadydnsd) | Go | DNS server core, responsible for DNS request resolution, intelligent upstream forwarding, configuration persistence, and other core logic |
| [steadydns_ui](./steadydns_ui) | React | Web management panel, providing visual configuration, status monitoring, rule management, etc. |

## Core Features

### Overall Features

- 🚀 **Lightweight** - Deploy via single binary file with no additional dependencies
- ⚡ **High Performance** - Based on Go's native concurrency model, supporting 10,000+ DNS requests per second
- 🎨 **Visual Management** - One-click configuration via Web panel, no need to modify configuration files
- ⚡ **Real-time Effectiveness** - Domain configuration changes take effect immediately without restarting the DNS service
- 🔄 **Intelligent Forwarding** - Support configuration of multiple upstream DNS servers with automatic switching by priority/availability
- ⚡ **Local Caching** - High-performance local caching with automatic expiration based on TTL
- 📊 **Status Monitoring** - Real-time view of DNS request volume, response latency, upstream availability and other metrics
- 🛡️ **Stability Assurance** - Automatically block unavailable upstream DNS to avoid resolution failures

### Backend Features (steadydnsd)

- Support resolution of mainstream DNS record types including A/AAAA/CNAME/MX/NS/TXT/SRV
- Custom local authoritative zones (based on BIND9.18+ service)
- Support TCP/UDP protocols, compatible with IPv4/IPv6
- Automatic backup and recovery of configuration files
- Logging and auditing functions
- RESTful API interface
- JWT authentication

### Frontend (steadydns_ui) Features

- Clean and easy-to-use operation interface
- Upstream DNS server management (add/delete/priority adjustment)
- Integrated BIND service management
- Real-time monitoring panel for DNS service status
- QPS/CPU/Memory/Network trend monitoring
- TOP resolved domains, TOP client ranking

## Project Structure

```
SteadyDNS/
├── README.md                 # Project overview (this file)
├── CHANGELOG.md              # Changelog
├── LICENSE                   # License (AGPLv3)
│
├── steadydnsd/               # Backend project
│   ├── src/                  # Source code
│   ├── docs/                 # Documentation
│   ├── Makefile              # Build script
│   └── README.md             # Backend detailed documentation
│
└── steadydns_ui/             # Frontend project
    ├── src/                  # Source code
    ├── public/               # Static resources
    ├── package.json          # Dependency configuration
    └── README.md             # Frontend detailed documentation
```

## Development Guide

### Backend Development

```bash
cd steadydnsd
make help          # View available commands
make build         # Compile
make test          # Run tests
make run-dev       # Run in development mode
```

See [steadydnsd/README.md](./steadydnsd/README.md) for details

### Frontend Development

```bash
cd steadydns_ui
npm install        # Install dependencies
npm run dev        # Development mode
npm run build      # Build production version
```

See [steadydns_ui/README.md](./steadydns_ui/README.md) for details

## License

This project is licensed under the GNU Affero General Public License v3.0 (AGPLv3).

See the [LICENSE](LICENSE) file for details.

## Contribution

Issues and Pull Requests are welcome.

## Contact

- GitHub: https://github.com/adilife/SteadyDNS
