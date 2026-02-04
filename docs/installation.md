# Installation Guide

This guide covers all installation methods for GAGOS.

## Prerequisites

- Docker 20.10+ or Kubernetes 1.20+
- Network access for container images
- `NET_RAW` capability for ping/traceroute (Docker)

## Docker

### Quick Start

```bash
# Generate a secure password
export GAGOS_PASSWORD=$(openssl rand -base64 12)
echo "Password: $GAGOS_PASSWORD"

# Run GAGOS
docker run -d --name gagos -p 8080:8080 \
  -e GAGOS_PASSWORD="$GAGOS_PASSWORD" \
  -e GAGOS_RUNTIME=docker \
  --cap-add=NET_RAW \
  netstudioge/gagos:latest
```

Access at: http://localhost:8080

### Docker Compose

```yaml
# docker-compose.yml
version: '3.8'
services:
  gagos:
    image: netstudioge/gagos:latest
    ports:
      - "8080:8080"
    environment:
      - GAGOS_PASSWORD=${GAGOS_PASSWORD}
      - GAGOS_RUNTIME=docker
    cap_add:
      - NET_RAW
    restart: unless-stopped
```

Run:
```bash
export GAGOS_PASSWORD=$(openssl rand -base64 12)
docker-compose up -d
```

### Retrieve Password

```bash
docker exec gagos printenv GAGOS_PASSWORD
```

## Kubernetes

### Single File Deployment

```bash
# Deploy all resources
kubectl apply -f https://raw.githubusercontent.com/gaga951/gagos/main/deploy/kubernetes/gagos-all-in-one.yaml

# Get auto-generated password
kubectl get secret gagos-auth -n gagos -o jsonpath='{.data.password}' | base64 -d

# Access via port-forward
kubectl port-forward -n gagos svc/gagos 8080:8080
```

### Helm Chart

```bash
# Add repository (if published)
helm repo add gagos https://gaga951.github.io/gagos

# Install
helm install gagos gagos/gagos -n gagos --create-namespace

# Or install from local chart
helm install gagos ./charts/gagos -n gagos --create-namespace

# Get password
kubectl get secret gagos-auth -n gagos -o jsonpath='{.data.password}' | base64 -d
```

### Custom Kubernetes Manifest

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: gagos
---
apiVersion: v1
kind: Secret
metadata:
  name: gagos-auth
  namespace: gagos
type: Opaque
data:
  password: <base64-encoded-password>
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gagos
  namespace: gagos
spec:
  replicas: 1
  selector:
    matchLabels:
      app: gagos
  template:
    metadata:
      labels:
        app: gagos
    spec:
      containers:
      - name: gagos
        image: netstudioge/gagos:latest
        ports:
        - containerPort: 8080
        env:
        - name: GAGOS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: gagos-auth
              key: password
        - name: GAGOS_RUNTIME
          value: "kubernetes"
        securityContext:
          capabilities:
            add:
            - NET_RAW
---
apiVersion: v1
kind: Service
metadata:
  name: gagos
  namespace: gagos
spec:
  selector:
    app: gagos
  ports:
  - port: 8080
    targetPort: 8080
  type: ClusterIP
```

### Ingress Example

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: gagos
  namespace: gagos
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  rules:
  - host: gagos.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: gagos
            port:
              number: 8080
  tls:
  - hosts:
    - gagos.example.com
    secretName: gagos-tls
```

## Build from Source

### Requirements

- Go 1.21+
- Docker (for container build)

### Build Binary

```bash
git clone https://github.com/gaga951/gagos.git
cd gagos

# Build
go build -o gagos ./cmd/gagos

# Run
export GAGOS_PASSWORD=$(openssl rand -base64 12)
./gagos
```

### Build Container

```bash
# Build image
docker build -f deploy/docker/Dockerfile -t netstudioge/gagos:latest \
  --build-arg VERSION=$(git describe --tags --always) \
  --build-arg BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) .

# Run
docker run -d --name gagos -p 8080:8080 \
  -e GAGOS_PASSWORD="your-password" \
  --cap-add=NET_RAW \
  netstudioge/gagos:latest
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GAGOS_HOST` | `0.0.0.0` | Listen address |
| `GAGOS_PORT` | `8080` | Listen port |
| `GAGOS_PASSWORD` | (required) | Authentication password |
| `GAGOS_RUNTIME` | `docker` | Runtime environment (`docker` or `kubernetes`) |
| `GAGOS_LOG_LEVEL` | `info` | Log level (`debug`, `info`, `warn`, `error`) |

## Security Considerations

1. **Always set a strong password** - Use `openssl rand -base64 12` or similar
2. **Use HTTPS in production** - Put GAGOS behind a TLS-terminating proxy or ingress
3. **Network isolation** - Limit network access to trusted users
4. **RBAC in Kubernetes** - The ServiceAccount needs appropriate permissions for K8s features

## Troubleshooting

### Container won't start

Check logs:
```bash
docker logs gagos
# or
kubectl logs -n gagos deployment/gagos
```

### Ping/traceroute not working

Ensure `NET_RAW` capability is enabled:
```bash
docker run --cap-add=NET_RAW ...
```

### Kubernetes features not working

Check ServiceAccount permissions:
```bash
kubectl auth can-i list pods --as=system:serviceaccount:gagos:default -n gagos
```

### Password not working

Reset password:
```bash
# Docker
docker rm -f gagos
export GAGOS_PASSWORD=$(openssl rand -base64 12)
docker run ... -e GAGOS_PASSWORD="$GAGOS_PASSWORD" ...

# Kubernetes
kubectl delete secret gagos-auth -n gagos
kubectl create secret generic gagos-auth -n gagos --from-literal=password=new-password
kubectl rollout restart deployment/gagos -n gagos
```
