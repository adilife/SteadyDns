# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.9.0-beta.1] - 2026-02-27

### Added
- DNS forwarding service with UDP/TCP support
- Multi-forward-group configuration support
- BIND server integration and management
- RESTful API interface based on Gin framework
- JWT authentication for API security
- User management module (CRUD operations)
- Forward group management (CRUD operations)
- DNS server management within forward groups
- QPS history tracking and statistics
- Resource usage monitoring (CPU, Memory, Disk)
- Network traffic monitoring
- DNS Cookie support for security enhancement
- TCP connection pool optimization
- Domain Trie for efficient DNS routing
- Plugin system architecture
- Backup and restore functionality for BIND configuration
- Rate limiting for API protection
- Daemon process management (start/stop/restart/status)
- Rotating log system
- SQLite database for data persistence

### Security
- bcrypt password hashing for user credentials
- JWT token-based authentication
- Rate limiting to prevent API abuse
- DNS Cookie support for DNS security

### Changed
- Optimized TCP connection pool with configurable parameters
- Improved DNS forwarding performance with worker pool

### Technical Details
- Go version: 1.25.5
- License: AGPLv3
- Database: SQLite
- Web Framework: Gin

---

## Release Notes

### v0.9.0-beta.1 (First Beta Release)

This is the first beta release of SteadyDNS, intended for testing and evaluation purposes. 

**Target Users:** Enterprise users who need a flexible DNS server solution with BIND integration.

**Key Features:**
- Complete DNS forwarding functionality
- Web-based management API
- BIND server integration
- User and permission management
- System monitoring capabilities

**Known Limitations:**
- This is a beta version, not recommended for production environments
- Some advanced features may need further testing
- Documentation is still being improved

**Feedback:**
Please report any issues or suggestions through the project's issue tracker.
