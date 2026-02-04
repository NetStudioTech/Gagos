// Kubernetes Module for GAGOS

import { API_BASE } from './app.js';
import { saveState } from './state.js';
import { escapeHtml } from './utils.js';

// State
let selectedNamespace = '';
let namespacesList = [];
let editModeEnabled = false;
let editModeTimer = null;
let editModeSeconds = 300; // 5 minutes
let autoRefreshEnabled = false;
let autoRefreshTimer = null;
let refreshInterval = 10; // seconds
let currentResource = { type: '', namespace: '', name: '' };

// Resource templates
export const resourceTemplates = {
    deployment: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  labels:
    app: my-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
      - name: my-app
        image: nginx:latest
        ports:
        - containerPort: 80
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "128Mi"
            cpu: "500m"`,

    service: `apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  selector:
    app: my-app
  ports:
  - port: 80
    targetPort: 80
  type: ClusterIP`,

    configmap: `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
data:
  key1: value1
  key2: value2
  config.json: |
    {
      "setting": "value"
    }`,

    secret: `apiVersion: v1
kind: Secret
metadata:
  name: my-secret
type: Opaque
stringData:
  username: admin
  password: changeme`,

    ingress: `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-ingress
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  ingressClassName: nginx
  rules:
  - host: example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: my-service
            port:
              number: 80`,

    pod: `apiVersion: v1
kind: Pod
metadata:
  name: my-pod
  labels:
    app: my-pod
spec:
  containers:
  - name: main
    image: nginx:latest
    ports:
    - containerPort: 80`,

    cronjob: `apiVersion: batch/v1
kind: CronJob
metadata:
  name: my-cronjob
spec:
  schedule: "*/5 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: job
            image: busybox:latest
            command:
            - /bin/sh
            - -c
            - echo "Hello from CronJob"
          restartPolicy: OnFailure`,

    job: `apiVersion: batch/v1
kind: Job
metadata:
  name: my-job
spec:
  template:
    spec:
      containers:
      - name: job
        image: busybox:latest
        command:
        - /bin/sh
        - -c
        - echo "Hello from Job" && sleep 10
      restartPolicy: Never
  backoffLimit: 4`,

    pvc: `apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi`,

    serviceaccount: `apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-serviceaccount`,

    daemonset: `apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: my-daemonset
  labels:
    app: my-daemonset
spec:
  selector:
    matchLabels:
      app: my-daemonset
  template:
    metadata:
      labels:
        app: my-daemonset
    spec:
      containers:
      - name: main
        image: nginx:latest
        resources:
          limits:
            memory: "128Mi"
            cpu: "100m"`,

    statefulset: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: my-statefulset
spec:
  serviceName: my-statefulset
  replicas: 1
  selector:
    matchLabels:
      app: my-statefulset
  template:
    metadata:
      labels:
        app: my-statefulset
    spec:
      containers:
      - name: main
        image: nginx:latest
        ports:
        - containerPort: 80`
};

// Tab switching
export function showK8sTab(tabId) {
    document.querySelectorAll('#window-kubernetes .tab-btn').forEach(b => b.classList.remove('active'));
    document.querySelectorAll('#window-kubernetes .tab-content').forEach(c => c.classList.remove('active'));
    const tabBtn = document.querySelector(`#window-kubernetes .tab-btn[onclick="showK8sTab('${tabId}')"]`);
    if (tabBtn) tabBtn.classList.add('active');
    const tabContent = document.getElementById('k8s-tab-' + tabId);
    if (tabContent) tabContent.classList.add('active');
    saveState();
}

// Load all K8s data
export async function loadK8sData() {
    await Promise.all([
        loadNamespaces(), loadNodes(), loadPods(), loadServices(), loadDeployments(),
        loadDaemonSets(), loadStatefulSets(), loadJobs(), loadCronJobs(),
        loadConfigMaps(), loadSecrets(), loadIngresses(), loadPVCs(), loadEvents()
    ]);
}

export async function loadNamespaces() {
    try {
        const r = await fetch(`${API_BASE}/k8s/namespaces`);
        const d = await r.json();
        document.getElementById('ns-count').textContent = d.count || 0;
        namespacesList = d.namespaces || [];

        // Update namespace pills
        const list = document.getElementById('namespaces-list');
        list.innerHTML = '';

        const allPill = document.createElement('span');
        allPill.className = 'namespace-pill' + (selectedNamespace === '' ? ' active' : '');
        allPill.textContent = 'All';
        allPill.onclick = () => { selectedNamespace = ''; loadK8sData(); };
        list.appendChild(allPill);

        namespacesList.forEach(ns => {
            const pill = document.createElement('span');
            pill.className = 'namespace-pill' + (selectedNamespace === ns.name ? ' active' : '');
            pill.textContent = ns.name;
            pill.onclick = () => { selectedNamespace = ns.name; loadK8sData(); };
            list.appendChild(pill);
        });

        // Update dropdowns
        updateNamespaceDropdowns();
    } catch (e) { console.error(e); }
}

function updateNamespaceDropdowns() {
    const dropdowns = [
        'pods-namespace-select', 'svc-namespace-select', 'deploy-namespace-select',
        'ds-namespace-select', 'sts-namespace-select', 'job-namespace-select',
        'cj-namespace-select', 'cm-namespace-select', 'secret-namespace-select',
        'ing-namespace-select', 'pvc-namespace-select', 'event-namespace-select'
    ];
    dropdowns.forEach(id => {
        const sel = document.getElementById(id);
        if (sel) {
            const current = sel.value;
            sel.innerHTML = '<option value="">All Namespaces</option>';
            namespacesList.forEach(ns => {
                sel.innerHTML += `<option value="${ns.name}">${ns.name}</option>`;
            });
            sel.value = current;
        }
    });
}

export async function loadPods() {
    try {
        const url = selectedNamespace ? `${API_BASE}/k8s/pods/${selectedNamespace}` : `${API_BASE}/k8s/pods`;
        const r = await fetch(url);
        const d = await r.json();
        document.getElementById('pod-count').textContent = d.count || 0;

        const nsLabel = selectedNamespace ? `(${selectedNamespace})` : '(all)';
        document.getElementById('pods-ns-label').textContent = nsLabel;
        const fullLabel = document.getElementById('pods-full-ns-label');
        if (fullLabel) fullLabel.textContent = nsLabel;

        // Preview table (overview tab) - first 5
        const previewTbody = document.getElementById('pods-tbody-preview');
        if (previewTbody) {
            previewTbody.innerHTML = '';
            const previewPods = (d.pods || []).slice(0, 5);
            if (previewPods.length > 0) {
                previewPods.forEach(p => {
                    const statusClass = p.status === 'Running' ? 'status-running' : p.status === 'Pending' ? 'status-pending' : 'status-failed';
                    previewTbody.innerHTML += `<tr><td style="font-family:monospace;font-size:12px">${p.name}</td><td>${p.namespace}</td><td class="${statusClass}">${p.status}</td><td>${p.ready||'-'}</td><td>${p.restarts||0}</td></tr>`;
                });
            } else {
                previewTbody.innerHTML = '<tr><td colspan="5" style="text-align:center;color:#6a6a7a">No pods found</td></tr>';
            }
        }

        // Full table (pods tab)
        const tbody = document.getElementById('pods-tbody');
        tbody.innerHTML = '';
        if (d.pods?.length > 0) {
            d.pods.forEach(p => {
                const statusClass = p.status === 'Running' ? 'status-running' : p.status === 'Pending' ? 'status-pending' : 'status-failed';
                tbody.innerHTML += `<tr><td style="font-family:monospace;font-size:12px">${p.name}</td><td>${p.namespace}</td><td class="${statusClass}">${p.status}</td><td>${p.ready||'-'}</td><td>${p.restarts||0}</td><td>${p.age||'-'}</td>${getActionButtons('pod', p.namespace, p.name)}</tr>`;
            });
        } else {
            tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;color:#6a6a7a">No pods found</td></tr>';
        }
    } catch (e) { console.error(e); }
}

export function loadPodsForNamespace() {
    selectedNamespace = document.getElementById('pods-namespace-select').value;
    loadPods();
}

export async function loadNodes() {
    try {
        const r = await fetch(`${API_BASE}/k8s/nodes`);
        const d = await r.json();
        document.getElementById('node-count').textContent = d.count || 0;
        const tbody = document.getElementById('nodes-tbody');
        tbody.innerHTML = '';
        if (d.nodes?.length > 0) {
            d.nodes.forEach(n => {
                const statusClass = n.status === 'Ready' ? 'status-running' : 'status-failed';
                const describeBtn = `<td class="action-cell"><button class="row-action-btn describe" onclick="describeResource('node', '', '${n.name}')">Describe</button></td>`;
                tbody.innerHTML += `<tr><td style="font-family:monospace;font-size:12px">${n.name}</td><td class="${statusClass}">${n.status}</td><td>${n.roles||'-'}</td><td>${n.kubelet_version||'-'}</td><td>${n.os||'-'}</td><td>${n.kernel_version||'-'}</td>${describeBtn}</tr>`;
            });
        } else {
            tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;color:#6a6a7a">No nodes found</td></tr>';
        }
    } catch (e) { console.error(e); }
}

export async function loadServices() {
    try {
        const url = selectedNamespace ? `${API_BASE}/k8s/services/${selectedNamespace}` : `${API_BASE}/k8s/services`;
        const r = await fetch(url);
        const d = await r.json();
        document.getElementById('svc-count').textContent = d.count || 0;
        const tbody = document.getElementById('services-tbody');
        tbody.innerHTML = '';
        if (d.services?.length > 0) {
            d.services.forEach(s => {
                const portsStr = s.ports?.map(p => `${p.port}${p.node_port ? ':'+p.node_port : ''}/${p.protocol}`).join(', ') || '-';
                tbody.innerHTML += `<tr><td style="font-family:monospace;font-size:12px">${s.name}</td><td>${s.namespace}</td><td>${s.type}</td><td style="font-family:monospace;font-size:12px">${s.cluster_ip||'-'}</td><td style="font-family:monospace;font-size:12px">${portsStr}</td>${getActionButtons('service', s.namespace, s.name)}</tr>`;
            });
        } else {
            tbody.innerHTML = '<tr><td colspan="6" style="text-align:center;color:#6a6a7a">No services found</td></tr>';
        }
    } catch (e) { console.error(e); }
}

export function loadServicesForNamespace() {
    selectedNamespace = document.getElementById('svc-namespace-select').value;
    loadServices();
}

export async function loadDeployments() {
    try {
        const url = selectedNamespace ? `${API_BASE}/k8s/deployments/${selectedNamespace}` : `${API_BASE}/k8s/deployments`;
        const r = await fetch(url);
        const d = await r.json();
        const countEl = document.getElementById('deploy-count');
        if (countEl) countEl.textContent = d.count || 0;
        const tbody = document.getElementById('deployments-tbody');
        if (!tbody) return;
        tbody.innerHTML = '';
        if (d.deployments?.length > 0) {
            d.deployments.forEach(dep => {
                const ready = dep.ready || `${dep.ready_replicas || 0}/${dep.replicas || 0}`;
                const statusClass = ready.split('/')[0] >= ready.split('/')[1] ? 'status-running' : 'status-pending';
                tbody.innerHTML += `<tr><td style="font-family:monospace;font-size:12px">${dep.name}</td><td>${dep.namespace}</td><td class="${statusClass}">${ready}</td><td>${dep.available||dep.available_replicas||0}</td><td>${dep.age||'-'}</td>${getActionButtons('deployment', dep.namespace, dep.name)}</tr>`;
            });
        } else {
            tbody.innerHTML = '<tr><td colspan="6" style="text-align:center;color:#6a6a7a">No deployments found</td></tr>';
        }
    } catch (e) { console.error(e); }
}

export function loadDeploymentsForNamespace() {
    selectedNamespace = document.getElementById('deploy-namespace-select').value;
    loadDeployments();
}

export async function loadDaemonSets() {
    try {
        const url = selectedNamespace ? `${API_BASE}/k8s/daemonsets/${selectedNamespace}` : `${API_BASE}/k8s/daemonsets`;
        const r = await fetch(url);
        const d = await r.json();
        const tbody = document.getElementById('daemonsets-tbody');
        if (!tbody) return;
        tbody.innerHTML = '';
        if (d.daemonsets?.length > 0) {
            d.daemonsets.forEach(ds => {
                const statusClass = ds.ready >= ds.desired ? 'status-running' : 'status-pending';
                tbody.innerHTML += `<tr><td style="font-family:monospace;font-size:12px">${ds.name}</td><td>${ds.namespace}</td><td>${ds.desired}</td><td class="${statusClass}">${ds.ready}</td><td>${ds.available||0}</td><td>${ds.age||'-'}</td>${getActionButtons('daemonset', ds.namespace, ds.name)}</tr>`;
            });
        } else {
            tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;color:#6a6a7a">No daemonsets found</td></tr>';
        }
    } catch (e) { console.error(e); }
}

export function loadDaemonSetsForNamespace() {
    selectedNamespace = document.getElementById('ds-namespace-select').value;
    loadDaemonSets();
}

export async function loadStatefulSets() {
    try {
        const url = selectedNamespace ? `${API_BASE}/k8s/statefulsets/${selectedNamespace}` : `${API_BASE}/k8s/statefulsets`;
        const r = await fetch(url);
        const d = await r.json();
        const tbody = document.getElementById('statefulsets-tbody');
        if (!tbody) return;
        tbody.innerHTML = '';
        if (d.statefulsets?.length > 0) {
            d.statefulsets.forEach(ss => {
                const parts = ss.ready.split('/');
                const statusClass = parts[0] >= parts[1] ? 'status-running' : 'status-pending';
                tbody.innerHTML += `<tr><td style="font-family:monospace;font-size:12px">${ss.name}</td><td>${ss.namespace}</td><td class="${statusClass}">${ss.ready}</td><td>${ss.age||'-'}</td>${getActionButtons('statefulset', ss.namespace, ss.name)}</tr>`;
            });
        } else {
            tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;color:#6a6a7a">No statefulsets found</td></tr>';
        }
    } catch (e) { console.error(e); }
}

export function loadStatefulSetsForNamespace() {
    selectedNamespace = document.getElementById('sts-namespace-select').value;
    loadStatefulSets();
}

export async function loadJobs() {
    try {
        const url = selectedNamespace ? `${API_BASE}/k8s/jobs/${selectedNamespace}` : `${API_BASE}/k8s/jobs`;
        const r = await fetch(url);
        const d = await r.json();
        const tbody = document.getElementById('jobs-tbody');
        if (!tbody) return;
        tbody.innerHTML = '';
        if (d.jobs?.length > 0) {
            d.jobs.forEach(job => {
                const statusClass = job.status === 'Complete' ? 'status-running' : job.status === 'Failed' ? 'status-failed' : 'status-pending';
                tbody.innerHTML += `<tr><td style="font-family:monospace;font-size:12px">${job.name}</td><td>${job.namespace}</td><td>${job.completions}</td><td>${job.duration||'-'}</td><td class="${statusClass}">${job.status}</td><td>${job.age||'-'}</td>${getActionButtons('job', job.namespace, job.name)}</tr>`;
            });
        } else {
            tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;color:#6a6a7a">No jobs found</td></tr>';
        }
    } catch (e) { console.error(e); }
}

export function loadJobsForNamespace() {
    selectedNamespace = document.getElementById('job-namespace-select').value;
    loadJobs();
}

export async function loadCronJobs() {
    try {
        const url = selectedNamespace ? `${API_BASE}/k8s/cronjobs/${selectedNamespace}` : `${API_BASE}/k8s/cronjobs`;
        const r = await fetch(url);
        const d = await r.json();
        const tbody = document.getElementById('cronjobs-tbody');
        if (!tbody) return;
        tbody.innerHTML = '';
        if (d.cronjobs?.length > 0) {
            d.cronjobs.forEach(cj => {
                const suspendClass = cj.suspend ? 'status-pending' : '';
                tbody.innerHTML += `<tr><td style="font-family:monospace;font-size:12px">${cj.name}</td><td>${cj.namespace}</td><td style="font-family:monospace;font-size:11px">${cj.schedule}</td><td class="${suspendClass}">${cj.suspend?'Yes':'No'}</td><td>${cj.active}</td><td>${cj.last_schedule||'-'}</td><td>${cj.age||'-'}</td>${getActionButtons('cronjob', cj.namespace, cj.name)}</tr>`;
            });
        } else {
            tbody.innerHTML = '<tr><td colspan="8" style="text-align:center;color:#6a6a7a">No cronjobs found</td></tr>';
        }
    } catch (e) { console.error(e); }
}

export function loadCronJobsForNamespace() {
    selectedNamespace = document.getElementById('cj-namespace-select').value;
    loadCronJobs();
}

export async function loadConfigMaps() {
    try {
        const url = selectedNamespace ? `${API_BASE}/k8s/configmaps/${selectedNamespace}` : `${API_BASE}/k8s/configmaps`;
        const r = await fetch(url);
        const d = await r.json();
        const tbody = document.getElementById('configmaps-tbody');
        if (!tbody) return;
        tbody.innerHTML = '';
        if (d.configmaps?.length > 0) {
            d.configmaps.forEach(cm => {
                tbody.innerHTML += `<tr><td style="font-family:monospace;font-size:12px">${cm.name}</td><td>${cm.namespace}</td><td>${cm.data_count||0}</td><td>${cm.age||'-'}</td>${getActionButtons('configmap', cm.namespace, cm.name)}</tr>`;
            });
        } else {
            tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;color:#6a6a7a">No configmaps found</td></tr>';
        }
    } catch (e) { console.error(e); }
}

export function loadConfigMapsForNamespace() {
    selectedNamespace = document.getElementById('cm-namespace-select').value;
    loadConfigMaps();
}

export async function loadSecrets() {
    try {
        const url = selectedNamespace ? `${API_BASE}/k8s/secrets/${selectedNamespace}` : `${API_BASE}/k8s/secrets`;
        const r = await fetch(url);
        const d = await r.json();
        const tbody = document.getElementById('secrets-tbody');
        if (!tbody) return;
        tbody.innerHTML = '';
        if (d.secrets?.length > 0) {
            d.secrets.forEach(s => {
                tbody.innerHTML += `<tr><td style="font-family:monospace;font-size:12px">${s.name}</td><td>${s.namespace}</td><td style="font-size:11px">${s.type}</td><td>${s.data_count||0}</td><td>${s.age||'-'}</td>${getActionButtons('secret', s.namespace, s.name)}</tr>`;
            });
        } else {
            tbody.innerHTML = '<tr><td colspan="6" style="text-align:center;color:#6a6a7a">No secrets found</td></tr>';
        }
    } catch (e) { console.error(e); }
}

export function loadSecretsForNamespace() {
    selectedNamespace = document.getElementById('secret-namespace-select').value;
    loadSecrets();
}

export async function loadIngresses() {
    try {
        const url = selectedNamespace ? `${API_BASE}/k8s/ingresses/${selectedNamespace}` : `${API_BASE}/k8s/ingresses`;
        const r = await fetch(url);
        const d = await r.json();
        const tbody = document.getElementById('ingresses-tbody');
        if (!tbody) return;
        tbody.innerHTML = '';
        if (d.ingresses?.length > 0) {
            d.ingresses.forEach(ing => {
                tbody.innerHTML += `<tr><td style="font-family:monospace;font-size:12px">${ing.name}</td><td>${ing.namespace}</td><td>${ing.class||'-'}</td><td style="font-size:11px">${ing.hosts?.join(', ')||'-'}</td><td style="font-family:monospace;font-size:11px">${ing.address||'-'}</td><td>${ing.age||'-'}</td>${getActionButtons('ingress', ing.namespace, ing.name)}</tr>`;
            });
        } else {
            tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;color:#6a6a7a">No ingresses found</td></tr>';
        }
    } catch (e) { console.error(e); }
}

export function loadIngressesForNamespace() {
    selectedNamespace = document.getElementById('ing-namespace-select').value;
    loadIngresses();
}

export async function loadPVCs() {
    try {
        const url = selectedNamespace ? `${API_BASE}/k8s/pvcs/${selectedNamespace}` : `${API_BASE}/k8s/pvcs`;
        const r = await fetch(url);
        const d = await r.json();
        const tbody = document.getElementById('pvcs-tbody');
        if (!tbody) return;
        tbody.innerHTML = '';
        if (d.pvcs?.length > 0) {
            d.pvcs.forEach(pvc => {
                const statusClass = pvc.status === 'Bound' ? 'status-running' : 'status-pending';
                tbody.innerHTML += `<tr><td style="font-family:monospace;font-size:12px">${pvc.name}</td><td>${pvc.namespace}</td><td class="${statusClass}">${pvc.status}</td><td style="font-family:monospace;font-size:11px">${pvc.volume||'-'}</td><td>${pvc.capacity||'-'}</td><td>${pvc.storage_class||'-'}</td><td>${pvc.age||'-'}</td>${getActionButtons('pvc', pvc.namespace, pvc.name)}</tr>`;
            });
        } else {
            tbody.innerHTML = '<tr><td colspan="8" style="text-align:center;color:#6a6a7a">No PVCs found</td></tr>';
        }
    } catch (e) { console.error(e); }
}

export function loadPVCsForNamespace() {
    selectedNamespace = document.getElementById('pvc-namespace-select').value;
    loadPVCs();
}

export async function loadEvents() {
    try {
        const url = selectedNamespace ? `${API_BASE}/k8s/events/${selectedNamespace}` : `${API_BASE}/k8s/events`;
        const r = await fetch(url);
        const d = await r.json();
        const tbody = document.getElementById('events-tbody');
        if (!tbody) return;
        tbody.innerHTML = '';
        if (d.events?.length > 0) {
            // Sort by last seen, most recent first
            const events = d.events.sort((a, b) => new Date(b.last_seen) - new Date(a.last_seen));
            events.slice(0, 100).forEach(e => {
                const typeClass = e.type === 'Warning' ? 'status-pending' : e.type === 'Normal' ? '' : 'status-failed';
                const msgShort = e.message?.length > 60 ? e.message.substring(0, 60) + '...' : e.message;
                tbody.innerHTML += `<tr><td class="${typeClass}">${e.type}</td><td>${e.reason}</td><td style="font-size:11px">${e.object}</td><td style="font-size:11px" title="${e.message}">${msgShort||'-'}</td><td>${e.count||1}</td><td>${e.age||'-'}</td>${getActionButtons('event', e.namespace, e.name)}</tr>`;
            });
        } else {
            tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;color:#6a6a7a">No events found</td></tr>';
        }
    } catch (e) { console.error(e); }
}

export function loadEventsForNamespace() {
    selectedNamespace = document.getElementById('event-namespace-select').value;
    loadEvents();
}

// Edit Mode
export function toggleEditMode() {
    if (editModeEnabled) {
        disableEditMode();
    } else {
        showModal('edit-mode-confirm-modal');
    }
}

export function enableEditMode() {
    closeModal('edit-mode-confirm-modal');
    editModeEnabled = true;
    editModeSeconds = 300;

    const btn = document.getElementById('edit-mode-btn');
    const label = document.getElementById('edit-mode-label');
    const timer = document.getElementById('edit-mode-timer');
    const createBtn = document.getElementById('create-resource-btn');

    btn.classList.add('active');
    label.textContent = 'Disable Edit Mode';
    timer.classList.add('active');
    createBtn.style.display = 'flex';
    updateTimerDisplay();

    // Start countdown
    editModeTimer = setInterval(() => {
        editModeSeconds--;
        updateTimerDisplay();
        if (editModeSeconds <= 0) {
            disableEditMode();
        }
    }, 1000);

    // Refresh tables to show action buttons
    loadK8sData();
}

export function disableEditMode() {
    editModeEnabled = false;
    if (editModeTimer) {
        clearInterval(editModeTimer);
        editModeTimer = null;
    }

    const btn = document.getElementById('edit-mode-btn');
    const label = document.getElementById('edit-mode-label');
    const timer = document.getElementById('edit-mode-timer');
    const createBtn = document.getElementById('create-resource-btn');

    btn.classList.remove('active');
    label.textContent = 'Enable Edit Mode';
    timer.classList.remove('active');
    createBtn.style.display = 'none';

    // Refresh tables to hide action buttons
    loadK8sData();
}

function updateTimerDisplay() {
    const mins = Math.floor(editModeSeconds / 60);
    const secs = editModeSeconds % 60;
    document.getElementById('edit-mode-timer').textContent =
        `${mins}:${secs.toString().padStart(2, '0')}`;
}

// Generate action buttons based on edit mode state
export function getActionButtons(resourceType, namespace, name) {
    const describeBtn = `<button class="row-action-btn describe" onclick="describeResource('${resourceType}', '${namespace}', '${name}')">Describe</button>`;

    if (!editModeEnabled) {
        return `<td class="action-cell">${describeBtn}</td>`;
    }

    let actions = describeBtn;

    if (resourceType === 'pod') {
        actions += `<button class="row-action-btn logs" onclick="viewPodLogs('${namespace}', '${name}')">Logs</button>`;
    }

    if (resourceType === 'deployment') {
        actions += `<button class="row-action-btn scale" onclick="showScaleModal('${namespace}', '${name}')">Scale</button>`;
        actions += `<button class="row-action-btn restart" onclick="showRestartModal('${namespace}', '${name}')">Restart</button>`;
    }

    actions += `<button class="row-action-btn edit" onclick="editResource('${resourceType}', '${namespace}', '${name}')">Edit</button>`;
    actions += `<button class="row-action-btn delete" onclick="showDeleteModal('${resourceType}', '${namespace}', '${name}')">Delete</button>`;

    return `<td class="action-cell">${actions}</td>`;
}

// Modal helpers
export function showModal(modalId) {
    document.getElementById(modalId).classList.add('active');
}

export function closeModal(modalId) {
    document.getElementById(modalId).classList.remove('active');
}

// Describe resource
export async function describeResource(resourceType, namespace, name) {
    currentResource = { type: resourceType, namespace, name };
    const modal = document.getElementById('describe-modal');
    const title = document.getElementById('describe-modal-title');
    const info = document.getElementById('describe-resource-info');
    const yaml = document.getElementById('describe-yaml');
    const decodeBtn = document.getElementById('describe-decode-btn');
    const decodedOutput = document.getElementById('describe-decoded-output');

    title.textContent = `${resourceType.charAt(0).toUpperCase() + resourceType.slice(1)}: ${name}`;
    info.innerHTML = `
        <div class="resource-info-item"><span class="label">Name</span><span class="value">${name}</span></div>
        <div class="resource-info-item"><span class="label">Namespace</span><span class="value">${namespace || 'N/A'}</span></div>
        <div class="resource-info-item"><span class="label">Type</span><span class="value">${resourceType}</span></div>
    `;
    yaml.textContent = 'Loading...';
    decodeBtn.style.display = 'none';
    decodedOutput.style.display = 'none';
    decodedOutput.textContent = '';
    showModal('describe-modal');

    try {
        const url = namespace
            ? `${API_BASE}/k8s/${resourceType}/${namespace}/${name}`
            : `${API_BASE}/k8s/${resourceType}/${name}`;
        const r = await fetch(url);
        const d = await r.json();
        yaml.textContent = d.yaml || JSON.stringify(d, null, 2);
        // Show decode button for secrets
        if (resourceType === 'secret') {
            decodeBtn.style.display = '';
        }
    } catch (e) {
        yaml.textContent = 'Error loading resource: ' + e.message;
    }
}

// Decode secret data values from describe modal
export function decodeDescribedSecret() {
    const yaml = document.getElementById('describe-yaml').textContent;
    const output = document.getElementById('describe-decoded-output');
    const btn = document.getElementById('describe-decode-btn');

    // Parse data: section from YAML
    const lines = yaml.split('\n');
    const decoded = {};
    let inData = false;
    let dataIndent = -1;

    for (const line of lines) {
        if (/^data:\s*$/.test(line)) {
            inData = true;
            continue;
        }
        if (inData) {
            // Check if line is indented (part of data section)
            const match = line.match(/^(\s+)(\S+):\s*(.*)$/);
            if (match) {
                if (dataIndent === -1) dataIndent = match[1].length;
                if (match[1].length === dataIndent) {
                    const key = match[2];
                    const val = match[3].trim();
                    try {
                        decoded[key] = atob(val);
                    } catch {
                        decoded[key] = '(binary or invalid base64)';
                    }
                    continue;
                }
            }
            // Non-indented line = end of data section
            if (line.match(/^\S/) && line.trim() !== '') {
                inData = false;
            }
        }
    }

    if (Object.keys(decoded).length === 0) {
        output.style.display = 'block';
        output.style.color = '#f59e0b';
        output.textContent = 'No data fields found in this secret.';
        return;
    }

    // Format output
    let text = '';
    for (const [key, val] of Object.entries(decoded)) {
        text += `── ${key} ──\n${val}\n\n`;
    }
    output.style.display = 'block';
    output.style.color = '#e2e8f0';
    output.textContent = text.trimEnd();
    btn.textContent = 'Decoded ✓';
    btn.style.opacity = '0.7';
}

// Edit resource
export async function editResource(resourceType, namespace, name) {
    currentResource = { type: resourceType, namespace, name };
    const modal = document.getElementById('edit-modal');
    const title = document.getElementById('edit-modal-title');
    const editor = document.getElementById('edit-yaml-editor');

    title.textContent = `Edit ${resourceType}: ${name}`;
    editor.value = 'Loading...';
    showModal('edit-modal');

    try {
        const url = namespace
            ? `${API_BASE}/k8s/${resourceType}/${namespace}/${name}`
            : `${API_BASE}/k8s/${resourceType}/${name}`;
        const r = await fetch(url);
        const d = await r.json();
        editor.value = d.yaml || JSON.stringify(d, null, 2);
    } catch (e) {
        editor.value = '# Error loading resource: ' + e.message;
    }
}

export async function saveResourceEdit() {
    const editor = document.getElementById('edit-yaml-editor');
    const yaml = editor.value;
    const { type, namespace, name } = currentResource;

    try {
        const url = namespace
            ? `${API_BASE}/k8s/${type}/${namespace}/${name}`
            : `${API_BASE}/k8s/${type}/${name}`;
        const r = await fetch(url, {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ yaml })
        });
        const d = await r.json();
        if (d.success) {
            closeModal('edit-modal');
            loadK8sData();
        } else {
            alert('Error: ' + (d.error || 'Unknown error'));
        }
    } catch (e) {
        alert('Error saving: ' + e.message);
    }
}

// Delete resource
export function showDeleteModal(resourceType, namespace, name) {
    currentResource = { type: resourceType, namespace, name };
    document.getElementById('delete-resource-name').textContent = name;
    document.getElementById('delete-resource-type').textContent = resourceType;
    document.getElementById('delete-resource-namespace').textContent = namespace || 'N/A';
    const confirmInput = document.getElementById('delete-confirm-input');
    const confirmBtn = document.getElementById('delete-confirm-btn');
    confirmInput.value = '';
    confirmBtn.disabled = true;
    confirmBtn.style.opacity = '0.5';
    confirmBtn.style.cursor = 'not-allowed';
    showModal('delete-modal');
}

export async function confirmDelete() {
    const { type, namespace, name } = currentResource;

    try {
        const url = namespace
            ? `${API_BASE}/k8s/${type}/${namespace}/${name}`
            : `${API_BASE}/k8s/${type}/${name}`;
        const r = await fetch(url, { method: 'DELETE' });
        const d = await r.json();
        if (d.success) {
            closeModal('delete-modal');
            loadK8sData();
        } else {
            alert('Error: ' + (d.error || 'Unknown error'));
        }
    } catch (e) {
        alert('Error deleting: ' + e.message);
    }
}

// Pod logs
export async function viewPodLogs(namespace, name) {
    currentResource = { type: 'pod', namespace, name };
    const logs = document.getElementById('logs-content');
    document.getElementById('logs-modal-title').textContent = `Logs: ${name}`;
    logs.textContent = 'Loading...';
    showModal('logs-modal');

    try {
        const r = await fetch(`${API_BASE}/k8s/pod/${namespace}/${name}/logs?tail=200`);
        const d = await r.json();
        logs.textContent = d.logs || 'No logs available';
    } catch (e) {
        logs.textContent = 'Error loading logs: ' + e.message;
    }
}

export async function refreshLogs() {
    const { namespace, name } = currentResource;
    const logs = document.getElementById('logs-content');
    logs.textContent = 'Refreshing...';

    try {
        const r = await fetch(`${API_BASE}/k8s/pod/${namespace}/${name}/logs?tail=200`);
        const d = await r.json();
        logs.textContent = d.logs || 'No logs available';
    } catch (e) {
        logs.textContent = 'Error loading logs: ' + e.message;
    }
}

// Scale deployment
export function showScaleModal(namespace, name) {
    currentResource = { type: 'deployment', namespace, name };
    document.getElementById('scale-deployment-name').textContent = name;
    document.getElementById('scale-replicas').value = 1;
    showModal('scale-modal');
}

export async function confirmScale() {
    const { namespace, name } = currentResource;
    const replicas = parseInt(document.getElementById('scale-replicas').value, 10);

    if (isNaN(replicas) || replicas < 0) {
        alert('Please enter a valid number of replicas');
        return;
    }

    try {
        const r = await fetch(`${API_BASE}/k8s/deployment/${namespace}/${name}/scale`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ replicas })
        });
        const d = await r.json();
        if (d.success) {
            closeModal('scale-modal');
            loadK8sData();
        } else {
            alert('Error: ' + (d.error || 'Unknown error'));
        }
    } catch (e) {
        alert('Error scaling: ' + e.message);
    }
}

// Restart deployment
export function showRestartModal(namespace, name) {
    currentResource = { type: 'deployment', namespace, name };
    document.getElementById('restart-deployment-name').textContent = name;
    showModal('restart-modal');
}

export async function confirmRestart() {
    const { namespace, name } = currentResource;

    try {
        const r = await fetch(`${API_BASE}/k8s/deployment/${namespace}/${name}/restart`, {
            method: 'POST'
        });
        const d = await r.json();
        if (d.success) {
            closeModal('restart-modal');
            loadK8sData();
        } else {
            alert('Error: ' + (d.error || 'Unknown error'));
        }
    } catch (e) {
        alert('Error restarting: ' + e.message);
    }
}

// Create Resource
export async function openCreateModal() {
    // Load namespaces into the dropdown
    const nsSelect = document.getElementById('create-namespace');
    try {
        const r = await fetch(`${API_BASE}/k8s/namespaces`);
        const d = await r.json();
        nsSelect.innerHTML = d.namespaces.map(ns =>
            `<option value="${ns.name}">${ns.name}</option>`
        ).join('');
    } catch (e) {
        console.error('Error loading namespaces:', e);
    }

    // Reset form
    document.getElementById('create-resource-type').value = '';
    document.getElementById('create-yaml-editor').value = '';
    showModal('create-modal');
}

export function loadResourceTemplate() {
    const type = document.getElementById('create-resource-type').value;
    const editor = document.getElementById('create-yaml-editor');

    if (type && resourceTemplates[type]) {
        editor.value = resourceTemplates[type];
    } else {
        editor.value = '';
    }
}

export async function createResource() {
    const type = document.getElementById('create-resource-type').value;
    const namespace = document.getElementById('create-namespace').value;
    const yaml = document.getElementById('create-yaml-editor').value;

    if (!type) {
        alert('Please select a resource type');
        return;
    }
    if (!yaml.trim()) {
        alert('Please provide resource YAML');
        return;
    }

    try {
        const r = await fetch(`${API_BASE}/k8s/create`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ type, namespace, yaml })
        });
        const d = await r.json();
        if (d.success) {
            closeModal('create-modal');
            loadK8sData();
            alert('Resource created successfully!');
        } else {
            alert('Error: ' + (d.error || 'Unknown error'));
        }
    } catch (e) {
        alert('Error creating resource: ' + e.message);
    }
}

// Auto Refresh
export function toggleAutoRefresh() {
    autoRefreshEnabled = !autoRefreshEnabled;
    const btn = document.getElementById('auto-refresh-btn');
    const label = document.getElementById('auto-refresh-label');

    if (autoRefreshEnabled) {
        btn.classList.add('active');
        label.textContent = 'Live';
        startAutoRefresh();
    } else {
        btn.classList.remove('active');
        label.textContent = 'Auto';
        stopAutoRefresh();
    }
}

function startAutoRefresh() {
    stopAutoRefresh(); // Clear any existing timer
    autoRefreshTimer = setInterval(() => {
        loadK8sData();
    }, refreshInterval * 1000);
}

function stopAutoRefresh() {
    if (autoRefreshTimer) {
        clearInterval(autoRefreshTimer);
        autoRefreshTimer = null;
    }
}

export function updateRefreshInterval() {
    refreshInterval = parseInt(document.getElementById('refresh-interval').value);
    if (autoRefreshEnabled) {
        startAutoRefresh(); // Restart with new interval
    }
}
