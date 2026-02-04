# Changelog

All notable changes to GAGOS are documented in this file.

## [0.10.9] - 2026-01-26

### Added
- **Elasticsearch Management** - New tool similar to Elasticvue
  - Connect to Elasticsearch clusters with optional authentication
  - View cluster health, stats, and node information
  - Index management: create, delete, refresh, view mappings/settings
  - Document browser: search, view, delete documents
  - Query console: execute raw Elasticsearch queries

## [0.10.8] - 2026-01-25

### Added
- **S3 Storage Browser** - Support for S3-compatible object storage
  - AWS S3, MinIO, and other S3-compatible services
  - Bucket management: create, delete, list
  - Object operations: upload, download, delete
  - Folder navigation with breadcrumb
  - Presigned URL generation for sharing
  - File metadata viewing

## [0.10.7] - 2026-01-24

### Added
- **CI/CD Freestyle Jobs** - SSH-based job execution
  - Execute commands on remote servers via SSH
  - Support for shell commands and scripts
  - SCP file transfer (push/pull)
  - Job parameters and environment variables
  - Build history and log streaming

- **SSH Host Management**
  - Add and manage SSH hosts
  - Password and key-based authentication
  - Connection testing
  - Host key verification

### Improved
- CI/CD pipeline reliability
- Webhook handling

## [0.10.6] - 2026-01-23

### Added
- **CI/CD Pipelines** - Kubernetes-based pipeline execution
  - YAML-defined pipelines
  - Job dependencies
  - Artifact collection
  - Webhook triggers
  - Cron scheduling
  - Notification webhooks

## [0.10.5] - 2026-01-22

### Added
- **MySQL Client** - Database management for MySQL
  - Connection with SSL support
  - Query execution with tabular results
  - Schema browser
  - Database dump export

### Improved
- PostgreSQL client performance
- Redis key browser pagination

## [0.10.4] - 2026-01-21

### Added
- **Redis Client** - Redis database browser
  - Connect to Redis (standalone and cluster)
  - Key scanning with pattern matching
  - Value viewing for all data types
  - Command execution
  - Cluster node information

## [0.10.3] - 2026-01-20

### Added
- **PostgreSQL Client** - Database management tool
  - Connect to PostgreSQL databases
  - Execute SQL queries
  - View schema information
  - Export database dumps

## [0.10.2] - 2026-01-19

### Added
- **Developer Tools** - Utility functions
  - Base64 encode/decode
  - Hash generation (MD5, SHA1, SHA256, SHA512)
  - Kubernetes Secret decoder
  - Certificate parser
  - SSH key generator
  - JSON formatter/minifier
  - Text diff comparison

## [0.10.1] - 2026-01-18

### Added
- **Monitoring Dashboard**
  - Cluster resource overview
  - Node CPU/memory metrics
  - Pod resource usage
  - Resource quota monitoring
  - HPA status

## [0.10.0] - 2026-01-17

### Added
- **Notepad** - Multi-tab text editor
  - Create and manage multiple tabs
  - Content persistence
  - Syntax-agnostic editor

### Improved
- Window management system
- Desktop icon organization

## [0.9.0] - 2026-01-15

### Added
- **Extended Kubernetes Resources**
  - DaemonSets, StatefulSets
  - Jobs, CronJobs
  - ConfigMaps, Secrets
  - Ingresses, PVCs
  - Events

- **Resource Operations**
  - Create resources from YAML
  - Edit resources inline
  - Scale deployments
  - Restart workloads

## [0.8.0] - 2026-01-10

### Added
- **Web Terminal** - Browser-based shell
  - Full PTY support
  - xterm.js integration
  - WebSocket communication
  - Terminal resizing

## [0.7.0] - 2026-01-05

### Added
- **Authentication System**
  - Password-based login
  - Session management
  - Secure cookies
  - Runtime-aware password retrieval

## [0.6.0] - 2025-12-28

### Added
- **Kubernetes Management**
  - Namespace listing
  - Node information
  - Pod browser with logs
  - Service listing
  - Deployment management

## [0.5.0] - 2025-12-20

### Added
- **Network Tools**
  - Ping with statistics
  - DNS lookup (multiple record types)
  - Port check
  - Traceroute
  - Telnet
  - Whois
  - SSL certificate check
  - HTTP/Curl requests

## [0.1.0] - 2025-12-01

### Initial Release
- Basic web interface
- Fiber web framework
- Static file serving
- Health check endpoint

---

## Version Numbering

GAGOS uses semantic versioning:
- **MAJOR** - Incompatible API changes
- **MINOR** - New features (backwards compatible)
- **PATCH** - Bug fixes (backwards compatible)
