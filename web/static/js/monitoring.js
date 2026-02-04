// Monitoring Module for GAGOS

import { API_BASE } from './app.js';
import { saveState } from './state.js';

export function showMonitoringTab(tabId) {
    document.querySelectorAll('#window-monitoring .tab-content').forEach(t => t.classList.remove('active'));
    document.querySelectorAll('#window-monitoring .tab-btn').forEach(b => b.classList.remove('active'));
    document.getElementById('monitoring-tab-' + tabId).classList.add('active');
    event.target.classList.add('active');

    if (tabId === 'overview') loadMonitoringSummary();
    else if (tabId === 'nodes') loadMonitoringNodes();
    else if (tabId === 'pods') loadMonitoringPods();
    else if (tabId === 'quotas') loadMonitoringQuotas();
    else if (tabId === 'hpa') loadMonitoringHPA();

    saveState();
}

export async function loadMonitoringData() {
    await loadMonitoringSummary();
    await populateMonitoringNamespaces();
}

export async function populateMonitoringNamespaces() {
    try {
        const r = await fetch(`${API_BASE}/k8s/namespaces`);
        const d = await r.json();
        if (d.namespaces) {
            const selects = ['mon-pods-ns', 'mon-quotas-ns', 'mon-hpa-ns'];
            selects.forEach(id => {
                const sel = document.getElementById(id);
                if (!sel) return;
                sel.innerHTML = '<option value="">All Namespaces</option>';
                d.namespaces.forEach(ns => {
                    sel.innerHTML += `<option value="${ns.name}">${ns.name}</option>`;
                });
            });
        }
    } catch (e) {
        console.error('Failed to load namespaces:', e);
    }
}

export async function loadMonitoringSummary() {
    try {
        const r = await fetch(`${API_BASE}/monitoring/summary`);
        const d = await r.json();
        if (d.error) {
            console.error(d.error);
            return;
        }

        document.getElementById('mon-stat-nodes').textContent = `${d.ready_nodes}/${d.total_nodes}`;
        document.getElementById('mon-stat-pods').textContent = `${d.running_pods}/${d.total_pods}`;
        document.getElementById('mon-stat-cpu').textContent = d.cpu_percent.toFixed(1) + '%';
        document.getElementById('mon-stat-mem').textContent = d.memory_percent.toFixed(1) + '%';

        // Update progress bars
        const cpuBar = document.getElementById('mon-cpu-bar');
        const memBar = document.getElementById('mon-mem-bar');
        cpuBar.style.width = Math.min(d.cpu_percent, 100) + '%';
        memBar.style.width = Math.min(d.memory_percent, 100) + '%';

        // Color based on usage
        cpuBar.style.background = getUsageGradient(d.cpu_percent);
        memBar.style.background = getUsageGradient(d.memory_percent);

        // Text
        const cpuCores = (d.total_cpu_millicores / 1000).toFixed(1);
        const cpuUsed = (d.used_cpu_millicores / 1000).toFixed(1);
        const memGB = (d.total_memory_bytes / (1024*1024*1024)).toFixed(1);
        const memUsed = (d.used_memory_bytes / (1024*1024*1024)).toFixed(1);
        document.getElementById('mon-cpu-text').textContent = `${cpuUsed} / ${cpuCores} cores`;
        document.getElementById('mon-mem-text').textContent = `${memUsed} / ${memGB} GB`;

        // Show notice if no metrics
        const notice = document.getElementById('mon-metrics-notice');
        if (d.used_cpu_millicores === 0 && d.used_memory_bytes === 0) {
            notice.style.display = 'block';
        } else {
            notice.style.display = 'none';
        }
    } catch (e) {
        console.error('Failed to load monitoring summary:', e);
    }
}

function getUsageGradient(percent) {
    if (percent < 50) return 'linear-gradient(90deg, #4ade80, #22c55e)';
    if (percent < 80) return 'linear-gradient(90deg, #fbbf24, #f59e0b)';
    return 'linear-gradient(90deg, #ef4444, #dc2626)';
}

export async function loadMonitoringNodes() {
    try {
        const r = await fetch(`${API_BASE}/monitoring/nodes`);
        const d = await r.json();
        const tbody = document.getElementById('mon-nodes-tbody');
        tbody.innerHTML = '';

        if (!d.nodes || d.nodes.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" style="text-align:center;color:#8a8a9a;">No nodes found</td></tr>';
            return;
        }

        d.nodes.forEach(n => {
            const cpuPct = n.cpu_percent.toFixed(1);
            const memPct = n.memory_percent.toFixed(1);
            const cpuColor = getUsageColor(n.cpu_percent);
            const memColor = getUsageColor(n.memory_percent);

            tbody.innerHTML += `
                <tr>
                    <td style="font-weight:500;">${n.name}</td>
                    <td><span style="background:rgba(102,126,234,0.2);padding:2px 8px;border-radius:4px;font-size:12px;">${n.roles ? n.roles.join(', ') : 'worker'}</span></td>
                    <td>
                        <div style="display:flex;align-items:center;gap:8px;">
                            <div style="flex:1;max-width:100px;background:#1e1e2e;border-radius:6px;height:16px;overflow:hidden;">
                                <div style="height:100%;width:${Math.min(n.cpu_percent,100)}%;background:${cpuColor};"></div>
                            </div>
                            <span style="font-size:12px;color:#a0a0b0;">${cpuPct}%</span>
                        </div>
                    </td>
                    <td>
                        <div style="display:flex;align-items:center;gap:8px;">
                            <div style="flex:1;max-width:100px;background:#1e1e2e;border-radius:6px;height:16px;overflow:hidden;">
                                <div style="height:100%;width:${Math.min(n.memory_percent,100)}%;background:${memColor};"></div>
                            </div>
                            <span style="font-size:12px;color:#a0a0b0;">${memPct}%</span>
                        </div>
                    </td>
                    <td>${n.pod_count}/${n.pod_capacity}</td>
                    <td>${n.conditions ? n.conditions.map(c => `<span style="background:rgba(74,222,128,0.2);color:#4ade80;padding:2px 6px;border-radius:4px;font-size:11px;margin-right:4px;">${c}</span>`).join('') : '-'}</td>
                </tr>
            `;
        });
    } catch (e) {
        console.error('Failed to load nodes:', e);
    }
}

function getUsageColor(percent) {
    if (percent < 50) return '#4ade80';
    if (percent < 80) return '#fbbf24';
    return '#ef4444';
}

export async function loadMonitoringPods() {
    const ns = document.getElementById('mon-pods-ns').value;
    try {
        const url = ns ? `${API_BASE}/monitoring/pods/${ns}` : `${API_BASE}/monitoring/pods`;
        const r = await fetch(url);
        const d = await r.json();
        const tbody = document.getElementById('mon-pods-tbody');
        tbody.innerHTML = '';

        if (!d.pods || d.pods.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" style="text-align:center;color:#8a8a9a;">No running pods found</td></tr>';
            return;
        }

        d.pods.forEach(p => {
            const cpuMilli = p.cpu_usage_millicores || 0;
            const memBytes = p.memory_usage_bytes || 0;
            const memMB = (memBytes / (1024*1024)).toFixed(1);

            tbody.innerHTML += `
                <tr>
                    <td style="font-weight:500;">${p.name}</td>
                    <td><span style="background:rgba(102,126,234,0.2);padding:2px 8px;border-radius:4px;font-size:12px;">${p.namespace}</span></td>
                    <td>${cpuMilli}m</td>
                    <td>${memMB} MB</td>
                    <td>${p.node || '-'}</td>
                    <td><span class="status-badge ${p.status === 'Running' ? 'running' : 'pending'}">${p.status}</span></td>
                </tr>
            `;
        });
    } catch (e) {
        console.error('Failed to load pods:', e);
    }
}

export async function loadMonitoringQuotas() {
    const ns = document.getElementById('mon-quotas-ns').value;
    try {
        const url = ns ? `${API_BASE}/monitoring/quotas/${ns}` : `${API_BASE}/monitoring/quotas`;
        const r = await fetch(url);
        const d = await r.json();
        const container = document.getElementById('mon-quotas-container');

        if (!d.quotas || d.quotas.length === 0) {
            container.innerHTML = '<div id="mon-quotas-empty" style="text-align:center;color:#8a8a9a;padding:40px;">No resource quotas found</div>';
            return;
        }

        let html = '';
        d.quotas.forEach(q => {
            html += `
                <div style="background:#2a2a3e;border-radius:8px;padding:15px;margin-bottom:15px;">
                    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:12px;">
                        <h4 style="color:#e0e0e0;font-size:14px;">${q.name}</h4>
                        <span style="background:rgba(102,126,234,0.2);padding:2px 8px;border-radius:4px;font-size:12px;color:#667eea;">${q.namespace}</span>
                    </div>
                    <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:12px;">
            `;

            if (q.usage) {
                q.usage.forEach(u => {
                    const statusColor = u.status === 'critical' ? '#ef4444' : (u.status === 'warning' ? '#fbbf24' : '#4ade80');
                    html += `
                        <div style="background:#1e1e2e;padding:10px;border-radius:6px;">
                            <div style="display:flex;justify-content:space-between;margin-bottom:6px;">
                                <span style="color:#8a8a9a;font-size:12px;">${u.resource}</span>
                                <span style="color:${statusColor};font-size:12px;">${u.percent.toFixed(1)}%</span>
                            </div>
                            <div style="background:#0d0d14;border-radius:4px;height:8px;overflow:hidden;">
                                <div style="height:100%;width:${Math.min(u.percent,100)}%;background:${statusColor};"></div>
                            </div>
                            <div style="display:flex;justify-content:space-between;margin-top:6px;">
                                <span style="color:#606070;font-size:11px;">Used: ${u.used}</span>
                                <span style="color:#606070;font-size:11px;">Hard: ${u.hard}</span>
                            </div>
                        </div>
                    `;
                });
            }

            html += '</div></div>';
        });

        container.innerHTML = html;
    } catch (e) {
        console.error('Failed to load quotas:', e);
    }
}

export async function loadMonitoringHPA() {
    const ns = document.getElementById('mon-hpa-ns').value;
    try {
        const url = ns ? `${API_BASE}/monitoring/hpa/${ns}` : `${API_BASE}/monitoring/hpa`;
        const r = await fetch(url);
        const d = await r.json();
        const tbody = document.getElementById('mon-hpa-tbody');
        tbody.innerHTML = '';

        if (!d.hpas || d.hpas.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;color:#8a8a9a;">No HPAs found</td></tr>';
            return;
        }

        d.hpas.forEach(h => {
            const cpuText = h.current_cpu_percent != null && h.target_cpu_percent != null
                ? `${h.current_cpu_percent}% / ${h.target_cpu_percent}%`
                : '-';

            tbody.innerHTML += `
                <tr>
                    <td style="font-weight:500;">${h.name}</td>
                    <td><span style="background:rgba(102,126,234,0.2);padding:2px 8px;border-radius:4px;font-size:12px;">${h.namespace}</span></td>
                    <td>${h.target_kind}/${h.target_name}</td>
                    <td>${h.min_replicas}-${h.max_replicas}</td>
                    <td>
                        <span style="background:rgba(74,222,128,0.2);color:#4ade80;padding:2px 8px;border-radius:4px;">
                            ${h.current_replicas}${h.desired_replicas !== h.current_replicas ? ` -> ${h.desired_replicas}` : ''}
                        </span>
                    </td>
                    <td>${cpuText}</td>
                    <td style="color:#8a8a9a;font-size:12px;">${h.age || '-'}</td>
                </tr>
            `;
        });
    } catch (e) {
        console.error('Failed to load HPAs:', e);
    }
}
