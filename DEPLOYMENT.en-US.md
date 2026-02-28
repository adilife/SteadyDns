# SteadyDNS Deployment Guide

This document provides a detailed deployment guide for SteadyDNS, including environment requirements, installation steps, configuration instructions, and frequently asked questions.

## Table of Contents

- [Environment Requirements](#environment-requirements)
  - [Operating System](#operating-system)
  - [Third-Party Components](#third-party-components)
  - [BIND Plugin Notes](#bind-plugin-notes)
  - [Port Requirements](#port-requirements)
  - [Hardware Requirements](#hardware-requirements)
- [Installation Methods](#installation-methods)
  - [Method 1: Precompiled Binary Package (Recommended)](#method-1-precompiled-binary-package-recommended)
  - [Method 2: Source Code Compilation (Development/Customization)](#method-2-source-code-compilation-developmentcustomization)
  - [Method 3: Quick Installation via Makefile](#method-3-quick-installation-via-makefile)
  - [Systemd Service Registration (Recommended for Production)](#systemd-service-registration-recommended-for-production)
- [Configuration Instructions](#configuration-instructions)
  - [Directory Structure](#directory-structure)
  - [Configuration File Management](#configuration-file-management)
  - [Environment Variables](#environment-variables)
- [Startup & Management](#startup--management)
  - [Command-Line Management](#command-line-management)
  - [Systemd Service (Recommended)](#systemd-service-recommended)
  - [Web Admin Panel Access](#web-admin-panel-access)
- [Upgrade Guide](#upgrade-guide)
  - [Upgrade Steps](#upgrade-steps)
  - [Database Migration](#database-migration)
- [Frequently Asked Questions](#frequently-asked-questions)

---

## Environment Requirements

### Operating System

| System | Version | Architecture |
|--------|---------|--------------|
| Linux  | CentOS 7+, Ubuntu 18.04+, Debian 10+ | x86_64, arm64 |

### Third-Party Components

| Component | Version Requirement | Description |
|-----------|---------------------|-------------|
| BIND      | 9.18+               | DNS authoritative server (required when BIND plugin is enabled) |
| SQLite    | 3.x                 | Database (embedded, no separate installation required) |

### BIND Plugin Notes

SteadyDNS supports running in two modes, depending on whether the BIND plugin is enabled:

| Mode | BIND Plugin | Feature Description |
|------|-------------|---------------------|
| **Authoritative Server Mode** | Enabled | Supports authoritative domain management, can act as primary/secondary DNS server |
| **Forwarder Mode** | Disabled | Only provides DNS forwarding functionality, no authoritative domain management |

#### Enabling the BIND Plugin

To enable authoritative domain management functionality, ensure:

> ⚠️ **Important Note**: SteadyDNS requires port 53 to provide DNS services. The locally deployed BIND must modify the default listening port (port 5300 is recommended).

**Step 1: Enable the BIND plugin in the configuration file**

Edit the `config/steadydns.conf` file and set the following in the `[Plugins]` section:

```ini
[Plugins]
# BIND Plugin - Authoritative Domain Management
# Enable the BIND plugin to support authoritative domain management
# Changes take effect after service restart
BIND_ENABLED=true
```

**Step 2: Install BIND 9.18+**
```bash
# CentOS/RHEL
yum install bind bind-utils

# Ubuntu/Debian
apt install bind9 bind9utils
```

**Step 3: Verify BIND version**
```bash
named -v # Expected output: BIND 9.18.x
```

**Step 4: Configure BIND-related parameters**

Configure BIND-related options in `config/steadydns.conf`:
```ini
[BIND]
# BIND service address (Note: not the default port 53)
BIND_ADDRESS=127.0.0.1:5300
# RNDC key file path
RNDC_KEY=/etc/named/rndc.key
# Zone file storage path
ZONE_FILE_PATH=/usr/local/bind9/var/named
# named configuration file path
NAMED_CONF_PATH=/etc/named
# RNDC port
RNDC_PORT=9530
```

**Step 5: Modify BIND listening port**

Edit the BIND configuration file `/etc/named.conf` and modify the listening port:
```bash
# Modify the options section
options {
    listen-on port 5300 { 127.0.0.1; };  # Listen only on localhost, port changed to 5300
    allow-query { 127.0.0.1; };          # Allow only local access
    // Other configurations...
};
```

**Step 6: Restart the BIND service:**
```bash
# Restart BIND service
systemctl restart named
# Verify listening port
ss -tulpn | grep named  # Should display 127.0.0.1:5300
```

**Step 7: Configure RNDC**
```bash
# Generate RNDC key (if not configured yet)
rndc-confgen -a
# Ensure SteadyDNS has permission to access the BIND configuration directory
chmod -R 755 /usr/local/bind9/var/named
```

**Step 8: Restart the SteadyDNS service**
```bash
./steadydns restart
# Or use systemd
systemctl restart steadydns
```

#### Disabling the BIND Plugin (Forwarder Only Mode)

Only need to modify the configuration file to disable the plugin, no need to install BIND:

```ini
[Plugins]
# Disable authoritative domain management, keep only forwarding functionality
BIND_ENABLED=false
```

> **✨ Features supported after disabling**: DNS recursive forwarding, upstream DNS configuration, domain filtering / blacklist/whitelist; **Not supported**: Authoritative domain management.
> 
> ⚠️ **Note**: Restart the SteadyDNS service for plugin configuration changes to take effect.

### Port Requirements

| Port | Protocol | Description | Security Recommendation |
|------|----------|-------------|-------------------------|
| 53   | UDP/TCP  | DNS service port | Open only to business network segments |
| 8080 | TCP      | Web admin panel (customizable) | Open only to operation and maintenance network segments (customizable) |

### Hardware Requirements

| Configuration Item | Minimum Requirement | Recommended Configuration |
|--------------------|---------------------|---------------------------|
| CPU                | 1 core              | 2 cores+                  |
| Memory             | 512 MB              | 1 GB+                     |
| Disk               | 100 MB              | 1 GB+                     |

---

## Installation Methods

### Method 1: Precompiled Binary Package (Recommended)
Suitable for production environments, no compilation dependencies required, quick deployment:

```bash
# 1. Download the latest version (replace with actual version number)
VERSION="v0.9.0-beta.1"
wget https://github.com/adilife/SteadyDNS/releases/download/${VERSION}/steadydns-linux-amd64.tar.gz -O steadydns.tar.gz

# 2. Extract to specified directory (standardized path)
mkdir -p /opt/steadydns
tar -xzf steadydns.tar.gz -C /opt/steadydns --strip-components=1

# 3. Start the service
/opt/steadydns/steadydns start
```

### Method 2: Source Code Compilation (Development/Customization)
Suitable for scenarios requiring source code modification and custom building:

```bash
# 1. Install compilation dependencies
# CentOS/RHEL
yum install -y git golang npm make gcc
# Ubuntu/Debian
apt install -y git golang npm make gcc

# 2. Clone the repository
git clone https://github.com/adilife/SteadyDNS.git
cd SteadyDNS

# 3. Build the frontend (if customizing the frontend)
cd steadydns_ui
npm install && npm run build
cd ../

# 4. Compile the backend (full build including frontend)
cd steadydnsd
make build-full

# 5. Deploy to standardized directory
mkdir -p /opt/steadydns
cp src/cmd/steadydns /opt/steadydns/
chmod +x /opt/steadydns/steadydns
```

### Method 3: Quick Installation via Makefile
Suitable for users familiar with Makefile who need to simplify the compilation process:

```bash
cd SteadyDNS/steadydnsd
make install  # Automatically install compilation dependencies
make build    # Compile binary file
cp src/cmd/steadydns /opt/steadydns/  # Deploy to target directory
```

### Systemd Service Registration (Recommended for Production)
Manage the service via Systemd, supporting auto-start on boot and process monitoring:

```bash
# Copy binary file
sudo cp steadydns /opt/steadydns/

# Install service
sudo cp scripts/steadydns.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable steadydns
sudo systemctl start steadydns
```
---

## Configuration Instructions

### Directory Structure

```
/etc/steadydns/
├── steadydns              # Executable file
├── config/
│   └── steadydns.conf     # Main configuration file
├── log/                   # Log directory
├── backup/                # Backup directory
└── steadydns.db           # SQLite database (auto-created)
```

### Configuration File Management

> **Tip**: If config/steadydns.conf does not exist when starting the service, a default configuration is automatically generated.

  - Manual customization: For production environments, it is recommended to create the configuration file first before starting the service (to avoid default configuration risks);
  - Configuration backup: It is recommended to back up config/steadydns.conf before manually modifying the configuration;
  - Admin interface: Configuration file content can be managed through the web admin panel.

#### Main Configuration Item Description

| Configuration Section | Main Configuration Items | Description |
|-----------------------|--------------------------|-------------|
| `[Database]`          | `DB_PATH`                | Database file path (recommended to keep default) |
| `[APIServer]`         | `API_SERVER_PORT`        | API service port (default 8080) |
| `[JWT]`               | `JWT_SECRET_KEY`         | JWT secret key (**must be modified in production**) |
| `[BIND]`              | `BIND_ADDRESS`           | BIND server address |
| `[DNS]`               | `DNS_CLIENT_WORKERS`     | DNS client worker pool size |
| `[Security]`          | `DNS_RATE_LIMIT_PER_IP`  | Single IP request limit |

#### Important Production Environment Configurations

```ini
[JWT]
# Must be modified to a strong secret key in production
JWT_SECRET_KEY=your-strong-secret-key-change-this

[APIServer]
# Recommended to set to release for production
GIN_MODE=release

[Security]
# Adjust rate limit parameters according to actual needs
DNS_RATE_LIMIT_PER_IP=300
DNS_RATE_LIMIT_GLOBAL=10000
```

Refer to the comments in the auto-generated configuration file for complete configuration item descriptions.

### Environment Variables

| Variable Name | Description | Default Value |
|---------------|-------------|---------------|
| `STEADYDNS_DEV_MODE` | Development mode (read frontend from filesystem) | `false` |

---

## Startup & Management

### Command-Line Management
Suitable for test environments and temporary operations:

```bash
# Enter program directory
cd /opt/steadydns

# Start service
./steadydns start

# Stop service
./steadydns stop

# Restart service
./steadydns restart

# Check status
./steadydns status

# Check version (verify deployment version)
./steadydns --version

# Show help
./steadydns --help
```

### Systemd Service (Recommended)
Suitable for production environments, supporting process monitoring and centralized log management:

```bash
# Reload systemd configuration
systemctl daemon-reload

# Start service
systemctl start steadydns

# Stop service
systemctl stop steadydns

# Restart service
systemctl restart steadydns

# Check status
systemctl status steadydns

# Set auto-start on boot
systemctl enable steadydns

# Disable auto-start on boot
systemctl disable steadydns

# View logs (real-time tracking)
journalctl -u steadydns -f --no-pager

# View historical logs (last 1 hour)
journalctl -u steadydns --since "1 hour ago"
```

### Web Admin Panel Access

1. Confirm the service is running normally (systemctl status steadydns);
2. Access via browser: http://<Server IP>:<API_SERVER_PORT> (e.g., http://192.168.1.100:8080)
3. First login:
   - Default username: `admin`
   - Default password: `admin123`
     > ⚠️ **Security Note**: Change the default password immediately after first login!

---

## Upgrade Guide

### Upgrade Steps

```bash
# 1. Backup core data (critical step to avoid data loss)
mkdir -p /opt/steadydns/backup/$(date +%Y%m%d)
cp /opt/steadydns/steadydns.db /opt/steadydns/backup/$(date +%Y%m%d)/
cp /opt/steadydns/config/steadydns.conf /opt/steadydns/backup/$(date +%Y%m%d)/

# 2. Stop the service (recommend grayscale/pausing traffic first in production)
systemctl stop steadydns

# 3. Download and replace binary file (replace with new version)
VERSION="v0.9.0-beta.2"
wget https://github.com/adilife/SteadyDNS/releases/download/${VERSION}/steadydns-linux-amd64.tar.gz -O new-steadydns.tar.gz
tar -xzf new-steadydns.tar.gz -C /tmp/
cp /tmp/steadydns-*/steadydns /opt/steadydns/

# 4. Start service and verify
systemctl start steadydns
sleep 5  # Wait for service startup
systemctl status steadydns -l

# 5. Verify version and functionality
/opt/steadydns/steadydns --version
# Verify DNS functionality (example: resolve local domain)
dig @127.0.0.1 www.baidu.com -p 53
```

### Database Migration

- SteadyDNS automatically detects the database version during upgrade and completes table structure migration;

- If migration fails:
  1. Check logs: journalctl -u steadydns --since "5 minutes ago";
  2. Restore backup database: cp /opt/steadydns/backup/$(date +%Y%m%d)/steadydns.db /opt/steadydns/;
  3. Roll back the binary file to the old version and contact technical support.

---

## Frequently Asked Questions

### 1. Port Occupied (Startup Failure)

**Issue**: Log shows error `bind: address already in use`, service fails to start.

**Solution**:
```bash
# Check port occupancy
netstat -tlnp | grep :53
netstat -tlnp | grep :8080

# Stop the process occupying the port, or modify the port in the configuration file
```

### 2. Insufficient Permissions (Operation Error)

**Issue**: Error `permission denied` when starting the service.

**Solution**:
```bash
# Run with root user, or grant appropriate permissions
sudo ./steadydns start

# Or use systemd service management
systemctl start steadydns
```

### 3. BIND Configuration Error (Authoritative Mode)

**Issue**: BIND-related functions are unavailable.

**Solution**:
```bash
# 1. Check if BIND is installed
named -v

# 2. Verify BIND configuration
named-checkconf /etc/named.conf

# 3. Verify BIND listening port
ss -tulpn | grep 5300

# 4. Verify RNDC connection
rndc -p 9530 status

# 5. Ensure SteadyDNS can access the RNDC key
chmod 644 /etc/named/rndc.key
```

### 4. Database Locked (Operation Timeout)

**Issue**: Error `database is locked` during operations, DNS queries / admin operations time out.

**Solution**:
```bash
# 1. Stop the service
systemctl stop steadydns

# 2. Find processes occupying the database
lsof /opt/steadydns/steadydns.db

# 3. Kill abnormal processes (replace with actual PID)
kill -9 <PID>

# 4. Restart the service
systemctl start steadydns
```

### 5. Forgot Admin Password

**Solution**:
```bash
# 1. Install sqlite3 (if not installed)
yum install -y sqlite3  # CentOS
apt install -y sqlite3  # Ubuntu

# 2. Generate bcrypt hash password (example: new password is newAdmin@123)
# Can be generated via online tools/code, or use the following command (python required)
python3 -c "import bcrypt; print(bcrypt.hashpw('newAdmin@123'.encode(), bcrypt.gensalt(rounds=12)).decode())"

# 3. Reset password (replace with generated hash value)
sqlite3 /opt/steadydns/steadydns.db "UPDATE users SET password='$2b$12$xxxxxx' WHERE username='admin';"

# 4. Restart service (optional)
systemctl restart steadydns
```

---

## Technical Support

- **GitHub Issues**: https://github.com/adilife/SteadyDNS/issues
- **Documentation**: https://github.com/adilife/SteadyDNS/tree/main/docs

---

## Related Documents

- [README.md](./README.md) - Project Overview
- [CHANGELOG.md](./CHANGELOG.md) - Changelog
- [steadydnsd/README.md](./steadydnsd/README.md) - Backend Detailed Documentation
- [steadydns_ui/README.md](./steadydns_ui/README.md) - Frontend Detailed Documentation
