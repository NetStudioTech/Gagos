// Database Module for GAGOS (PostgreSQL, Redis, MySQL)

import { API_BASE } from './app.js';
import { saveState } from './state.js';
import { escapeHtml } from './utils.js';

// ========== PostgreSQL Functions ==========

let pgConnected = false;
let pgConfig = {};

export function showPgTab(tabId) {
    document.querySelectorAll('#window-postgres .tab-content').forEach(t => t.classList.remove('active'));
    document.querySelectorAll('#window-postgres .tab-btn').forEach(b => b.classList.remove('active'));
    document.getElementById('pg-tab-' + tabId).classList.add('active');
    event.target.classList.add('active');
    saveState();
}

export async function pgConnect() {
    const btn = document.querySelector('#window-postgres .conn-form button');
    const status = document.getElementById('pg-conn-status');

    pgConfig = {
        host: document.getElementById('pg-host').value || 'localhost',
        port: parseInt(document.getElementById('pg-port').value) || 5432,
        user: document.getElementById('pg-user').value || 'postgres',
        password: document.getElementById('pg-password').value || '',
        database: document.getElementById('pg-database').value || 'postgres',
        ssl_mode: document.getElementById('pg-sslmode').value || 'disable'
    };

    btn.disabled = true;
    btn.textContent = 'Connecting...';
    status.innerHTML = '<span style="color:#60a5fa;">Testing connection...</span>';

    try {
        const r = await fetch(`${API_BASE}/db/postgres/connect`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(pgConfig)
        });
        const d = await r.json();

        if (d.success) {
            pgConnected = true;
            status.innerHTML = `<span style="color:#4ade80;">Connected to PostgreSQL ${d.version ? '(' + d.version.split(' ')[0] + ' ' + d.version.split(' ')[1] + ')' : ''}</span>`;
            btn.textContent = 'Reconnect';
            pgLoadInfo();
            pgLoadDatabases();
        } else {
            pgConnected = false;
            status.innerHTML = `<span style="color:#ef4444;">Failed: ${d.error}</span>`;
            btn.textContent = 'Connect';
        }
    } catch (e) {
        status.innerHTML = `<span style="color:#ef4444;">Error: ${e.message}</span>`;
        btn.textContent = 'Connect';
    }
    btn.disabled = false;
}

export async function pgLoadInfo() {
    if (!pgConnected) return;
    const container = document.getElementById('pg-info-content');
    container.innerHTML = '<div style="color:#60a5fa;">Loading...</div>';

    try {
        const r = await fetch(`${API_BASE}/db/postgres/info`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(pgConfig)
        });
        const d = await r.json();

        if (d.error) {
            container.innerHTML = `<div style="color:#ef4444;">${d.error}</div>`;
            return;
        }

        let tablesHtml = '';
        if (d.tables && d.tables.length > 0) {
            tablesHtml = `
                <div class="info-section">
                    <h4>Tables (Top ${d.tables.length})</h4>
                    <table class="db-table">
                        <thead><tr><th>Schema</th><th>Name</th><th>Rows</th><th>Size</th><th>Index Size</th></tr></thead>
                        <tbody>
                            ${d.tables.map(t => `<tr><td>${t.schema}</td><td>${t.name}</td><td>${t.row_count.toLocaleString()}</td><td>${t.size}</td><td>${t.index_size}</td></tr>`).join('')}
                        </tbody>
                    </table>
                </div>
            `;
        }

        container.innerHTML = `
            <div class="db-stats-grid">
                <div class="db-stat-card">
                    <div class="stat-label">Version</div>
                    <div class="stat-value" style="font-size:12px;">${d.version ? d.version.split(',')[0] : 'N/A'}</div>
                </div>
                <div class="db-stat-card">
                    <div class="stat-label">Database Size</div>
                    <div class="stat-value">${d.database_size || 'N/A'}</div>
                </div>
                <div class="db-stat-card">
                    <div class="stat-label">Tables</div>
                    <div class="stat-value">${d.table_count || 0}</div>
                </div>
                <div class="db-stat-card">
                    <div class="stat-label">Connections</div>
                    <div class="stat-value">${d.active_connections || 0} / ${d.max_connections || 0}</div>
                </div>
                <div class="db-stat-card">
                    <div class="stat-label">Uptime</div>
                    <div class="stat-value" style="font-size:12px;">${d.uptime || 'N/A'}</div>
                </div>
            </div>
            ${tablesHtml}
        `;
    } catch (e) {
        container.innerHTML = `<div style="color:#ef4444;">Error: ${e.message}</div>`;
    }
}

export async function pgLoadDatabases() {
    if (!pgConnected) return;
    try {
        const r = await fetch(`${API_BASE}/db/postgres/databases`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(pgConfig)
        });
        const d = await r.json();
        if (d.databases) {
            const select = document.getElementById('pg-database');
            const current = select.value;
            // Keep current value, just for reference
        }
    } catch (e) {
        console.error('Failed to load databases:', e);
    }
}

export async function pgExecuteQuery() {
    if (!pgConnected) {
        alert('Please connect first');
        return;
    }
    const query = document.getElementById('pg-query-input').value.trim();
    if (!query) return;

    const readonly = document.getElementById('pg-readonly').checked;
    const output = document.getElementById('pg-query-output');
    output.innerHTML = '<div style="color:#60a5fa;">Executing...</div>';

    try {
        const r = await fetch(`${API_BASE}/db/postgres/query`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...pgConfig, query, readonly })
        });
        const d = await r.json();

        if (d.error) {
            output.innerHTML = `<div style="color:#ef4444;">Error: ${d.error}</div><div style="color:#6b7280;font-size:11px;margin-top:4px;">Duration: ${d.duration_ms?.toFixed(2) || 0}ms</div>`;
            return;
        }

        if (d.columns && d.rows) {
            output.innerHTML = `
                <div style="color:#6b7280;font-size:11px;margin-bottom:8px;">${d.rows.length} row(s) in ${d.duration_ms?.toFixed(2) || 0}ms</div>
                <table class="db-table">
                    <thead><tr>${d.columns.map(c => `<th>${escapeHtml(c)}</th>`).join('')}</tr></thead>
                    <tbody>${d.rows.map(row => `<tr>${row.map(v => `<td>${v === null ? '<span style="color:#6b7280;">NULL</span>' : escapeHtml(String(v))}</td>`).join('')}</tr>`).join('')}</tbody>
                </table>
            `;
        } else {
            output.innerHTML = `<div style="color:#4ade80;">${d.rows_affected} row(s) affected</div><div style="color:#6b7280;font-size:11px;margin-top:4px;">Duration: ${d.duration_ms?.toFixed(2) || 0}ms</div>`;
        }
    } catch (e) {
        output.innerHTML = `<div style="color:#ef4444;">Error: ${e.message}</div>`;
    }
}

export async function pgDump() {
    if (!pgConnected) {
        alert('Please connect first');
        return;
    }
    const schemaOnly = document.getElementById('pg-dump-schema').checked;
    const dataOnly = document.getElementById('pg-dump-data').checked;
    const tables = document.getElementById('pg-dump-tables').value.split(',').map(t => t.trim()).filter(t => t);
    const output = document.getElementById('pg-dump-output');
    output.value = 'Generating dump...';

    try {
        const r = await fetch(`${API_BASE}/db/postgres/dump`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...pgConfig, schema_only: schemaOnly, data_only: dataOnly, tables })
        });
        const d = await r.json();

        if (d.error) {
            output.value = 'Error: ' + d.error;
        } else {
            output.value = d.output || 'No output';
        }
    } catch (e) {
        output.value = 'Error: ' + e.message;
    }
}

export function pgCopyDump() {
    const output = document.getElementById('pg-dump-output');
    window._copyText(output.value).then(() => {
        const btn = event.target;
        const orig = btn.textContent;
        btn.textContent = 'Copied!';
        setTimeout(() => btn.textContent = orig, 1500);
    });
}

export function pgDownloadDump() {
    const output = document.getElementById('pg-dump-output').value;
    if (!output || output.startsWith('Error') || output === 'Generating dump...') return;
    const blob = new Blob([output], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${pgConfig.database || 'dump'}_${new Date().toISOString().split('T')[0]}.sql`;
    a.click();
    URL.revokeObjectURL(url);
}

// ========== Redis Functions ==========

let redisConnected = false;
let redisConfig = {};

export function showRedisTab(tabId) {
    document.querySelectorAll('#window-redis .tab-content').forEach(t => t.classList.remove('active'));
    document.querySelectorAll('#window-redis .tab-btn').forEach(b => b.classList.remove('active'));
    document.getElementById('redis-tab-' + tabId).classList.add('active');
    event.target.classList.add('active');
    saveState();
}

export async function redisConnect() {
    const btn = document.querySelector('#window-redis .conn-form button');
    const status = document.getElementById('redis-conn-status');

    redisConfig = {
        host: document.getElementById('redis-host').value || 'localhost',
        port: parseInt(document.getElementById('redis-port').value) || 6379,
        password: document.getElementById('redis-password').value || '',
        db: parseInt(document.getElementById('redis-db').value) || 0,
        use_tls: document.getElementById('redis-tls').checked
    };

    btn.disabled = true;
    btn.textContent = 'Connecting...';
    status.innerHTML = '<span style="color:#60a5fa;">Testing connection...</span>';

    try {
        const r = await fetch(`${API_BASE}/db/redis/connect`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(redisConfig)
        });
        const d = await r.json();

        if (d.success) {
            redisConnected = true;
            status.innerHTML = `<span style="color:#4ade80;">Connected to Redis ${d.version || ''} (${d.mode || 'standalone'})</span>`;
            btn.textContent = 'Reconnect';
            redisLoadInfo();
        } else {
            redisConnected = false;
            status.innerHTML = `<span style="color:#ef4444;">Failed: ${d.error}</span>`;
            btn.textContent = 'Connect';
        }
    } catch (e) {
        status.innerHTML = `<span style="color:#ef4444;">Error: ${e.message}</span>`;
        btn.textContent = 'Connect';
    }
    btn.disabled = false;
}

export async function redisLoadInfo() {
    if (!redisConnected) return;
    const container = document.getElementById('redis-info-content');
    container.innerHTML = '<div style="color:#60a5fa;">Loading...</div>';

    try {
        const r = await fetch(`${API_BASE}/db/redis/info`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(redisConfig)
        });
        const d = await r.json();

        if (d.error) {
            container.innerHTML = `<div style="color:#ef4444;">${d.error}</div>`;
            return;
        }

        container.innerHTML = `
            <div class="db-stats-grid">
                <div class="db-stat-card">
                    <div class="stat-label">Version</div>
                    <div class="stat-value">${d.version || 'N/A'}</div>
                </div>
                <div class="db-stat-card">
                    <div class="stat-label">Mode</div>
                    <div class="stat-value">${d.mode || 'standalone'}</div>
                </div>
                <div class="db-stat-card">
                    <div class="stat-label">Connected Clients</div>
                    <div class="stat-value">${d.connected_clients || 0}</div>
                </div>
                <div class="db-stat-card">
                    <div class="stat-label">Used Memory</div>
                    <div class="stat-value">${d.used_memory_human || 'N/A'}</div>
                </div>
                <div class="db-stat-card">
                    <div class="stat-label">Total Keys</div>
                    <div class="stat-value">${(d.total_keys || 0).toLocaleString()}</div>
                </div>
                <div class="db-stat-card">
                    <div class="stat-label">Uptime</div>
                    <div class="stat-value">${d.uptime_human || 'N/A'}</div>
                </div>
                <div class="db-stat-card">
                    <div class="stat-label">Ops/sec</div>
                    <div class="stat-value">${d.ops_per_sec || 0}</div>
                </div>
                <div class="db-stat-card">
                    <div class="stat-label">Hit Rate</div>
                    <div class="stat-value">${d.hit_rate ? d.hit_rate.toFixed(1) + '%' : 'N/A'}</div>
                </div>
            </div>
            ${d.databases && d.databases.length ? `
                <div class="info-section">
                    <h4>Databases</h4>
                    <div style="display:flex;flex-wrap:wrap;gap:8px;">
                        ${d.databases.map(db => `<div class="db-stat-card" style="min-width:100px;"><div class="stat-label">DB ${db.index}</div><div class="stat-value">${db.keys} keys</div></div>`).join('')}
                    </div>
                </div>
            ` : ''}
        `;
    } catch (e) {
        container.innerHTML = `<div style="color:#ef4444;">Error: ${e.message}</div>`;
    }
}

export async function redisScanKeys() {
    if (!redisConnected) {
        alert('Please connect first');
        return;
    }
    const pattern = document.getElementById('redis-key-pattern').value || '*';
    const output = document.getElementById('redis-keys-output');
    output.innerHTML = '<div style="color:#60a5fa;">Scanning...</div>';

    try {
        const r = await fetch(`${API_BASE}/db/redis/scan`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...redisConfig, pattern, count: 100 })
        });
        const d = await r.json();

        if (d.error) {
            output.innerHTML = `<div style="color:#ef4444;">${d.error}</div>`;
            return;
        }

        if (!d.keys || d.keys.length === 0) {
            output.innerHTML = '<div style="color:#6b7280;">No keys found</div>';
            return;
        }

        output.innerHTML = `
            <div style="color:#6b7280;font-size:11px;margin-bottom:8px;">Found ${d.keys.length} key(s)</div>
            <div class="redis-keys-list">
                ${d.keys.map(k => `<div class="redis-key-item" onclick="window.redisGetKey('${escapeHtml(k)}')">${escapeHtml(k)}</div>`).join('')}
            </div>
        `;
    } catch (e) {
        output.innerHTML = `<div style="color:#ef4444;">Error: ${e.message}</div>`;
    }
}

export async function redisGetKey(key) {
    const output = document.getElementById('redis-key-value');
    output.innerHTML = '<div style="color:#60a5fa;">Loading...</div>';

    try {
        const r = await fetch(`${API_BASE}/db/redis/key`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...redisConfig, key })
        });
        const d = await r.json();

        if (d.error) {
            output.innerHTML = `<div style="color:#ef4444;">${d.error}</div>`;
            return;
        }

        let valueHtml = '';
        if (d.type === 'string') {
            valueHtml = `<pre style="margin:0;white-space:pre-wrap;word-break:break-all;">${escapeHtml(d.value)}</pre>`;
        } else if (d.type === 'list' || d.type === 'set') {
            valueHtml = `<div>${d.value.map((v, i) => `<div style="padding:4px 0;border-bottom:1px solid #3a3a4e;"><span style="color:#6b7280;margin-right:8px;">${i}</span>${escapeHtml(v)}</div>`).join('')}</div>`;
        } else if (d.type === 'hash') {
            valueHtml = `<table class="db-table"><thead><tr><th>Field</th><th>Value</th></tr></thead><tbody>${Object.entries(d.value).map(([k, v]) => `<tr><td>${escapeHtml(k)}</td><td>${escapeHtml(v)}</td></tr>`).join('')}</tbody></table>`;
        } else if (d.type === 'zset') {
            valueHtml = `<table class="db-table"><thead><tr><th>Member</th><th>Score</th></tr></thead><tbody>${d.value.map(m => `<tr><td>${escapeHtml(m.member)}</td><td>${m.score}</td></tr>`).join('')}</tbody></table>`;
        } else {
            valueHtml = `<pre style="margin:0;">${escapeHtml(JSON.stringify(d.value, null, 2))}</pre>`;
        }

        output.innerHTML = `
            <div style="margin-bottom:8px;">
                <strong style="color:#f0f0f0;">${escapeHtml(key)}</strong>
                <span style="color:#6b7280;font-size:11px;margin-left:8px;">Type: ${d.type} | TTL: ${d.ttl < 0 ? 'No expiry' : d.ttl + 's'}</span>
            </div>
            ${valueHtml}
        `;
    } catch (e) {
        output.innerHTML = `<div style="color:#ef4444;">Error: ${e.message}</div>`;
    }
}

export async function redisExecCommand() {
    if (!redisConnected) {
        alert('Please connect first');
        return;
    }
    const command = document.getElementById('redis-command-input').value.trim();
    if (!command) return;

    const output = document.getElementById('redis-command-output');
    output.innerHTML = '<div style="color:#60a5fa;">Executing...</div>';

    try {
        const r = await fetch(`${API_BASE}/db/redis/command`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...redisConfig, command })
        });
        const d = await r.json();

        if (d.error) {
            output.innerHTML = `<div style="color:#ef4444;">Error: ${d.error}</div>`;
            return;
        }

        let resultHtml = '';
        if (d.result === null) {
            resultHtml = '<span style="color:#6b7280;">(nil)</span>';
        } else if (typeof d.result === 'string') {
            resultHtml = `<pre style="margin:0;white-space:pre-wrap;">"${escapeHtml(d.result)}"</pre>`;
        } else if (Array.isArray(d.result)) {
            resultHtml = `<div>${d.result.map((v, i) => `<div style="padding:2px 0;"><span style="color:#6b7280;margin-right:8px;">${i + 1})</span>${v === null ? '<span style="color:#6b7280;">(nil)</span>' : escapeHtml(String(v))}</div>`).join('')}</div>`;
        } else {
            resultHtml = `<pre style="margin:0;">${escapeHtml(String(d.result))}</pre>`;
        }

        output.innerHTML = `
            <div style="color:#6b7280;font-size:11px;margin-bottom:4px;">Duration: ${d.duration_ms?.toFixed(2) || 0}ms</div>
            ${resultHtml}
        `;
    } catch (e) {
        output.innerHTML = `<div style="color:#ef4444;">Error: ${e.message}</div>`;
    }
}

export async function redisLoadCluster() {
    if (!redisConnected) return;
    const container = document.getElementById('redis-cluster-content');
    container.innerHTML = '<div style="color:#60a5fa;">Loading cluster info...</div>';

    try {
        const r = await fetch(`${API_BASE}/db/redis/cluster`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(redisConfig)
        });
        const d = await r.json();

        if (d.error) {
            container.innerHTML = `<div style="color:#6b7280;">${d.error}</div>`;
            return;
        }

        if (!d.enabled) {
            container.innerHTML = '<div style="color:#6b7280;">Cluster mode is not enabled on this Redis instance</div>';
            return;
        }

        container.innerHTML = `
            <div class="db-stats-grid">
                <div class="db-stat-card">
                    <div class="stat-label">State</div>
                    <div class="stat-value" style="color:${d.state === 'ok' ? '#4ade80' : '#ef4444'};">${d.state}</div>
                </div>
                <div class="db-stat-card">
                    <div class="stat-label">Slots OK</div>
                    <div class="stat-value">${d.slots_ok || 0}</div>
                </div>
                <div class="db-stat-card">
                    <div class="stat-label">Known Nodes</div>
                    <div class="stat-value">${d.known_nodes || 0}</div>
                </div>
                <div class="db-stat-card">
                    <div class="stat-label">Size</div>
                    <div class="stat-value">${d.size || 0}</div>
                </div>
            </div>
            ${d.nodes && d.nodes.length ? `
                <div class="info-section">
                    <h4>Nodes</h4>
                    <table class="db-table">
                        <thead><tr><th>ID</th><th>Address</th><th>Role</th><th>Slots</th></tr></thead>
                        <tbody>${d.nodes.map(n => `<tr><td style="font-family:monospace;font-size:11px;">${n.id.substring(0, 8)}...</td><td>${n.address}</td><td>${n.role}</td><td>${n.slots || '-'}</td></tr>`).join('')}</tbody>
                    </table>
                </div>
            ` : ''}
        `;
    } catch (e) {
        container.innerHTML = `<div style="color:#ef4444;">Error: ${e.message}</div>`;
    }
}

// ========== MySQL Functions ==========

let mysqlConnected = false;
let mysqlConfig = {};

export function showMysqlTab(tabId) {
    document.querySelectorAll('#window-mysql .tab-content').forEach(t => t.classList.remove('active'));
    document.querySelectorAll('#window-mysql .tab-btn').forEach(b => b.classList.remove('active'));
    document.getElementById('mysql-tab-' + tabId).classList.add('active');
    event.target.classList.add('active');
    saveState();
}

export async function mysqlConnect() {
    const btn = document.querySelector('#window-mysql .conn-form button');
    const status = document.getElementById('mysql-conn-status');

    mysqlConfig = {
        host: document.getElementById('mysql-host').value || 'localhost',
        port: parseInt(document.getElementById('mysql-port').value) || 3306,
        user: document.getElementById('mysql-user').value || 'root',
        password: document.getElementById('mysql-password').value || '',
        database: document.getElementById('mysql-database').value || ''
    };

    btn.disabled = true;
    btn.textContent = 'Connecting...';
    status.innerHTML = '<span style="color:#60a5fa;">Testing connection...</span>';

    try {
        const r = await fetch(`${API_BASE}/db/mysql/connect`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(mysqlConfig)
        });
        const d = await r.json();

        if (d.success) {
            mysqlConnected = true;
            status.innerHTML = `<span style="color:#4ade80;">Connected to ${d.server_type || 'MySQL'} ${d.version || ''}</span>`;
            btn.textContent = 'Reconnect';
            mysqlLoadInfo();
            mysqlLoadDatabases();
        } else {
            mysqlConnected = false;
            status.innerHTML = `<span style="color:#ef4444;">Failed: ${d.error}</span>`;
            btn.textContent = 'Connect';
        }
    } catch (e) {
        status.innerHTML = `<span style="color:#ef4444;">Error: ${e.message}</span>`;
        btn.textContent = 'Connect';
    }
    btn.disabled = false;
}

export async function mysqlLoadInfo() {
    if (!mysqlConnected) return;
    const container = document.getElementById('mysql-info-content');
    container.innerHTML = '<div style="color:#60a5fa;">Loading...</div>';

    try {
        const r = await fetch(`${API_BASE}/db/mysql/info`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(mysqlConfig)
        });
        const d = await r.json();

        if (d.error) {
            container.innerHTML = `<div style="color:#ef4444;">${d.error}</div>`;
            return;
        }

        let tablesHtml = '';
        if (d.tables && d.tables.length > 0) {
            tablesHtml = `
                <div class="info-section">
                    <h4>Tables (Top ${d.tables.length})</h4>
                    <table class="db-table">
                        <thead><tr><th>Name</th><th>Engine</th><th>Rows</th><th>Data Size</th><th>Index Size</th></tr></thead>
                        <tbody>
                            ${d.tables.map(t => `<tr><td>${t.name}</td><td>${t.engine}</td><td>${t.row_count.toLocaleString()}</td><td>${t.data_size}</td><td>${t.index_size}</td></tr>`).join('')}
                        </tbody>
                    </table>
                </div>
            `;
        }

        container.innerHTML = `
            <div class="db-stats-grid">
                <div class="db-stat-card">
                    <div class="stat-label">Server</div>
                    <div class="stat-value">${d.server_type || 'MySQL'}</div>
                </div>
                <div class="db-stat-card">
                    <div class="stat-label">Version</div>
                    <div class="stat-value" style="font-size:12px;">${d.version || 'N/A'}</div>
                </div>
                <div class="db-stat-card">
                    <div class="stat-label">Database Size</div>
                    <div class="stat-value">${d.database_size || 'N/A'}</div>
                </div>
                <div class="db-stat-card">
                    <div class="stat-label">Tables</div>
                    <div class="stat-value">${d.table_count || 0}</div>
                </div>
                <div class="db-stat-card">
                    <div class="stat-label">Connections</div>
                    <div class="stat-value">${d.connections || 0} / ${d.max_connections || 0}</div>
                </div>
                <div class="db-stat-card">
                    <div class="stat-label">Uptime</div>
                    <div class="stat-value">${d.uptime_human || 'N/A'}</div>
                </div>
                <div class="db-stat-card">
                    <div class="stat-label">Queries/sec</div>
                    <div class="stat-value">${d.queries_per_sec ? d.queries_per_sec.toFixed(1) : 'N/A'}</div>
                </div>
            </div>
            ${tablesHtml}
        `;
    } catch (e) {
        container.innerHTML = `<div style="color:#ef4444;">Error: ${e.message}</div>`;
    }
}

export async function mysqlLoadDatabases() {
    if (!mysqlConnected) return;
    try {
        const r = await fetch(`${API_BASE}/db/mysql/databases`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(mysqlConfig)
        });
        const d = await r.json();
        if (d.databases) {
            // Databases loaded for reference
        }
    } catch (e) {
        console.error('Failed to load databases:', e);
    }
}

export async function mysqlExecuteQuery() {
    if (!mysqlConnected) {
        alert('Please connect first');
        return;
    }
    const query = document.getElementById('mysql-query-input').value.trim();
    if (!query) return;

    const readonly = document.getElementById('mysql-readonly').checked;
    const output = document.getElementById('mysql-query-output');
    output.innerHTML = '<div style="color:#60a5fa;">Executing...</div>';

    try {
        const r = await fetch(`${API_BASE}/db/mysql/query`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...mysqlConfig, query, readonly })
        });
        const d = await r.json();

        if (d.error) {
            output.innerHTML = `<div style="color:#ef4444;">Error: ${d.error}</div><div style="color:#6b7280;font-size:11px;margin-top:4px;">Duration: ${d.duration_ms?.toFixed(2) || 0}ms</div>`;
            return;
        }

        if (d.columns && d.rows) {
            output.innerHTML = `
                <div style="color:#6b7280;font-size:11px;margin-bottom:8px;">${d.rows.length} row(s) in ${d.duration_ms?.toFixed(2) || 0}ms</div>
                <table class="db-table">
                    <thead><tr>${d.columns.map(c => `<th>${escapeHtml(c)}</th>`).join('')}</tr></thead>
                    <tbody>${d.rows.map(row => `<tr>${row.map(v => `<td>${v === null ? '<span style="color:#6b7280;">NULL</span>' : escapeHtml(String(v))}</td>`).join('')}</tr>`).join('')}</tbody>
                </table>
            `;
        } else {
            output.innerHTML = `<div style="color:#4ade80;">${d.rows_affected} row(s) affected</div><div style="color:#6b7280;font-size:11px;margin-top:4px;">Duration: ${d.duration_ms?.toFixed(2) || 0}ms</div>`;
        }
    } catch (e) {
        output.innerHTML = `<div style="color:#ef4444;">Error: ${e.message}</div>`;
    }
}

export async function mysqlDump() {
    if (!mysqlConnected) {
        alert('Please connect first');
        return;
    }
    const schemaOnly = document.getElementById('mysql-dump-schema').checked;
    const dataOnly = document.getElementById('mysql-dump-data').checked;
    const tables = document.getElementById('mysql-dump-tables').value.split(',').map(t => t.trim()).filter(t => t);
    const output = document.getElementById('mysql-dump-output');
    output.value = 'Generating dump...';

    try {
        const r = await fetch(`${API_BASE}/db/mysql/dump`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...mysqlConfig, schema_only: schemaOnly, data_only: dataOnly, tables })
        });
        const d = await r.json();

        if (d.error) {
            output.value = 'Error: ' + d.error;
        } else {
            output.value = d.output || 'No output';
        }
    } catch (e) {
        output.value = 'Error: ' + e.message;
    }
}

export function mysqlCopyDump() {
    const output = document.getElementById('mysql-dump-output');
    window._copyText(output.value).then(() => {
        const btn = event.target;
        const orig = btn.textContent;
        btn.textContent = 'Copied!';
        setTimeout(() => btn.textContent = orig, 1500);
    });
}

export function mysqlDownloadDump() {
    const output = document.getElementById('mysql-dump-output').value;
    if (!output || output.startsWith('Error') || output === 'Generating dump...') return;
    const blob = new Blob([output], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${mysqlConfig.database || 'dump'}_${new Date().toISOString().split('T')[0]}.sql`;
    a.click();
    URL.revokeObjectURL(url);
}
