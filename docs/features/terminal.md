# Web Terminal

GAGOS includes a browser-based terminal with full PTY support.

## Features

- **Full Terminal Emulation** - xterm.js-based terminal
- **PTY Support** - Real pseudo-terminal with job control
- **Shell Access** - /bin/sh shell environment
- **Resizable** - Terminal adapts to window size
- **WebSocket** - Real-time bidirectional communication

## Usage

1. Click the Terminal icon on the desktop
2. Terminal window opens with shell prompt
3. Type commands as you would in a local terminal
4. Resize window to adjust terminal size

## Shell Environment

The terminal runs with:

| Setting | Value |
|---------|-------|
| Shell | /bin/sh |
| Working Directory | /tmp |
| TERM | xterm-256color |
| PATH | /usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin |

## Available Tools

The container includes common utilities:

- **Network**: curl, wget, nc, nslookup, dig, ping, traceroute
- **Text**: vim, less, cat, grep, awk, sed
- **System**: ps, top, df, du, free
- **Kubernetes**: kubectl (if running in K8s with proper RBAC)

## Use Cases

### Quick Debugging

Run network tests from inside the cluster:

```bash
# Test DNS resolution
nslookup kubernetes.default.svc

# Test service connectivity
curl http://my-service.default.svc:8080/health

# Check network routes
traceroute external-api.example.com
```

### Kubectl Access

If GAGOS has Kubernetes API access:

```bash
# Check pods
kubectl get pods -n my-namespace

# View logs
kubectl logs -n my-namespace my-pod

# Exec into another pod
kubectl exec -it -n my-namespace my-pod -- /bin/sh
```

### File Operations

```bash
# Create temporary files
echo "test data" > /tmp/test.txt

# View file contents
cat /tmp/test.txt

# Download files
curl -O https://example.com/file.txt
```

## Security Notes

1. **Container Scope** - Terminal runs inside the GAGOS container
2. **Limited Filesystem** - Working directory is /tmp
3. **Network Access** - Can reach what the container can reach
4. **No Persistence** - Files in /tmp are lost on container restart

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| Ctrl+C | Interrupt/cancel |
| Ctrl+D | EOF/exit |
| Ctrl+L | Clear screen |
| Ctrl+A | Move to line start |
| Ctrl+E | Move to line end |
| Ctrl+R | Reverse search history |
| Tab | Autocomplete |

## Troubleshooting

### Terminal not connecting

1. Check WebSocket connectivity
2. Verify browser supports WebSocket
3. Check for proxy/firewall blocking WS

### Command not found

The base image has limited tools. Use available package manager to install:

```bash
# Alpine-based
apk add --no-cache <package>
```

### Terminal size issues

Resize the window - terminal automatically adjusts. If display is corrupted:

```bash
reset
```
