<p align="center">
  <img src="https://gagos.dev/images/gagos_small.png" alt="GAGOS Logo" width="200"/>
</p>

<p align="center"><strong>Lightweight DevOps Platform</strong></p>

A comprehensive DevOps administration platform with network diagnostics, Kubernetes management, CI/CD pipelines, database tools, and more - all in a single container.

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org)

> Copyright 2024-2026 GAGOS Project. Licensed under the Apache License 2.0.

## Features

### Network Tools
- **Ping** - ICMP ping with packet statistics
- **DNS Lookup** - Resolve A, AAAA, CNAME, MX, NS, TXT records
- **Port Check** - TCP connectivity testing
- **Traceroute** - Network path tracing
- **Telnet** - TCP connection testing
- **Whois** - Domain/IP registration lookup
- **SSL Check** - Certificate inspection and validation
- **Curl** - HTTP requests with headers and response info
- **Network Interfaces** - View local network configuration

### Kubernetes Management
- **Full Resource Support** - Namespaces, Nodes, Pods, Services, Deployments, DaemonSets, StatefulSets, Jobs, CronJobs, ConfigMaps, Secrets, Ingresses, PVCs, Events
- **Resource Operations** - Create, Edit, Delete, Describe, Scale, Restart
- **Pod Operations** - View logs, exec into containers
- **Auto-refresh** - Real-time resource monitoring
- **YAML Editor** - Edit resources directly

### CI/CD Pipelines
- **Kubernetes Pipelines** - YAML-defined pipelines running as K8s Jobs
- **Freestyle Jobs** - SSH-based jobs for server deployments
- **SSH Host Management** - Securely store and test SSH connections
- **Webhooks** - Trigger builds from external systems
- **Artifacts** - Collect and download build outputs
- **Notifications** - Webhook notifications on build events

### Database Tools
- **PostgreSQL** - Connect, query, view schema, export dumps
- **MySQL** - Connect, query, view schema, export dumps
- **Redis** - Connect, browse keys, execute commands, cluster info
- **Elasticsearch** - Cluster health, index management, document search, query console

### S3 Storage
- **Multi-provider** - AWS S3, MinIO, and any S3-compatible storage
- **Bucket Management** - Create, delete, list buckets
- **Object Browser** - Upload, download, delete files with folder navigation
- **Presigned URLs** - Generate shareable links

### Monitoring
- **Cluster Overview** - Node and pod resource usage
- **Resource Quotas** - Namespace quota monitoring
- **HPA Status** - Horizontal Pod Autoscaler metrics

### Developer Tools
- **Base64** - Encode/decode strings
- **Hashing** - MD5, SHA1, SHA256, SHA512
- **K8s Secret Decoder** - Decode base64 secrets
- **Certificate Tools** - Parse and validate certificates
- **SSH Key Generator** - Generate RSA/ECDSA/ED25519 keys
- **JSON Formatter** - Format and minify JSON
- **Diff Tool** - Compare text side by side

### Additional Features
- **Web Terminal** - Full PTY terminal in the browser
- **Notepad** - Multi-tab text editor with persistence
- **Session Authentication** - Password-based auth with secure cookies
- **Desktop UI** - Multi-window desktop interface

## Quick Start

### Docker

```bash
export GAGOS_PASSWORD=$(openssl rand -base64 12)
echo "Password: $GAGOS_PASSWORD"

docker run -d --name gagos -p 8080:8080 \
  -e GAGOS_PASSWORD="$GAGOS_PASSWORD" \
  --cap-add=NET_RAW \
  netstudioge/gagos:latest
```

### Kubernetes

```bash
# Create namespace and generate random password
kubectl create namespace gagos
kubectl create secret generic gagos-auth -n gagos \
  --from-literal=password=$(openssl rand -base64 12)

# Deploy
kubectl apply -f https://raw.githubusercontent.com/NetStudioTech/Gagos/main/deploy/kubernetes/gagos-all-in-one.yaml

# Get your password
kubectl get secret gagos-auth -n gagos -o jsonpath='{.data.password}' | base64 -d && echo

# Access (open http://localhost:9197)
kubectl port-forward -n gagos svc/gagos 9197:8080
```

### Docker Compose

```bash
git clone https://github.com/NetStudioTech/Gagos.git
cd gagos
export GAGOS_PASSWORD=$(openssl rand -base64 12)
docker-compose up -d
```

## Documentation

- [Architecture](docs/ARCHITECTURE.md) - System design and components
- [CI/CD User Guide](docs/CICD_USER_GUIDE.md) - Pipeline and freestyle job setup
- [Installation Guide](docs/installation.md) - Detailed installation options
- [API Reference](docs/API.md) - REST API documentation

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| GAGOS_HOST | 0.0.0.0 | Listen address |
| GAGOS_PORT | 8080 | Listen port |
| GAGOS_PASSWORD | (required) | Authentication password |
| GAGOS_RUNTIME | docker | Runtime (docker/kubernetes) |
| GAGOS_LOG_LEVEL | info | Log level |

## Project Structure

```
gagos/
├── cmd/gagos/           # Application entry point
├── internal/
│   ├── auth/            # Authentication
│   ├── cicd/            # CI/CD pipeline engine
│   ├── database/        # Database clients (PostgreSQL, MySQL, Redis, ES, S3)
│   ├── devtools/        # Developer tools
│   ├── k8s/             # Kubernetes client
│   ├── network/         # Network diagnostic tools
│   └── terminal/        # Web terminal (PTY)
├── web/static/          # Web UI (HTML, CSS, JS)
├── deploy/
│   ├── docker/          # Dockerfile
│   └── kubernetes/      # K8s manifests
├── docs/                # Documentation
├── LICENSE              # Apache 2.0 License
└── README.md
```

## License

This project is licensed under the [Apache License 2.0](LICENSE) - a permissive, business-friendly open source license.

**You are free to:**
- Use commercially
- Modify and distribute
- Use privately
- Use patents

## Pricing

### Community Edition (Free)
Full-featured, self-hosted, forever free:
- All network diagnostic tools
- Kubernetes management
- CI/CD pipelines & freestyle jobs
- Database tools (PostgreSQL, MySQL, Redis, Elasticsearch)
- S3 storage management
- Developer tools
- Web terminal
- Community support via GitHub

### Pro Edition (Coming Soon)
For teams and businesses:
- Everything in Community
- Priority support
- SSO/LDAP integration
- Audit logging
- Multi-cluster support
- Custom branding

### Enterprise (Coming Soon)
For large organizations:
- Everything in Pro
- Dedicated support
- SLA guarantees
- On-premise deployment assistance
- Custom feature development

## Support

- **Community:** [GitHub Issues](https://github.com/NetStudioTech/Gagos/issues)
- **Discussions:** [GitHub Discussions](https://github.com/NetStudioTech/Gagos/discussions)
- **Sponsor:** [GitHub Sponsors](https://github.com/sponsors/gaga951)

## Contributing

Contributions are welcome! Please read our contributing guidelines before submitting PRs.

## Acknowledgments

Built with:
- [Go](https://golang.org/) - Backend
- [Fiber](https://gofiber.io/) - Web framework
- [xterm.js](https://xtermjs.org/) - Terminal emulator
- [client-go](https://github.com/kubernetes/client-go) - Kubernetes client
