# Network Tools

GAGOS provides a comprehensive suite of network diagnostic tools accessible through the web interface.

## Available Tools

### Ping

Test connectivity and measure round-trip time to a host.

**Features:**
- ICMP ping with configurable packet count
- Statistics: packets sent/received, loss percentage
- RTT metrics: min, avg, max

**Usage:**
1. Open Network Tools window
2. Select "Ping" tab
3. Enter hostname or IP address
4. Set packet count (default: 4)
5. Click "Run"

**API:**
```bash
curl -X POST http://localhost:8080/api/v1/network/ping \
  -H "Content-Type: application/json" \
  -d '{"host":"google.com","count":4}'
```

---

### DNS Lookup

Query DNS records for a domain.

**Supported Record Types:**
- `A` - IPv4 address
- `AAAA` - IPv6 address
- `CNAME` - Canonical name
- `MX` - Mail exchange
- `NS` - Name servers
- `TXT` - Text records
- `SOA` - Start of authority
- `PTR` - Pointer (reverse DNS)

**Usage:**
1. Open Network Tools window
2. Select "DNS" tab
3. Enter domain name
4. Select record type
5. Click "Lookup"

**API:**
```bash
curl -X POST http://localhost:8080/api/v1/network/dns \
  -H "Content-Type: application/json" \
  -d '{"host":"example.com","record_type":"A"}'
```

---

### Port Check

Test TCP connectivity to a specific port.

**Features:**
- TCP connection test
- Configurable timeout
- Response time measurement

**Usage:**
1. Open Network Tools window
2. Select "Port" tab
3. Enter hostname and port number
4. Click "Check"

**API:**
```bash
curl -X POST http://localhost:8080/api/v1/network/port-check \
  -H "Content-Type: application/json" \
  -d '{"host":"postgres.default.svc","port":5432}'
```

---

### Traceroute

Trace the network path to a destination.

**Features:**
- Hop-by-hop path display
- RTT for each hop
- Configurable max hops

**Usage:**
1. Open Network Tools window
2. Select "Traceroute" tab
3. Enter destination host
4. Set max hops (default: 30)
5. Click "Trace"

**API:**
```bash
curl -X POST http://localhost:8080/api/v1/network/traceroute \
  -H "Content-Type: application/json" \
  -d '{"host":"8.8.8.8","max_hops":15}'
```

---

### Telnet

Test TCP connections with optional command.

**Features:**
- Raw TCP connection
- Send initial command
- View response data

**Usage:**
1. Open Network Tools window
2. Select "Telnet" tab
3. Enter host and port
4. Optionally enter command
5. Click "Connect"

---

### Whois

Look up domain or IP registration information.

**Features:**
- Domain registration details
- Registrar information
- Expiration dates
- Name servers

**Usage:**
1. Open Network Tools window
2. Select "Whois" tab
3. Enter domain or IP
4. Click "Lookup"

**API:**
```bash
curl -X POST http://localhost:8080/api/v1/network/whois \
  -H "Content-Type: application/json" \
  -d '{"host":"example.com"}'
```

---

### SSL Check

Inspect SSL/TLS certificates.

**Features:**
- Certificate details (subject, issuer, validity)
- Chain validation
- Expiration warning
- Protocol and cipher info

**Usage:**
1. Open Network Tools window
2. Select "SSL" tab
3. Enter hostname (port optional, default 443)
4. Click "Check"

**API:**
```bash
curl -X POST http://localhost:8080/api/v1/network/ssl-check \
  -H "Content-Type: application/json" \
  -d '{"host":"example.com","port":443}'
```

---

### Curl / HTTP Request

Make HTTP requests and inspect responses.

**Features:**
- All HTTP methods (GET, POST, PUT, DELETE, etc.)
- Custom headers
- Request body
- Response headers and body
- Status code and timing

**Usage:**
1. Open Network Tools window
2. Select "Curl" tab
3. Enter URL
4. Select HTTP method
5. Add headers/body as needed
6. Click "Send"

**API:**
```bash
curl -X POST http://localhost:8080/api/v1/network/curl \
  -H "Content-Type: application/json" \
  -d '{
    "url":"https://api.example.com/health",
    "method":"GET",
    "headers":{"Authorization":"Bearer token"}
  }'
```

---

### Network Interfaces

View local network interface configuration.

**Features:**
- Interface names and status
- IP addresses (IPv4 and IPv6)
- MAC addresses
- MTU values

**Usage:**
1. Open Network Tools window
2. Select "Interfaces" tab
3. View interface list

## Use Cases

### Troubleshooting DNS Issues

1. Use **DNS Lookup** to verify record resolution
2. Compare results with expected values
3. Check different record types (A, CNAME, etc.)

### Testing Service Connectivity

1. Use **Port Check** to verify TCP connectivity
2. Use **Ping** to test basic network reachability
3. Use **Traceroute** to identify routing issues

### Validating SSL Certificates

1. Use **SSL Check** to inspect certificate details
2. Verify expiration dates
3. Check certificate chain validity

### Testing APIs

1. Use **Curl** to send HTTP requests
2. Inspect response codes and headers
3. Verify response body content
