# Kubernetes Management

GAGOS provides comprehensive Kubernetes cluster management through an intuitive web interface.

## Supported Resources

| Category | Resources |
|----------|-----------|
| Workloads | Pods, Deployments, DaemonSets, StatefulSets, Jobs, CronJobs |
| Networking | Services, Ingresses |
| Configuration | ConfigMaps, Secrets |
| Storage | PersistentVolumeClaims |
| Cluster | Namespaces, Nodes, Events |

## Features

### Resource Listing

Browse all Kubernetes resources with real-time data:

1. Open the Kubernetes window
2. Select resource type from tabs
3. Choose namespace (or "All Namespaces")
4. View resource list with status indicators

**Columns include:**
- Name, Namespace, Status
- Age, Labels
- Resource-specific info (replicas, IP, ports, etc.)

### Auto-Refresh

Enable automatic data refresh:

1. Click the refresh toggle in the toolbar
2. Select refresh interval (5s, 10s, 30s, 60s)
3. Data updates automatically

### Resource Operations

#### View Details (Describe)

Click the "info" icon on any resource to see full YAML and status details.

#### Edit Resources

1. Click "Edit" on a resource
2. Modify YAML in the editor
3. Click "Save" to apply changes

#### Delete Resources

1. Click "Delete" on a resource
2. Confirm deletion in the modal
3. Resource is removed from cluster

#### Scale Deployments

1. Click "Scale" on a Deployment/StatefulSet
2. Enter desired replica count
3. Click "Confirm"

#### Restart Deployments

1. Click "Restart" on a Deployment/DaemonSet/StatefulSet
2. Confirm the rolling restart
3. Pods are recreated with new revision

### Pod Operations

#### View Logs

1. Click "Logs" icon on a pod
2. Select container (if multiple)
3. View live log output
4. Use "Refresh" to update

#### Exec into Container

Use the Web Terminal feature to exec into pods via `kubectl exec`.

### Create Resources

1. Click "Create" button
2. Select resource type
3. Load template or write custom YAML
4. Click "Create"

**Templates available for:**
- Pod
- Deployment
- Service
- ConfigMap
- Secret
- Job
- CronJob

## API Reference

### List Resources

```bash
# Namespaces
curl http://localhost:8080/api/v1/k8s/namespaces

# Nodes
curl http://localhost:8080/api/v1/k8s/nodes

# Pods (all namespaces)
curl http://localhost:8080/api/v1/k8s/pods

# Pods (specific namespace)
curl http://localhost:8080/api/v1/k8s/pods/default

# Services
curl http://localhost:8080/api/v1/k8s/services/kube-system

# Deployments
curl http://localhost:8080/api/v1/k8s/deployments/default
```

### Resource Operations

```bash
# Get resource YAML
curl http://localhost:8080/api/v1/k8s/resource/deployment/default/my-app

# Create resource
curl -X POST http://localhost:8080/api/v1/k8s/resource \
  -H "Content-Type: application/json" \
  -d '{"yaml":"apiVersion: v1\nkind: ConfigMap..."}'

# Update resource
curl -X PUT http://localhost:8080/api/v1/k8s/resource \
  -H "Content-Type: application/json" \
  -d '{"yaml":"..."}'

# Delete resource
curl -X DELETE http://localhost:8080/api/v1/k8s/resource/pod/default/my-pod

# Scale deployment
curl -X POST http://localhost:8080/api/v1/k8s/scale \
  -H "Content-Type: application/json" \
  -d '{"namespace":"default","name":"my-app","replicas":3}'

# Restart deployment
curl -X POST http://localhost:8080/api/v1/k8s/restart \
  -H "Content-Type: application/json" \
  -d '{"namespace":"default","name":"my-app","kind":"Deployment"}'
```

### Pod Logs

```bash
curl "http://localhost:8080/api/v1/k8s/pods/default/my-pod/logs?container=main&tail=100"
```

## Permissions

GAGOS requires a ServiceAccount with appropriate RBAC permissions:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: gagos
rules:
- apiGroups: [""]
  resources: ["namespaces", "nodes", "pods", "pods/log", "services", "configmaps", "secrets", "persistentvolumeclaims", "events"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["apps"]
  resources: ["deployments", "daemonsets", "statefulsets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["batch"]
  resources: ["jobs", "cronjobs"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["networking.k8s.io"]
  resources: ["ingresses"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

## Tips

1. **Use namespace filtering** - Select specific namespaces to reduce clutter
2. **Enable auto-refresh** - Keep data current during troubleshooting
3. **Use labels** - Filter resources by labels for better organization
4. **Check Events** - Events tab shows recent cluster activity and errors
