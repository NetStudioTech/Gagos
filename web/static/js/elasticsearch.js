// Elasticsearch Module for GAGOS

import { API_BASE } from './app.js';
import { escapeHtml, formatSize } from './utils.js';

let esConnected = false;
let esConfig = {};
let currentIndex = '';

export function showEsTab(tabId) {
    document.querySelectorAll('#window-elasticsearch .tab-btn').forEach(t => t.classList.remove('active'));
    document.querySelectorAll('#window-elasticsearch .tab-content').forEach(t => t.classList.remove('active'));
    document.querySelector(`#window-elasticsearch .tab-btn[onclick*="${tabId}"]`)?.classList.add('active');
    document.getElementById('es-tab-' + tabId)?.classList.add('active');
}

export function esLoadPreset() {
    const preset = document.getElementById('es-preset').value;
    const hostInput = document.getElementById('es-host');
    const portInput = document.getElementById('es-port');
    const sslCheckbox = document.getElementById('es-ssl');

    if (preset === 'local') {
        hostInput.value = 'localhost';
        portInput.value = '9200';
        sslCheckbox.checked = false;
    } else if (preset === 'docker') {
        hostInput.value = 'elasticsearch';
        portInput.value = '9200';
        sslCheckbox.checked = false;
    } else if (preset === 'cloud') {
        hostInput.value = '';
        portInput.value = '443';
        sslCheckbox.checked = true;
    }
}

export async function esConnect() {
    const host = document.getElementById('es-host').value.trim();
    const port = parseInt(document.getElementById('es-port').value) || 9200;
    const username = document.getElementById('es-username').value.trim();
    const password = document.getElementById('es-password').value;
    const useSSL = document.getElementById('es-ssl').checked;
    const statusEl = document.getElementById('es-conn-status');

    if (!host) {
        statusEl.innerHTML = '<span style="color:#ef4444;">Host is required</span>';
        return;
    }

    statusEl.innerHTML = '<span style="color:#60a5fa;">Connecting...</span>';

    esConfig = { host, port, username, password, use_ssl: useSSL };

    try {
        const resp = await fetch(`${API_BASE}/elasticsearch/connect`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(esConfig)
        });
        const data = await resp.json();

        if (data.success) {
            esConnected = true;
            statusEl.innerHTML = `<span style="color:#4ade80;">Connected to ${escapeHtml(data.cluster_name)} (v${data.version}) - ${data.response_time_ms.toFixed(0)}ms</span>`;
            esLoadClusterInfo();
            esLoadIndices();
        } else {
            esConnected = false;
            statusEl.innerHTML = `<span style="color:#ef4444;">Failed: ${escapeHtml(data.error)}</span>`;
        }
    } catch (e) {
        esConnected = false;
        statusEl.innerHTML = `<span style="color:#ef4444;">Error: ${escapeHtml(e.message)}</span>`;
    }
}

async function esLoadClusterInfo() {
    if (!esConnected) return;

    try {
        const [healthResp, statsResp, nodesResp] = await Promise.all([
            fetch(`${API_BASE}/elasticsearch/health`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(esConfig)
            }),
            fetch(`${API_BASE}/elasticsearch/stats`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(esConfig)
            }),
            fetch(`${API_BASE}/elasticsearch/nodes`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(esConfig)
            })
        ]);

        const health = await healthResp.json();
        const stats = await statsResp.json();
        const nodes = await nodesResp.json();

        renderClusterOverview(health, stats, nodes);
    } catch (e) {
        console.error('Failed to load cluster info:', e);
    }
}

function renderClusterOverview(health, stats, nodes) {
    const container = document.getElementById('es-cluster-overview');
    if (!container) return;

    const statusColor = health.status === 'green' ? '#4ade80' : health.status === 'yellow' ? '#fbbf24' : '#ef4444';

    let html = `
        <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:15px;margin-bottom:20px;">
            <div class="stat-card" style="background:#1e1e2e;padding:15px;border-radius:8px;border-left:4px solid ${statusColor};">
                <div style="color:#8a8a9a;font-size:12px;">Cluster Status</div>
                <div style="font-size:24px;font-weight:600;color:${statusColor};text-transform:uppercase;">${health.status}</div>
                <div style="color:#8a8a9a;font-size:11px;">${health.cluster_name}</div>
            </div>
            <div class="stat-card" style="background:#1e1e2e;padding:15px;border-radius:8px;">
                <div style="color:#8a8a9a;font-size:12px;">Nodes</div>
                <div style="font-size:24px;font-weight:600;color:#e4e4e7;">${health.number_of_nodes}</div>
                <div style="color:#8a8a9a;font-size:11px;">${health.number_of_data_nodes} data nodes</div>
            </div>
            <div class="stat-card" style="background:#1e1e2e;padding:15px;border-radius:8px;">
                <div style="color:#8a8a9a;font-size:12px;">Indices</div>
                <div style="font-size:24px;font-weight:600;color:#e4e4e7;">${stats.indices?.count || 0}</div>
                <div style="color:#8a8a9a;font-size:11px;">${formatSize(stats.indices?.store?.size_in_bytes || 0)}</div>
            </div>
            <div class="stat-card" style="background:#1e1e2e;padding:15px;border-radius:8px;">
                <div style="color:#8a8a9a;font-size:12px;">Documents</div>
                <div style="font-size:24px;font-weight:600;color:#e4e4e7;">${(stats.indices?.docs?.count || 0).toLocaleString()}</div>
                <div style="color:#8a8a9a;font-size:11px;">${(stats.indices?.docs?.deleted || 0).toLocaleString()} deleted</div>
            </div>
            <div class="stat-card" style="background:#1e1e2e;padding:15px;border-radius:8px;">
                <div style="color:#8a8a9a;font-size:12px;">Shards</div>
                <div style="font-size:24px;font-weight:600;color:#e4e4e7;">${health.active_shards}</div>
                <div style="color:#8a8a9a;font-size:11px;">${health.unassigned_shards} unassigned</div>
            </div>
            <div class="stat-card" style="background:#1e1e2e;padding:15px;border-radius:8px;">
                <div style="color:#8a8a9a;font-size:12px;">Active Shards %</div>
                <div style="font-size:24px;font-weight:600;color:#e4e4e7;">${(health.active_shards_percent_as_number || 0).toFixed(1)}%</div>
                <div style="color:#8a8a9a;font-size:11px;">${health.relocating_shards} relocating</div>
            </div>
        </div>
    `;

    // Nodes table
    if (Array.isArray(nodes) && nodes.length > 0) {
        html += `
            <h4 style="color:#e4e4e7;margin-bottom:10px;">Nodes</h4>
            <div class="table-container" style="max-height:200px;overflow:auto;">
                <table class="data-table">
                    <thead>
                        <tr>
                            <th>Name</th>
                            <th>IP</th>
                            <th>Role</th>
                            <th>Master</th>
                            <th>Heap %</th>
                            <th>RAM %</th>
                            <th>CPU</th>
                            <th>Load</th>
                        </tr>
                    </thead>
                    <tbody>
        `;
        for (const node of nodes) {
            html += `
                <tr>
                    <td>${escapeHtml(node.name || '-')}</td>
                    <td style="font-family:monospace;">${escapeHtml(node.ip || '-')}</td>
                    <td>${escapeHtml(node['node.role'] || '-')}</td>
                    <td>${node.master === '*' ? '<span style="color:#4ade80;">Yes</span>' : 'No'}</td>
                    <td>${escapeHtml(node['heap.percent'] || '-')}%</td>
                    <td>${escapeHtml(node['ram.percent'] || '-')}%</td>
                    <td>${escapeHtml(node.cpu || '-')}</td>
                    <td>${escapeHtml(node.load_1m || '-')}</td>
                </tr>
            `;
        }
        html += '</tbody></table></div>';
    }

    container.innerHTML = html;
}

export async function esLoadIndices() {
    if (!esConnected) return;

    const container = document.getElementById('es-indices-list');
    container.innerHTML = '<div style="color:#60a5fa;padding:20px;">Loading indices...</div>';

    try {
        const resp = await fetch(`${API_BASE}/elasticsearch/indices`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(esConfig)
        });
        const indices = await resp.json();

        if (indices.error) {
            container.innerHTML = `<div style="color:#ef4444;padding:20px;">Error: ${escapeHtml(indices.error)}</div>`;
            return;
        }

        if (!Array.isArray(indices) || indices.length === 0) {
            container.innerHTML = '<div style="color:#8a8a9a;padding:20px;">No indices found</div>';
            return;
        }

        // Sort by index name
        indices.sort((a, b) => (a.index || '').localeCompare(b.index || ''));

        let html = `
            <table class="data-table">
                <thead>
                    <tr>
                        <th>Index</th>
                        <th>Health</th>
                        <th>Status</th>
                        <th>Pri</th>
                        <th>Rep</th>
                        <th>Docs</th>
                        <th>Size</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
        `;

        for (const idx of indices) {
            const healthColor = idx.health === 'green' ? '#4ade80' : idx.health === 'yellow' ? '#fbbf24' : '#ef4444';
            html += `
                <tr>
                    <td>
                        <a href="javascript:void(0)" onclick="esSelectIndex('${escapeHtml(idx.index)}')" style="color:#60a5fa;">
                            ${escapeHtml(idx.index)}
                        </a>
                    </td>
                    <td><span style="color:${healthColor};">${escapeHtml(idx.health || '-')}</span></td>
                    <td>${escapeHtml(idx.status || '-')}</td>
                    <td>${escapeHtml(idx.pri || '-')}</td>
                    <td>${escapeHtml(idx.rep || '-')}</td>
                    <td>${escapeHtml(idx['docs.count'] || '0')}</td>
                    <td>${escapeHtml(idx['store.size'] || '-')}</td>
                    <td>
                        <button class="action-btn small" onclick="esViewMapping('${escapeHtml(idx.index)}')" title="View Mapping">M</button>
                        <button class="action-btn small" onclick="esViewSettings('${escapeHtml(idx.index)}')" title="View Settings">S</button>
                        <button class="action-btn small" onclick="esRefreshIndex('${escapeHtml(idx.index)}')" title="Refresh">R</button>
                        <button class="action-btn small danger" onclick="esDeleteIndex('${escapeHtml(idx.index)}')" title="Delete">X</button>
                    </td>
                </tr>
            `;
        }

        html += '</tbody></table>';
        container.innerHTML = html;
    } catch (e) {
        container.innerHTML = `<div style="color:#ef4444;padding:20px;">Error: ${escapeHtml(e.message)}</div>`;
    }
}

export async function esCreateIndex() {
    const indexName = prompt('Enter new index name:');
    if (!indexName) return;

    try {
        const resp = await fetch(`${API_BASE}/elasticsearch/index/create`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...esConfig, index: indexName })
        });
        const data = await resp.json();

        if (data.success) {
            alert('Index created successfully');
            esLoadIndices();
        } else {
            alert('Failed to create index: ' + data.error);
        }
    } catch (e) {
        alert('Error: ' + e.message);
    }
}

export async function esDeleteIndex(index) {
    if (!confirm(`Are you sure you want to delete index "${index}"? This cannot be undone.`)) return;

    try {
        const resp = await fetch(`${API_BASE}/elasticsearch/index/delete`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...esConfig, index })
        });
        const data = await resp.json();

        if (data.success) {
            alert('Index deleted successfully');
            esLoadIndices();
            if (currentIndex === index) {
                currentIndex = '';
                document.getElementById('es-current-index').textContent = '-';
            }
        } else {
            alert('Failed to delete index: ' + data.error);
        }
    } catch (e) {
        alert('Error: ' + e.message);
    }
}

export async function esRefreshIndex(index) {
    try {
        const resp = await fetch(`${API_BASE}/elasticsearch/index/refresh`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...esConfig, index })
        });
        const data = await resp.json();

        if (data.success) {
            alert('Index refreshed successfully');
        } else {
            alert('Failed to refresh index: ' + data.error);
        }
    } catch (e) {
        alert('Error: ' + e.message);
    }
}

export async function esViewMapping(index) {
    try {
        const resp = await fetch(`${API_BASE}/elasticsearch/index/mapping`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...esConfig, index })
        });
        const data = await resp.json();

        showEsModal('Index Mapping: ' + index, JSON.stringify(data, null, 2));
    } catch (e) {
        alert('Error: ' + e.message);
    }
}

export async function esViewSettings(index) {
    try {
        const resp = await fetch(`${API_BASE}/elasticsearch/index/settings`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...esConfig, index })
        });
        const data = await resp.json();

        showEsModal('Index Settings: ' + index, JSON.stringify(data, null, 2));
    } catch (e) {
        alert('Error: ' + e.message);
    }
}

export function esSelectIndex(index) {
    currentIndex = index;
    document.getElementById('es-current-index').textContent = index;
    document.getElementById('es-search-index').value = index;
    showEsTab('documents');
    esSearchDocuments();
}

export async function esSearchDocuments() {
    const index = document.getElementById('es-search-index').value.trim();
    const query = document.getElementById('es-search-query').value.trim();
    const container = document.getElementById('es-documents-list');

    if (!index) {
        container.innerHTML = '<div style="color:#8a8a9a;padding:20px;">Select an index first</div>';
        return;
    }

    container.innerHTML = '<div style="color:#60a5fa;padding:20px;">Searching...</div>';

    try {
        const resp = await fetch(`${API_BASE}/elasticsearch/search`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...esConfig, index, query, from: 0, size: 50 })
        });
        const data = await resp.json();

        if (data.error) {
            container.innerHTML = `<div style="color:#ef4444;padding:20px;">Error: ${escapeHtml(data.error)}</div>`;
            return;
        }

        const hits = data.hits?.hits || [];
        const total = data.hits?.total?.value || 0;

        if (hits.length === 0) {
            container.innerHTML = '<div style="color:#8a8a9a;padding:20px;">No documents found</div>';
            return;
        }

        let html = `<div style="color:#8a8a9a;font-size:12px;margin-bottom:10px;">Found ${total.toLocaleString()} documents (showing ${hits.length})</div>`;
        html += '<div class="es-documents">';

        for (const hit of hits) {
            html += `
                <div class="es-doc-card" style="background:#1e1e2e;padding:12px;border-radius:6px;margin-bottom:8px;">
                    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:8px;">
                        <div style="font-family:monospace;color:#60a5fa;font-size:12px;">
                            ${escapeHtml(hit._id)}
                        </div>
                        <div>
                            <button class="action-btn small" onclick="esViewDocument('${escapeHtml(index)}', '${escapeHtml(hit._id)}')" title="View">View</button>
                            <button class="action-btn small danger" onclick="esDeleteDocument('${escapeHtml(index)}', '${escapeHtml(hit._id)}')" title="Delete">Del</button>
                        </div>
                    </div>
                    <pre style="background:#0d0d14;padding:8px;border-radius:4px;overflow:auto;max-height:150px;font-size:11px;color:#a8a8b8;margin:0;">${escapeHtml(JSON.stringify(hit._source, null, 2))}</pre>
                </div>
            `;
        }

        html += '</div>';
        container.innerHTML = html;
    } catch (e) {
        container.innerHTML = `<div style="color:#ef4444;padding:20px;">Error: ${escapeHtml(e.message)}</div>`;
    }
}

export async function esViewDocument(index, id) {
    try {
        const resp = await fetch(`${API_BASE}/elasticsearch/document`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...esConfig, index, id })
        });
        const data = await resp.json();

        showEsModal('Document: ' + id, JSON.stringify(data, null, 2));
    } catch (e) {
        alert('Error: ' + e.message);
    }
}

export async function esDeleteDocument(index, id) {
    if (!confirm(`Delete document "${id}"?`)) return;

    try {
        const resp = await fetch(`${API_BASE}/elasticsearch/document/delete`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...esConfig, index, id })
        });
        const data = await resp.json();

        if (data.success) {
            esSearchDocuments();
        } else {
            alert('Failed to delete document: ' + data.error);
        }
    } catch (e) {
        alert('Error: ' + e.message);
    }
}

export async function esExecuteQuery() {
    const method = document.getElementById('es-query-method').value;
    const path = document.getElementById('es-query-path').value.trim();
    const body = document.getElementById('es-query-body').value.trim();
    const resultEl = document.getElementById('es-query-result');

    resultEl.textContent = 'Executing...';

    try {
        let bodyJson = null;
        if (body) {
            bodyJson = JSON.parse(body);
        }

        const resp = await fetch(`${API_BASE}/elasticsearch/query`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...esConfig, method, path, body: bodyJson })
        });
        const data = await resp.json();

        if (data.error) {
            resultEl.textContent = 'Error: ' + data.error;
        } else {
            let result = data.body;
            try {
                result = JSON.stringify(JSON.parse(data.body), null, 2);
            } catch (e) {
                // Keep as is
            }
            resultEl.textContent = `Status: ${data.status_code}\n\n${result}`;
        }
    } catch (e) {
        resultEl.textContent = 'Error: ' + e.message;
    }
}

function showEsModal(title, content) {
    const modal = document.getElementById('es-modal');
    document.getElementById('es-modal-title').textContent = title;
    document.getElementById('es-modal-content').textContent = content;
    modal.style.display = 'flex';
}

export function esCloseModal() {
    document.getElementById('es-modal').style.display = 'none';
}

export function esCopyModalContent() {
    const content = document.getElementById('es-modal-content').textContent;
    window._copyText(content);
    alert('Copied to clipboard');
}
