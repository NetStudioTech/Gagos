# API Reference

GAGOS provides a REST API for all functionality. Base URL: `/api/v1`

## Authentication

When authentication is enabled, include the session cookie in requests:

```bash
# Login to get session
curl -c cookies.txt -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"password":"your-password"}'

# Use session cookie for subsequent requests
curl -b cookies.txt http://localhost:8080/api/v1/k8s/namespaces
```

## Health & Info

### Health Check
```
GET /api/health
```

Response:
```json
{
  "status": "ok",
  "timestamp": "2026-01-26T12:00:00Z"
}
```

### Version Info
```
GET /api/version
```

Response:
```json
{
  "version": "0.10.9",
  "build_time": "2026-01-26T10:00:00Z",
  "go_version": "go1.21.5"
}
```

### Runtime Info
```
GET /api/runtime
```

Response:
```json
{
  "runtime": "kubernetes",
  "hostname": "gagos-abc123"
}
```

---

## Network Tools

### Ping
```
POST /api/v1/network/ping
```

Request:
```json
{
  "host": "google.com",
  "count": 4
}
```

### DNS Lookup
```
POST /api/v1/network/dns
```

Request:
```json
{
  "host": "example.com",
  "record_type": "A"
}
```

### Port Check
```
POST /api/v1/network/port-check
```

Request:
```json
{
  "host": "postgres.default.svc",
  "port": 5432
}
```

### Traceroute
```
POST /api/v1/network/traceroute
```

Request:
```json
{
  "host": "8.8.8.8",
  "max_hops": 30
}
```

### Telnet
```
POST /api/v1/network/telnet
```

Request:
```json
{
  "host": "example.com",
  "port": 80,
  "command": "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"
}
```

### Whois
```
POST /api/v1/network/whois
```

Request:
```json
{
  "host": "example.com"
}
```

### SSL Check
```
POST /api/v1/network/ssl-check
```

Request:
```json
{
  "host": "example.com",
  "port": 443
}
```

### Curl / HTTP Request
```
POST /api/v1/network/curl
```

Request:
```json
{
  "url": "https://api.example.com/endpoint",
  "method": "GET",
  "headers": {
    "Authorization": "Bearer token"
  },
  "body": ""
}
```

### Network Interfaces
```
GET /api/v1/network/interfaces
```

---

## Kubernetes

### Namespaces
```
GET /api/v1/k8s/namespaces
```

### Nodes
```
GET /api/v1/k8s/nodes
```

### Pods
```
GET /api/v1/k8s/pods
GET /api/v1/k8s/pods/{namespace}
```

### Pod Logs
```
GET /api/v1/k8s/pods/{namespace}/{pod}/logs?container={container}&tail={lines}
```

### Services
```
GET /api/v1/k8s/services/{namespace}
```

### Deployments
```
GET /api/v1/k8s/deployments/{namespace}
```

### DaemonSets
```
GET /api/v1/k8s/daemonsets/{namespace}
```

### StatefulSets
```
GET /api/v1/k8s/statefulsets/{namespace}
```

### Jobs
```
GET /api/v1/k8s/jobs/{namespace}
```

### CronJobs
```
GET /api/v1/k8s/cronjobs/{namespace}
```

### ConfigMaps
```
GET /api/v1/k8s/configmaps/{namespace}
```

### Secrets
```
GET /api/v1/k8s/secrets/{namespace}
```

### Ingresses
```
GET /api/v1/k8s/ingresses/{namespace}
```

### PVCs
```
GET /api/v1/k8s/pvcs/{namespace}
```

### Events
```
GET /api/v1/k8s/events/{namespace}
```

### Resource Operations
```
GET    /api/v1/k8s/resource/{kind}/{namespace}/{name}
POST   /api/v1/k8s/resource
PUT    /api/v1/k8s/resource
DELETE /api/v1/k8s/resource/{kind}/{namespace}/{name}
```

### Scale
```
POST /api/v1/k8s/scale
```

Request:
```json
{
  "namespace": "default",
  "name": "my-deployment",
  "replicas": 3
}
```

### Restart
```
POST /api/v1/k8s/restart
```

Request:
```json
{
  "namespace": "default",
  "name": "my-deployment",
  "kind": "Deployment"
}
```

---

## CI/CD

### Pipelines
```
GET    /api/v1/cicd/pipelines
POST   /api/v1/cicd/pipelines
GET    /api/v1/cicd/pipelines/{id}
PUT    /api/v1/cicd/pipelines/{id}
DELETE /api/v1/cicd/pipelines/{id}
POST   /api/v1/cicd/pipelines/{id}/trigger
```

### Runs
```
GET  /api/v1/cicd/runs
GET  /api/v1/cicd/runs/{id}
POST /api/v1/cicd/runs/{id}/cancel
GET  /api/v1/cicd/runs/{id}/jobs/{job}/logs
```

### SSH Hosts
```
GET    /api/v1/cicd/ssh/hosts
POST   /api/v1/cicd/ssh/hosts
GET    /api/v1/cicd/ssh/hosts/{id}
PUT    /api/v1/cicd/ssh/hosts/{id}
DELETE /api/v1/cicd/ssh/hosts/{id}
POST   /api/v1/cicd/ssh/hosts/{id}/test
```

### Freestyle Jobs
```
GET    /api/v1/cicd/freestyle/jobs
POST   /api/v1/cicd/freestyle/jobs
GET    /api/v1/cicd/freestyle/jobs/{id}
PUT    /api/v1/cicd/freestyle/jobs/{id}
DELETE /api/v1/cicd/freestyle/jobs/{id}
POST   /api/v1/cicd/freestyle/jobs/{id}/build
```

### Freestyle Builds
```
GET  /api/v1/cicd/freestyle/builds
GET  /api/v1/cicd/freestyle/builds/{id}
POST /api/v1/cicd/freestyle/builds/{id}/cancel
GET  /api/v1/cicd/freestyle/builds/{id}/logs
```

### Artifacts
```
GET    /api/v1/cicd/artifacts
GET    /api/v1/cicd/artifacts/{id}/download
DELETE /api/v1/cicd/artifacts/{id}
```

---

## Database - PostgreSQL

### Connect
```
POST /api/v1/database/postgres/connect
```

Request:
```json
{
  "host": "localhost",
  "port": 5432,
  "database": "mydb",
  "user": "postgres",
  "password": "secret",
  "sslmode": "disable"
}
```

### Execute Query
```
POST /api/v1/database/postgres/query
```

### Database Dump
```
POST /api/v1/database/postgres/dump
```

---

## Database - MySQL

### Connect
```
POST /api/v1/database/mysql/connect
```

Request:
```json
{
  "host": "localhost",
  "port": 3306,
  "database": "mydb",
  "user": "root",
  "password": "secret"
}
```

### Execute Query
```
POST /api/v1/database/mysql/query
```

### Database Dump
```
POST /api/v1/database/mysql/dump
```

---

## Database - Redis

### Connect
```
POST /api/v1/database/redis/connect
```

Request:
```json
{
  "host": "localhost",
  "port": 6379,
  "password": "",
  "db": 0
}
```

### Scan Keys
```
POST /api/v1/database/redis/scan
```

### Get Key
```
POST /api/v1/database/redis/get
```

### Execute Command
```
POST /api/v1/database/redis/exec
```

---

## Elasticsearch

### Connect
```
POST /api/v1/elasticsearch/connect
```

### Cluster Health
```
POST /api/v1/elasticsearch/health
```

### Cluster Stats
```
POST /api/v1/elasticsearch/stats
```

### List Indices
```
POST /api/v1/elasticsearch/indices
```

### Create Index
```
POST /api/v1/elasticsearch/index/create
```

### Delete Index
```
POST /api/v1/elasticsearch/index/delete
```

### Search Documents
```
POST /api/v1/elasticsearch/search
```

### Execute Query
```
POST /api/v1/elasticsearch/query
```

---

## S3 Storage

### Connect
```
POST /api/v1/storage/s3/connect
```

### List Buckets
```
POST /api/v1/storage/s3/buckets
```

### Create Bucket
```
POST /api/v1/storage/s3/bucket/create
```

### Delete Bucket
```
POST /api/v1/storage/s3/bucket/delete
```

### List Objects
```
POST /api/v1/storage/s3/objects
```

### Upload Object
```
POST /api/v1/storage/s3/object/upload
```

### Download Object
```
POST /api/v1/storage/s3/object/download
```

### Delete Object
```
POST /api/v1/storage/s3/object/delete
```

### Presigned URL
```
POST /api/v1/storage/s3/object/presign
```

---

## Developer Tools

### Base64 Encode
```
POST /api/v1/devtools/base64/encode
```

### Base64 Decode
```
POST /api/v1/devtools/base64/decode
```

### Generate Hashes
```
POST /api/v1/devtools/hash
```

### Parse Certificate
```
POST /api/v1/devtools/cert/parse
```

### Generate SSH Key
```
POST /api/v1/devtools/ssh/generate
```

### Format JSON
```
POST /api/v1/devtools/json/format
```

### Minify JSON
```
POST /api/v1/devtools/json/minify
```

### Text Diff
```
POST /api/v1/devtools/diff
```

---

## Monitoring

### Summary
```
GET /api/v1/monitoring/summary
```

### Nodes
```
GET /api/v1/monitoring/nodes
```

### Pods
```
GET /api/v1/monitoring/pods/{namespace}
```

### Resource Quotas
```
GET /api/v1/monitoring/quotas/{namespace}
```

### HPA
```
GET /api/v1/monitoring/hpa/{namespace}
```

---

## Error Responses

All errors return JSON:

```json
{
  "error": "error message",
  "details": "optional details"
}
```

Common HTTP status codes:
- `400` - Bad Request (invalid input)
- `401` - Unauthorized (not logged in)
- `403` - Forbidden (insufficient permissions)
- `404` - Not Found
- `500` - Internal Server Error
