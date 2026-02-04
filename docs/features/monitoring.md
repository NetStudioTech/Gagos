# Monitoring

GAGOS provides a monitoring dashboard for Kubernetes cluster resource usage.

## Features

### Cluster Summary

Overview of cluster-wide metrics:

- **Total Nodes** - Number of cluster nodes
- **Total Pods** - Running pods across all namespaces
- **CPU Usage** - Cluster-wide CPU utilization
- **Memory Usage** - Cluster-wide memory utilization

### Node Metrics

View resource usage per node:

| Metric | Description |
|--------|-------------|
| Name | Node hostname |
| Status | Ready/NotReady |
| CPU Capacity | Total CPU cores |
| CPU Used | Current usage |
| CPU % | Utilization percentage |
| Memory Capacity | Total RAM |
| Memory Used | Current usage |
| Memory % | Utilization percentage |
| Pods | Running pods on node |

### Pod Metrics

View resource usage per pod:

1. Select namespace
2. View pod resource usage

| Metric | Description |
|--------|-------------|
| Name | Pod name |
| Namespace | Pod namespace |
| CPU Request | Requested CPU |
| CPU Limit | CPU limit |
| CPU Used | Current usage |
| Memory Request | Requested memory |
| Memory Limit | Memory limit |
| Memory Used | Current usage |

### Resource Quotas

Monitor namespace resource quotas:

1. Select namespace
2. View quota usage vs limits

| Metric | Description |
|--------|-------------|
| Resource | CPU, memory, pods, etc. |
| Used | Current usage |
| Hard Limit | Maximum allowed |
| % Used | Utilization |

### Horizontal Pod Autoscaler (HPA)

Monitor HPA status:

1. Select namespace
2. View HPA configurations

| Field | Description |
|-------|-------------|
| Name | HPA name |
| Target | Deployment/StatefulSet |
| Min Replicas | Minimum pods |
| Max Replicas | Maximum pods |
| Current Replicas | Actual pod count |
| CPU Target | Target CPU % |
| CPU Current | Actual CPU % |

## Usage

1. Open the Monitoring window
2. View cluster summary at top
3. Use tabs to switch between:
   - Summary
   - Nodes
   - Pods
   - Quotas
   - HPA
4. Select namespace for namespace-scoped views

## API Reference

```bash
# Cluster summary
curl http://localhost:8080/api/v1/monitoring/summary

# Node metrics
curl http://localhost:8080/api/v1/monitoring/nodes

# Pod metrics (by namespace)
curl http://localhost:8080/api/v1/monitoring/pods/default

# Resource quotas
curl http://localhost:8080/api/v1/monitoring/quotas/default

# HPA status
curl http://localhost:8080/api/v1/monitoring/hpa/default
```

## Requirements

For full metrics, your cluster needs:
- **Metrics Server** - For CPU/memory metrics
- **Resource Quotas** - For quota monitoring
- **HPA** - For autoscaler status

Install metrics-server if not present:
```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```

## Use Cases

### Capacity Planning

1. Monitor node utilization over time
2. Identify nodes approaching capacity
3. Plan node additions before issues occur

### Troubleshooting Performance

1. Check pod resource usage
2. Identify pods exceeding limits
3. Find resource-starved pods (requests vs usage)

### Quota Management

1. Monitor namespace quota consumption
2. Adjust quotas before teams hit limits
3. Identify namespaces needing more resources

### Autoscaler Verification

1. Check HPA is scaling as expected
2. Verify current replicas match load
3. Adjust min/max/target as needed
