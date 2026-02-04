// S3 Storage Module for GAGOS
// Supports AWS S3, MinIO, and other S3-compatible services

import { API_BASE } from './app.js';
import { saveState } from './state.js';
import { escapeHtml, formatSize, formatTime } from './utils.js';

let s3Connected = false;
let s3Config = {};
let currentBucket = '';
let currentPrefix = '';

export function showS3Tab(tabId) {
    document.querySelectorAll('#window-s3 .tab-content').forEach(t => t.classList.remove('active'));
    document.querySelectorAll('#window-s3 .tab-btn').forEach(b => b.classList.remove('active'));
    document.getElementById('s3-tab-' + tabId).classList.add('active');
    event.target.classList.add('active');
    saveState();
}

export function s3LoadPreset() {
    const preset = document.getElementById('s3-preset').value;
    const endpoint = document.getElementById('s3-endpoint');
    const region = document.getElementById('s3-region');
    const ssl = document.getElementById('s3-ssl');

    if (preset === 'aws') {
        endpoint.value = 's3.amazonaws.com';
        region.value = 'us-east-1';
        ssl.checked = true;
    } else if (preset === 'minio') {
        endpoint.value = 'localhost:9000';
        region.value = 'us-east-1';
        ssl.checked = false;
    }
}

export async function s3Connect() {
    const btn = document.querySelector('#window-s3 .conn-form .action-btn');
    const status = document.getElementById('s3-conn-status');

    s3Config = {
        endpoint: document.getElementById('s3-endpoint').value || 's3.amazonaws.com',
        region: document.getElementById('s3-region').value || 'us-east-1',
        access_key_id: document.getElementById('s3-access-key').value,
        secret_access_key: document.getElementById('s3-secret-key').value,
        use_ssl: document.getElementById('s3-ssl').checked
    };

    if (!s3Config.access_key_id || !s3Config.secret_access_key) {
        status.innerHTML = '<span style="color:#ef4444;">Access key and secret key are required</span>';
        return;
    }

    btn.disabled = true;
    btn.textContent = 'Connecting...';
    status.innerHTML = '<span style="color:#60a5fa;">Testing connection...</span>';

    try {
        const r = await fetch(`${API_BASE}/storage/s3/connect`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(s3Config)
        });
        const d = await r.json();

        if (d.success) {
            s3Connected = true;
            status.innerHTML = `<span style="color:#4ade80;">Connected (${d.response_time_ms?.toFixed(0) || 0}ms) - ${d.buckets?.length || 0} buckets</span>`;
            btn.textContent = 'Reconnect';
            s3LoadBuckets();
        } else {
            s3Connected = false;
            status.innerHTML = `<span style="color:#ef4444;">Failed: ${d.error}</span>`;
            btn.textContent = 'Connect';
        }
    } catch (e) {
        status.innerHTML = `<span style="color:#ef4444;">Error: ${e.message}</span>`;
        btn.textContent = 'Connect';
    }
    btn.disabled = false;
}

export async function s3LoadBuckets() {
    if (!s3Connected) return;
    const container = document.getElementById('s3-buckets-list');
    container.innerHTML = '<div style="color:#60a5fa;padding:20px;">Loading buckets...</div>';

    try {
        const r = await fetch(`${API_BASE}/storage/s3/buckets`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(s3Config)
        });
        const d = await r.json();

        if (d.error) {
            container.innerHTML = `<div style="color:#ef4444;padding:20px;">${d.error}</div>`;
            return;
        }

        if (!d.buckets || d.buckets.length === 0) {
            container.innerHTML = '<div style="color:#6b7280;padding:20px;">No buckets found. Create one to get started.</div>';
            return;
        }

        container.innerHTML = `
            <table class="db-table">
                <thead>
                    <tr>
                        <th>Name</th>
                        <th>Created</th>
                        <th style="width:120px;">Actions</th>
                    </tr>
                </thead>
                <tbody>
                    ${d.buckets.map(b => `
                        <tr>
                            <td>
                                <a href="#" onclick="event.preventDefault(); s3SelectBucket('${escapeHtml(b.name)}')" style="color:#60a5fa;text-decoration:none;">
                                    ${escapeHtml(b.name)}
                                </a>
                            </td>
                            <td style="color:#8a8a9a;font-size:12px;">${formatTime(b.creation_date)}</td>
                            <td>
                                <button class="action-btn small" onclick="s3SelectBucket('${escapeHtml(b.name)}')" title="Browse">Browse</button>
                                <button class="action-btn small danger" onclick="s3DeleteBucket('${escapeHtml(b.name)}')" title="Delete">Delete</button>
                            </td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        `;
    } catch (e) {
        container.innerHTML = `<div style="color:#ef4444;padding:20px;">Error: ${e.message}</div>`;
    }
}

export async function s3CreateBucket() {
    if (!s3Connected) {
        alert('Please connect first');
        return;
    }

    const name = prompt('Enter bucket name (lowercase, no spaces):');
    if (!name) return;

    // Validate bucket name
    if (!/^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$/.test(name)) {
        alert('Invalid bucket name. Use lowercase letters, numbers, dots, and hyphens. Must be 3-63 characters.');
        return;
    }

    try {
        const r = await fetch(`${API_BASE}/storage/s3/bucket/create`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...s3Config, bucket: name })
        });
        const d = await r.json();

        if (d.success) {
            s3LoadBuckets();
        } else {
            alert('Failed to create bucket: ' + d.error);
        }
    } catch (e) {
        alert('Error: ' + e.message);
    }
}

export async function s3DeleteBucket(name) {
    if (!s3Connected) return;

    if (!confirm(`Delete bucket "${name}"? The bucket must be empty.`)) return;

    try {
        const r = await fetch(`${API_BASE}/storage/s3/bucket/delete`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...s3Config, bucket: name })
        });
        const d = await r.json();

        if (d.success) {
            s3LoadBuckets();
            if (currentBucket === name) {
                currentBucket = '';
                currentPrefix = '';
            }
        } else {
            alert('Failed to delete bucket: ' + d.error);
        }
    } catch (e) {
        alert('Error: ' + e.message);
    }
}

export async function s3SelectBucket(bucket) {
    currentBucket = bucket;
    currentPrefix = '';

    // Switch to objects tab
    document.querySelectorAll('#window-s3 .tab-content').forEach(t => t.classList.remove('active'));
    document.querySelectorAll('#window-s3 .tab-btn').forEach(b => b.classList.remove('active'));
    document.getElementById('s3-tab-objects').classList.add('active');
    document.querySelectorAll('#window-s3 .tab-btn')[1].classList.add('active');

    document.getElementById('s3-current-bucket').textContent = bucket;
    s3LoadObjects();
}

export async function s3LoadObjects(prefix = '') {
    if (!s3Connected || !currentBucket) return;

    currentPrefix = prefix;
    const container = document.getElementById('s3-objects-list');
    const pathEl = document.getElementById('s3-path');

    pathEl.textContent = '/' + currentBucket + '/' + currentPrefix;
    container.innerHTML = '<div style="color:#60a5fa;padding:20px;">Loading objects...</div>';

    try {
        const r = await fetch(`${API_BASE}/storage/s3/objects`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...s3Config, bucket: currentBucket, prefix: currentPrefix, max_keys: 1000 })
        });
        const d = await r.json();

        if (d.error) {
            container.innerHTML = `<div style="color:#ef4444;padding:20px;">${d.error}</div>`;
            return;
        }

        if (!d.objects || d.objects.length === 0) {
            container.innerHTML = '<div style="color:#6b7280;padding:20px;">No objects in this location. Upload files to get started.</div>';
            return;
        }

        // Sort: directories first, then files alphabetically
        const sorted = d.objects.sort((a, b) => {
            if (a.is_dir && !b.is_dir) return -1;
            if (!a.is_dir && b.is_dir) return 1;
            return a.key.localeCompare(b.key);
        });

        container.innerHTML = `
            <table class="db-table">
                <thead>
                    <tr>
                        <th>Name</th>
                        <th style="width:100px;">Size</th>
                        <th style="width:160px;">Last Modified</th>
                        <th style="width:180px;">Actions</th>
                    </tr>
                </thead>
                <tbody>
                    ${sorted.map(obj => {
                        const displayName = getObjectDisplayName(obj.key, currentPrefix);
                        if (obj.is_dir) {
                            return `
                                <tr>
                                    <td>
                                        <span style="color:#fbbf24;margin-right:6px;">&#128193;</span>
                                        <a href="#" onclick="event.preventDefault(); s3NavigateFolder('${escapeHtml(obj.key)}')" style="color:#60a5fa;text-decoration:none;">
                                            ${escapeHtml(displayName)}
                                        </a>
                                    </td>
                                    <td style="color:#8a8a9a;">-</td>
                                    <td style="color:#8a8a9a;">-</td>
                                    <td>
                                        <button class="action-btn small" onclick="s3NavigateFolder('${escapeHtml(obj.key)}')">Open</button>
                                    </td>
                                </tr>
                            `;
                        } else {
                            return `
                                <tr>
                                    <td>
                                        <span style="color:#8a8a9a;margin-right:6px;">&#128196;</span>
                                        <a href="#" onclick="event.preventDefault(); s3GetInfo('${escapeHtml(obj.key)}')" style="color:#f0f0f0;text-decoration:none;">
                                            ${escapeHtml(displayName)}
                                        </a>
                                    </td>
                                    <td style="color:#8a8a9a;font-size:12px;">${formatSize(obj.size)}</td>
                                    <td style="color:#8a8a9a;font-size:12px;">${formatTime(obj.last_modified)}</td>
                                    <td>
                                        <button class="action-btn small" onclick="s3DownloadFile('${escapeHtml(obj.key)}')" title="Download">&#8595;</button>
                                        <button class="action-btn small" onclick="s3GetInfo('${escapeHtml(obj.key)}')" title="Info">&#9432;</button>
                                        <button class="action-btn small" onclick="s3GetPresignedURL('${escapeHtml(obj.key)}')" title="Share Link">&#128279;</button>
                                        <button class="action-btn small danger" onclick="s3DeleteFile('${escapeHtml(obj.key)}')" title="Delete">&#10005;</button>
                                    </td>
                                </tr>
                            `;
                        }
                    }).join('')}
                </tbody>
            </table>
        `;
    } catch (e) {
        container.innerHTML = `<div style="color:#ef4444;padding:20px;">Error: ${e.message}</div>`;
    }
}

function getObjectDisplayName(key, prefix) {
    let name = key;
    if (prefix && key.startsWith(prefix)) {
        name = key.substring(prefix.length);
    }
    // Remove trailing slash for display
    if (name.endsWith('/')) {
        name = name.slice(0, -1);
    }
    return name;
}

export function s3NavigateFolder(folder) {
    s3LoadObjects(folder);
}

export function s3GoUp() {
    if (!currentPrefix) return;

    // Remove trailing slash and go up one level
    let prefix = currentPrefix;
    if (prefix.endsWith('/')) {
        prefix = prefix.slice(0, -1);
    }
    const lastSlash = prefix.lastIndexOf('/');
    const newPrefix = lastSlash > 0 ? prefix.substring(0, lastSlash + 1) : '';
    s3LoadObjects(newPrefix);
}

export function s3UploadFiles() {
    if (!s3Connected || !currentBucket) {
        alert('Please select a bucket first');
        return;
    }
    document.getElementById('s3-file-input').click();
}

export async function s3HandleFileSelect() {
    const input = document.getElementById('s3-file-input');
    const files = input.files;
    if (!files || files.length === 0) return;

    const status = document.getElementById('s3-upload-status');
    status.style.display = 'block';
    status.innerHTML = `<span style="color:#60a5fa;">Uploading ${files.length} file(s)...</span>`;

    let successCount = 0;
    let errorCount = 0;

    for (const file of files) {
        try {
            const formData = new FormData();
            formData.append('file', file);
            formData.append('endpoint', s3Config.endpoint);
            formData.append('region', s3Config.region);
            formData.append('access_key_id', s3Config.access_key_id);
            formData.append('secret_access_key', s3Config.secret_access_key);
            formData.append('use_ssl', s3Config.use_ssl);
            formData.append('bucket', currentBucket);
            formData.append('prefix', currentPrefix);

            const r = await fetch(`${API_BASE}/storage/s3/object/upload`, {
                method: 'POST',
                body: formData
            });
            const d = await r.json();

            if (d.success) {
                successCount++;
            } else {
                errorCount++;
                console.error(`Failed to upload ${file.name}:`, d.error);
            }
        } catch (e) {
            errorCount++;
            console.error(`Error uploading ${file.name}:`, e);
        }
    }

    // Clear the file input
    input.value = '';

    if (errorCount === 0) {
        status.innerHTML = `<span style="color:#4ade80;">Uploaded ${successCount} file(s) successfully</span>`;
    } else {
        status.innerHTML = `<span style="color:#fbbf24;">Uploaded ${successCount} file(s), ${errorCount} failed</span>`;
    }

    setTimeout(() => {
        status.style.display = 'none';
    }, 3000);

    // Refresh the objects list
    s3LoadObjects(currentPrefix);
}

export async function s3DownloadFile(key) {
    if (!s3Connected || !currentBucket) return;

    try {
        const r = await fetch(`${API_BASE}/storage/s3/object/download`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...s3Config, bucket: currentBucket, key })
        });

        if (!r.ok) {
            const d = await r.json();
            alert('Download failed: ' + (d.error || 'Unknown error'));
            return;
        }

        // Get filename from Content-Disposition header or key
        let filename = key.split('/').pop();
        const disposition = r.headers.get('Content-Disposition');
        if (disposition) {
            const match = disposition.match(/filename="(.+)"/);
            if (match) filename = match[1];
        }

        const blob = await r.blob();
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        a.click();
        URL.revokeObjectURL(url);
    } catch (e) {
        alert('Error: ' + e.message);
    }
}

export async function s3DeleteFile(key) {
    if (!s3Connected || !currentBucket) return;

    const filename = key.split('/').pop();
    if (!confirm(`Delete "${filename}"?`)) return;

    try {
        const r = await fetch(`${API_BASE}/storage/s3/object/delete`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...s3Config, bucket: currentBucket, key })
        });
        const d = await r.json();

        if (d.success) {
            s3LoadObjects(currentPrefix);
        } else {
            alert('Failed to delete: ' + d.error);
        }
    } catch (e) {
        alert('Error: ' + e.message);
    }
}

export async function s3GetInfo(key) {
    if (!s3Connected || !currentBucket) return;

    const modal = document.getElementById('s3-info-modal');
    const content = document.getElementById('s3-info-content');

    modal.style.display = 'flex';
    content.innerHTML = '<div style="color:#60a5fa;">Loading metadata...</div>';

    try {
        const r = await fetch(`${API_BASE}/storage/s3/object/info`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...s3Config, bucket: currentBucket, key })
        });
        const d = await r.json();

        if (d.error) {
            content.innerHTML = `<div style="color:#ef4444;">${d.error}</div>`;
            return;
        }

        const filename = key.split('/').pop();
        content.innerHTML = `
            <h3 style="margin:0 0 15px 0;color:#f0f0f0;">${escapeHtml(filename)}</h3>
            <div class="info-grid" style="display:grid;grid-template-columns:120px 1fr;gap:8px;font-size:13px;">
                <div style="color:#8a8a9a;">Key:</div>
                <div style="word-break:break-all;">${escapeHtml(d.key)}</div>
                <div style="color:#8a8a9a;">Size:</div>
                <div>${formatSize(d.size)} (${d.size.toLocaleString()} bytes)</div>
                <div style="color:#8a8a9a;">Content Type:</div>
                <div>${escapeHtml(d.content_type)}</div>
                <div style="color:#8a8a9a;">Last Modified:</div>
                <div>${formatTime(d.last_modified)}</div>
                <div style="color:#8a8a9a;">ETag:</div>
                <div style="font-family:monospace;font-size:11px;">${escapeHtml(d.etag)}</div>
            </div>
            ${d.metadata && Object.keys(d.metadata).length > 0 ? `
                <h4 style="margin:15px 0 10px 0;color:#f0f0f0;">Metadata</h4>
                <div class="info-grid" style="display:grid;grid-template-columns:120px 1fr;gap:8px;font-size:13px;">
                    ${Object.entries(d.metadata).map(([k, v]) => `
                        <div style="color:#8a8a9a;">${escapeHtml(k)}:</div>
                        <div>${escapeHtml(v)}</div>
                    `).join('')}
                </div>
            ` : ''}
            <div style="margin-top:20px;display:flex;gap:10px;">
                <button class="action-btn" onclick="s3DownloadFile('${escapeHtml(key)}')">Download</button>
                <button class="action-btn" onclick="s3GetPresignedURL('${escapeHtml(key)}')">Get Share Link</button>
            </div>
        `;
    } catch (e) {
        content.innerHTML = `<div style="color:#ef4444;">Error: ${e.message}</div>`;
    }
}

export function s3CloseModal() {
    document.getElementById('s3-info-modal').style.display = 'none';
}

export async function s3GetPresignedURL(key) {
    if (!s3Connected || !currentBucket) return;

    const hours = parseInt(prompt('Link expiry in hours (1-168):', '24'));
    if (isNaN(hours) || hours < 1 || hours > 168) {
        alert('Please enter a valid number of hours (1-168)');
        return;
    }

    try {
        const r = await fetch(`${API_BASE}/storage/s3/object/presign`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...s3Config, bucket: currentBucket, key, expiry_hours: hours })
        });
        const d = await r.json();

        if (d.error) {
            alert('Failed to generate URL: ' + d.error);
            return;
        }

        // Show in modal
        const modal = document.getElementById('s3-info-modal');
        const content = document.getElementById('s3-info-content');
        modal.style.display = 'flex';

        const filename = key.split('/').pop();
        content.innerHTML = `
            <h3 style="margin:0 0 15px 0;color:#f0f0f0;">Share Link for ${escapeHtml(filename)}</h3>
            <p style="color:#8a8a9a;margin-bottom:10px;">This link expires in ${hours} hour(s):</p>
            <textarea id="s3-presign-url" readonly style="width:100%;height:100px;background:#1e1e2e;color:#f0f0f0;border:1px solid #3a3a4e;border-radius:4px;padding:10px;font-family:monospace;font-size:11px;resize:none;">${escapeHtml(d.url)}</textarea>
            <div style="margin-top:15px;display:flex;gap:10px;">
                <button class="action-btn" onclick="s3CopyPresignedURL()">Copy to Clipboard</button>
                <button class="action-btn" onclick="window.open('${escapeHtml(d.url)}', '_blank')">Open in New Tab</button>
            </div>
        `;
    } catch (e) {
        alert('Error: ' + e.message);
    }
}

export function s3CopyPresignedURL() {
    const textarea = document.getElementById('s3-presign-url');
    if (textarea) {
        window._copyText(textarea.value).then(() => {
            const btn = event.target;
            const orig = btn.textContent;
            btn.textContent = 'Copied!';
            setTimeout(() => btn.textContent = orig, 1500);
        });
    }
}
