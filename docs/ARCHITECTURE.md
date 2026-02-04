# GAGOS - Architecture & System Design

## Overview

GAGOS is a lightweight DevOps platform and management tool for Kubernetes clusters, Docker containers, and network diagnostics with a web-based terminal.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Browser                              │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐            │
│  │ Dashboard  │  │ K8s Manager│  │  Terminal  │            │
│  └────────────┘  └────────────┘  └────────────┘            │
└──────────────────────┬──────────────────────────────────────┘
                       │ HTTPS/WSS
┌──────────────────────▼──────────────────────────────────────┐
│                    GAGOS Container                           │
│  ┌────────────────────────────────────────────────────┐     │
│  │              Fiber Web Server                       │     │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐         │     │
│  │  │ REST API │  │WebSocket │  │ Static   │         │     │
│  │  │ Handlers │  │ Handlers │  │ Files    │         │     │
│  │  └────┬─────┘  └────┬─────┘  └──────────┘         │     │
│  └───────┼─────────────┼────────────────────────────────┘     │
│          │             │                                      │
│  ┌───────▼─────────────▼──────────────────────────────┐     │
│  │            Business Logic Layer                     │     │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌───────┐│     │
│  │  │K8s Client│ │  Docker  │ │ Network  │ │ PTY   ││     │
│  │  │ Manager  │ │ Manager  │ │  Tools   │ │Manager││     │
│  │  └────┬─────┘ └────┬─────┘ └──────────┘ └───────┘│     │
│  └───────┼────────────┼──────────────────────────────┘     │
│          │            │                                      │
└──────────┼────────────┼──────────────────────────────────────┘
           │            │
┌──────────▼────────┐   │   ┌──────────────────┐
│  Kubernetes API   │   │   │   Docker Socket  │
│    (External)     │   └───▶   (Optional)     │
└───────────────────┘       └──────────────────┘
```

## Component Architecture

```
GAGOS Application
│
├── Presentation Layer
│   ├── HTML Templates (HTMX)
│   ├── Static Assets (CSS, JS)
│   └── WebSocket Connections
│
├── API Layer
│   ├── REST Endpoints
│   │   ├── /api/k8s/*
│   │   ├── /api/docker/*
│   │   ├── /api/network/*
│   │   └── /api/health
│   │
│   ├── WebSocket Endpoints
│   │   ├── /api/terminal/ws
│   │   └── /api/k8s/logs
│   │
│   └── Middleware
│       ├── CORS
│       ├── Logger
│       ├── Auth (optional)
│       └── Rate Limiter
│
├── Business Logic Layer
│   ├── Kubernetes Manager
│   │   ├── Multi-cluster support
│   │   ├── Context switching
│   │   ├── Resource operations
│   │   └── Log streaming
│   │
│   ├── Docker Manager
│   │   ├── Container operations
│   │   └── Image management
│   │
│   ├── Network Tools
│   │   ├── Ping
│   │   ├── DNS Lookup
│   │   ├── Port Scanner
│   │   └── HTTP Tester
│   │
│   └── Terminal Manager
│       ├── PTY creation
│       ├── Session management
│       └── I/O handling
│
└── Data Layer
    ├── Configuration (YAML/ENV)
    ├── SQLite (sessions, history)
    └── In-memory cache
```

## Technology Stack

| Component | Technology | Reason |
|-----------|------------|--------|
| Language | Go 1.21+ | K8s native, performance, static binary |
| Web Framework | Fiber v2 | Fast, Express-like, WebSocket support |
| Frontend | HTMX | Simple, progressive enhancement |
| Styling | Tailwind CSS | Rapid development, consistent design |
| State | Alpine.js | Lightweight, reactive |
| Terminal | xterm.js | Full-featured, widely used |
| Config | Viper | Flexible, ENV support |
| Logging | zerolog | Fast, structured |
| K8s Client | client-go | Official, comprehensive |
| Docker Client | docker/client | Official SDK |
| Container Base | Ubuntu 22.04 Slim | Familiar, good package support |

## Module Breakdown

### Module 1: Core Infrastructure
- Config management
- Logging setup
- Error handling
- Lifecycle management

### Module 2: Web Server
- Fiber app initialization
- Routing
- Middleware stack
- Static file serving

### Module 3: Kubernetes Integration
- Client manager (multi-cluster)
- Resource controllers
- Log streaming
- Metrics collection

### Module 4: Docker Integration
- Docker client wrapper
- Container operations
- Image management

### Module 5: Network Tools
- ICMP ping
- DNS resolver
- TCP port scanner
- HTTP client with metrics

### Module 6: Terminal Emulation
- PTY manager
- Session lifecycle
- WebSocket bridge

## Data Flow

### Kubernetes Operations
```
User → Frontend → API Handler → K8s Manager → K8s API
                                      │
                                      ├→ Authentication
                                      ├→ Context Selection
                                      ├→ API Request
                                      └→ Response Transform
```

### Terminal Session
```
User Input (Browser)
    │
    ▼
WebSocket Handler
    │
    ├→ Session Manager (get/create session)
    │       │
    │       ▼
    │   PTY Process (/bin/bash)
    │       │
    │       ▼
    ├← Output Stream
    │
    ▼
Browser (xterm.js)
```

## Success Metrics

### Functional
- Connects to multiple K8s clusters
- Manages pods and deployments
- Streams logs in real-time
- Web terminal works
- All network tools functional

### Non-Functional
- Container image <200MB
- Startup time <5 seconds
- Memory usage <512MB under load
- API response time <100ms (p95)
- Supports 100+ concurrent users
