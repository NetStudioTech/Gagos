# GAGOS CI/CD User Guide

## Table of Contents
1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Pipelines (Kubernetes-based)](#pipelines-kubernetes-based)
4. [Freestyle Jobs (SSH-based)](#freestyle-jobs-ssh-based)
5. [SSH Host Management](#ssh-host-management)
6. [Notifications](#notifications)
7. [Artifacts](#artifacts)
8. [API Reference](#api-reference)

---

## Overview

GAGOS CI/CD provides two execution modes:

| Feature | Pipelines | Freestyle Jobs |
|---------|-----------|----------------|
| Execution | Kubernetes Pods | SSH to remote hosts |
| Configuration | YAML files | UI-based |
| Use Case | Container builds, testing | Server deployments, scripts |
| Isolation | Full container isolation | Shared server environment |

### Access the CI/CD Dashboard

1. Open GAGOS in your browser
2. Click the **CI/CD** button in the toolbar
3. Navigate between tabs: Overview, Pipelines, Runs, Artifacts, SSH Hosts, Freestyle Jobs

---

## Quick Start

### Create Your First Pipeline (30 seconds)

1. Go to **CI/CD > Create Pipeline** tab
2. Click **Load Sample** to get a template
3. Modify the pipeline name and script
4. Click **Create Pipeline**
5. Go to **Pipelines** tab and click the ▶ button to run

### Create Your First Freestyle Job (1 minute)

1. Go to **CI/CD > SSH Hosts** tab
2. Click **Add Host** and enter your server details
3. Click **Test Connection** to verify
4. Go to **Freestyle Jobs** tab
5. Click **Create Job**
6. Add a build step with a shell command
7. Click **Save** then ▶ to run

---

## Pipelines (Kubernetes-based)

Pipelines execute jobs in isolated Kubernetes pods. Each job runs in its own container.

### Pipeline YAML Structure

```yaml
apiVersion: gagos.io/v1
kind: Pipeline
metadata:
  name: my-pipeline
  description: Build and test my application
  labels:
    team: backend
    env: production

spec:
  # Variables available to all jobs
  variables:
    APP_ENV: production
    LOG_LEVEL: info

  # Trigger configuration
  triggers:
    - type: webhook
      enabled: true
    - type: cron
      schedule: "0 0 * * *"  # Daily at midnight
      enabled: true

  # Jobs to execute
  jobs:
    - name: build
      image: golang:1.21
      script: |
        go build -o app ./cmd/...
      resources:
        limits:
          cpu: "2"
          memory: "2Gi"
        requests:
          cpu: "500m"
          memory: "512Mi"
      timeout: 600

    - name: test
      image: golang:1.21
      dependsOn:
        - build
      script: |
        go test ./...
      env:
        - name: TEST_DB
          value: "postgres://test:test@db:5432/test"

    - name: deploy
      image: bitnami/kubectl:latest
      dependsOn:
        - test
      script: |
        kubectl apply -f k8s/
      secrets:
        - name: kubeconfig
          mountPath: /root/.kube/config
          key: config

  # Artifacts to collect after completion
  artifacts:
    - name: binary
      path: /workspace/app
    - name: coverage
      path: /workspace/coverage.html
```

### Pipeline Configuration Reference

#### metadata
| Field | Required | Description |
|-------|----------|-------------|
| name | Yes | Unique pipeline name |
| description | No | Human-readable description |
| labels | No | Key-value metadata for organization |

#### spec.variables
Environment variables available to all jobs. Can be overridden at trigger time.

```yaml
variables:
  KEY: value
```

#### spec.triggers
| Type | Fields | Description |
|------|--------|-------------|
| webhook | enabled | HTTP endpoint for external triggers |
| cron | schedule, enabled | Cron expression for scheduled runs |

Cron format: `minute hour day month weekday`
- `0 */4 * * *` = Every 4 hours
- `0 0 * * 1-5` = Weekdays at midnight
- `30 9 * * *` = Daily at 9:30 AM

#### spec.jobs
| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| name | Yes | - | Job identifier |
| image | Yes | - | Docker image to run |
| script | Yes | - | Shell script to execute |
| workdir | No | /workspace | Working directory |
| env | No | [] | Additional environment variables |
| secrets | No | [] | Kubernetes secrets to mount |
| resources | No | - | CPU/memory limits |
| timeout | No | 600 | Timeout in seconds |
| privileged | No | false | Run with elevated privileges |
| dependsOn | No | [] | Jobs that must complete first |

#### spec.artifacts
| Field | Required | Description |
|-------|----------|-------------|
| name | Yes | Artifact identifier |
| path | Yes | Path in container to collect |

### Triggering Pipelines

#### Manual Trigger (UI)
1. Go to **Pipelines** tab
2. Click ▶ on the pipeline row
3. Optionally provide variables in the modal
4. Click **Run**

#### Webhook Trigger
```bash
# Get webhook URL from pipeline details
curl -X POST "https://gagos.example.com/api/v1/cicd/webhooks/{pipelineId}/{token}" \
  -H "Content-Type: application/json" \
  -d '{
    "ref": "refs/heads/main",
    "branch": "main",
    "commit": "abc123",
    "variables": {
      "DEPLOY_ENV": "staging"
    }
  }'
```

#### Webhook with HMAC Signature
If webhook secret is configured:
```bash
PAYLOAD='{"ref":"refs/heads/main"}'
SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "your-secret" | cut -d' ' -f2)

curl -X POST "https://gagos.example.com/api/v1/cicd/webhooks/{pipelineId}/{token}" \
  -H "Content-Type: application/json" \
  -H "X-GAGOS-Signature: sha256=$SIGNATURE" \
  -d "$PAYLOAD"
```

### Viewing Logs

1. Go to **Runs** tab
2. Click the logs icon on a run
3. Select a job to view its output
4. Logs stream in real-time for running jobs

---

## Freestyle Jobs (SSH-based)

Freestyle jobs execute commands on remote servers via SSH. Ideal for deployments and server management.

### Build Step Types

| Type | Description | Required Fields |
|------|-------------|-----------------|
| shell | Execute single command | command, host_id |
| script | Execute multi-line script | script, host_id |
| scp_push | Copy file TO remote | local_path, remote_path, host_id |
| scp_pull | Copy file FROM remote | local_path, remote_path, host_id |

### Creating a Freestyle Job

1. **Basic Information**
   - Name: Job identifier
   - Description: What this job does
   - Enabled: Whether job can be triggered

2. **Environment Variables**
   - Key-value pairs available to all steps
   - Example: `DEPLOY_DIR=/opt/myapp`

3. **Build Steps** (executed in order)
   ```
   Step 1: Deploy Code
   - Type: shell
   - Host: production-server
   - Command: cd /opt/app && git pull origin main
   - Timeout: 300s

   Step 2: Restart Service
   - Type: shell
   - Host: production-server
   - Command: systemctl restart myapp
   - Continue on Error: false
   ```

4. **Parameters** (user inputs at runtime)
   | Type | Description |
   |------|-------------|
   | string | Free text input |
   | boolean | True/false checkbox |
   | choice | Dropdown selection |

5. **Triggers**
   - Manual: Always available
   - Cron: Scheduled execution
   - Webhook: HTTP trigger endpoint

### Example: Deploy Application

**Job Configuration:**
```
Name: deploy-to-production
Description: Deploy latest code and restart services

Parameters:
  - Name: VERSION
    Type: string
    Required: true
    Description: Version tag to deploy

  - Name: SKIP_TESTS
    Type: boolean
    Default: false
    Description: Skip pre-deploy tests

Environment:
  APP_DIR: /opt/myapp
  LOG_FILE: /var/log/deploy.log

Build Steps:
  1. Backup Current Version
     Type: shell
     Host: prod-server
     Command: cp -r $APP_DIR ${APP_DIR}.backup.$(date +%Y%m%d)

  2. Pull New Version
     Type: shell
     Host: prod-server
     Command: cd $APP_DIR && git fetch && git checkout ${VERSION}

  3. Install Dependencies
     Type: shell
     Host: prod-server
     Command: cd $APP_DIR && npm install --production
     Timeout: 600

  4. Run Migrations
     Type: shell
     Host: prod-server
     Command: cd $APP_DIR && npm run migrate

  5. Restart Application
     Type: shell
     Host: prod-server
     Command: systemctl restart myapp

  6. Health Check
     Type: shell
     Host: prod-server
     Command: curl -f http://localhost:3000/health || exit 1

Triggers:
  - Type: manual
  - Type: webhook
    Enabled: true
```

### Webhook for Freestyle Jobs

```bash
curl -X POST "https://gagos.example.com/api/v1/cicd/freestyle/webhook/{token}" \
  -H "Content-Type: application/json" \
  -d '{
    "parameters": {
      "VERSION": "v1.2.3",
      "SKIP_TESTS": "false"
    }
  }'
```

---

## SSH Host Management

### Adding an SSH Host

1. Go to **SSH Hosts** tab
2. Click **Add Host**
3. Fill in connection details:

| Field | Description |
|-------|-------------|
| Name | Display name for the host |
| Host | Hostname or IP address |
| Port | SSH port (default: 22) |
| Username | SSH username |
| Auth Method | password or key |
| Password | For password auth |
| Private Key | PEM format for key auth |
| Passphrase | If private key is encrypted |
| Host Groups | Tags for organization (comma-separated) |
| Description | Notes about this host |

### Authentication Methods

**Password Authentication:**
```
Auth Method: password
Password: your-password
```

**SSH Key Authentication:**
```
Auth Method: key
Private Key:
-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAA...
-----END OPENSSH PRIVATE KEY-----

Passphrase: (if key is encrypted)
```

### Testing Connections

1. Click the ✓ icon on a host row
2. GAGOS will attempt SSH connection
3. Status updates: ✓ Success or ✗ Failed with error

### Host Key Verification

For security, you can enable host key verification:

1. Click **Scan Host Key** when adding/editing a host
2. GAGOS retrieves the server's SSH fingerprint
3. Enable **Verify Host Key** option
4. Future connections verify the fingerprint matches

---

## Notifications

Get notified when builds complete or fail.

### Creating a Notification

1. Go to notification settings (via API currently)
2. Configure:

```json
{
  "name": "Slack Alerts",
  "type": "webhook",
  "enabled": true,
  "url": "https://hooks.slack.com/services/xxx",
  "events": [
    "build_failed",
    "build_succeeded",
    "run_failed"
  ],
  "job_ids": [],           // Empty = all jobs
  "pipeline_ids": [],      // Empty = all pipelines
  "secret": "hmac-secret", // For signature verification
  "headers": {
    "X-Custom-Header": "value"
  }
}
```

### Notification Events

| Event | Trigger |
|-------|---------|
| build_started | Freestyle build begins |
| build_succeeded | Freestyle build completes successfully |
| build_failed | Freestyle build fails |
| build_cancelled | Freestyle build is cancelled |
| run_started | Pipeline run begins |
| run_succeeded | Pipeline run completes successfully |
| run_failed | Pipeline run fails |
| run_cancelled | Pipeline run is cancelled |

### Webhook Payload Format

```json
{
  "event": "build_succeeded",
  "timestamp": "2026-01-24T12:00:00Z",
  "build": {
    "id": "build-123",
    "job_id": "job-456",
    "job_name": "deploy-production",
    "build_number": 42,
    "status": "succeeded",
    "trigger_type": "manual",
    "duration_ms": 45000
  }
}
```

Header `X-GAGOS-Signature: sha256=...` included if secret is configured.

---

## Artifacts

Artifacts are files collected from pipeline jobs after execution.

### Collecting Artifacts

In pipeline YAML:
```yaml
spec:
  artifacts:
    - name: build-output
      path: /workspace/dist/app.tar.gz
    - name: test-report
      path: /workspace/coverage/report.html
```

### Downloading Artifacts

1. Go to **Artifacts** tab
2. Find your artifact in the list
3. Click the ↓ download button

Or via API:
```bash
curl -O "https://gagos.example.com/api/v1/cicd/artifacts/{id}/download"
```

### Artifact Retention

Artifacts are automatically cleaned up based on retention policy:
- Default: 30 days
- Can be configured via scheduler settings

---

## API Reference

Base URL: `/api/v1/cicd`

### Pipelines

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /pipelines | List all pipelines |
| POST | /pipelines | Create pipeline (YAML body) |
| GET | /pipelines/:id | Get pipeline details |
| PUT | /pipelines/:id | Update pipeline |
| DELETE | /pipelines/:id | Delete pipeline |
| POST | /pipelines/:id/trigger | Trigger pipeline run |

### Runs

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /runs | List all runs |
| GET | /runs/:id | Get run details |
| POST | /runs/:id/cancel | Cancel running execution |
| GET | /runs/:id/jobs/:job/logs | Get job logs |

### SSH Hosts

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /ssh/hosts | List hosts |
| POST | /ssh/hosts | Create host |
| GET | /ssh/hosts/:id | Get host |
| PUT | /ssh/hosts/:id | Update host |
| DELETE | /ssh/hosts/:id | Delete host |
| POST | /ssh/hosts/:id/test | Test connection |

### Freestyle Jobs

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /freestyle/jobs | List jobs |
| POST | /freestyle/jobs | Create job |
| GET | /freestyle/jobs/:id | Get job |
| PUT | /freestyle/jobs/:id | Update job |
| DELETE | /freestyle/jobs/:id | Delete job |
| POST | /freestyle/jobs/:id/build | Trigger build |

### Freestyle Builds

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /freestyle/builds | List builds |
| GET | /freestyle/builds/:id | Get build |
| POST | /freestyle/builds/:id/cancel | Cancel build |
| GET | /freestyle/builds/:id/logs | Get logs |

### Notifications

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /notifications | List configs |
| POST | /notifications | Create config |
| GET | /notifications/:id | Get config |
| PUT | /notifications/:id | Update config |
| DELETE | /notifications/:id | Delete config |

---

## Troubleshooting

### Pipeline job stuck in "pending"
- Check Kubernetes cluster connectivity
- Verify image is accessible
- Check resource quotas in namespace

### SSH connection failed
- Verify host is reachable: `ping hostname`
- Check SSH service: `nc -zv hostname 22`
- Verify credentials are correct
- Check firewall rules

### Webhook not triggering
- Verify webhook URL and token
- Check HMAC signature if secret is set
- Review GAGOS logs for errors

### Logs not streaming
- Check WebSocket connection
- Verify browser supports WebSocket
- Check for proxy/firewall blocking WS

---

## Best Practices

1. **Use SSH keys over passwords** - More secure and easier to rotate
2. **Enable host key verification** - Prevents MITM attacks
3. **Set appropriate timeouts** - Prevent hung jobs
4. **Use continue_on_error wisely** - Only for non-critical steps
5. **Organize hosts with groups** - Easier management at scale
6. **Configure notifications** - Know when deployments fail
7. **Review artifact retention** - Manage storage costs
8. **Use variables for environments** - Same pipeline, different configs
