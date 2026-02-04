// Dev Tools Module for GAGOS

import { API_BASE } from './app.js';
import { saveState } from './state.js';
import { escapeHtml } from './utils.js';

export function showDevToolsTab(tabId) {
    document.querySelectorAll('#window-devtools .tab-content').forEach(t => t.classList.remove('active'));
    document.querySelectorAll('#window-devtools .tab-btn').forEach(b => b.classList.remove('active'));
    document.getElementById('devtools-tab-' + tabId).classList.add('active');
    event.target.classList.add('active');
    saveState();
}

export async function doBase64Encode() {
    const input = document.getElementById('base64-input').value;
    const urlSafe = document.getElementById('base64-urlsafe').checked;
    try {
        const r = await fetch(`${API_BASE}/tools/base64/encode`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ input, url_safe: urlSafe })
        });
        const d = await r.json();
        document.getElementById('base64-output').value = d.output || d.error || '';
    } catch (e) {
        document.getElementById('base64-output').value = 'Error: ' + e.message;
    }
}

export async function doBase64Decode() {
    const input = document.getElementById('base64-input').value;
    const urlSafe = document.getElementById('base64-urlsafe').checked;
    const output = document.getElementById('base64-output');
    if (!input.trim()) {
        output.value = 'Please enter Base64 text to decode';
        return;
    }
    try {
        const r = await fetch(`${API_BASE}/tools/base64/decode`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ input: input.trim(), url_safe: urlSafe })
        });
        const d = await r.json();
        if (d.error) {
            output.value = 'Error: ' + d.error;
        } else {
            output.value = d.output || '';
        }
    } catch (e) {
        output.value = 'Error: ' + e.message;
    }
}

// Parse YAML data: section into key-value object
function parseYamlSecretData(text) {
    const lines = text.split('\n');
    const data = {};
    let inData = false;
    let dataIndent = -1;
    for (const line of lines) {
        if (/^data:\s*$/.test(line)) { inData = true; continue; }
        if (inData) {
            const m = line.match(/^(\s+)(\S+):\s*(.*)$/);
            if (m) {
                if (dataIndent === -1) dataIndent = m[1].length;
                if (m[1].length === dataIndent) {
                    data[m[2]] = m[3].trim();
                    continue;
                }
            }
            if (line.match(/^\S/) && line.trim() !== '') inData = false;
        }
    }
    return data;
}

export async function decodeK8sSecret() {
    const input = document.getElementById('k8s-secret-input').value.trim();
    const output = document.getElementById('k8s-secret-output');
    if (!input) { return; }

    // Try YAML format first (has "data:" line or "kind: Secret")
    if (/^(api[Vv]ersion:|kind:|metadata:|data:)/m.test(input)) {
        const data = parseYamlSecretData(input);
        if (Object.keys(data).length > 0) {
            const decoded = {};
            for (const [k, v] of Object.entries(data)) {
                try { decoded[k] = atob(v); } catch { decoded[k] = '(binary or invalid base64)'; }
            }
            output.style.display = 'block';
            output.style.color = '#4ade80';
            let text = '';
            for (const [k, v] of Object.entries(decoded)) {
                text += `── ${k} ──\n${v}\n\n`;
            }
            output.textContent = text.trimEnd();
            return;
        }
    }

    // Fall back to JSON format
    try {
        const parsed = JSON.parse(input);
        // Support full kubectl JSON (has .data field) or raw data object
        const data = parsed.data || parsed;
        const r = await fetch(`${API_BASE}/tools/base64/k8s-secret`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ data })
        });
        const d = await r.json();
        output.style.display = 'block';
        output.style.color = '#4ade80';
        output.textContent = JSON.stringify(d.decoded, null, 2);
    } catch (e) {
        output.style.display = 'block';
        output.style.color = '#ef4444';
        output.textContent = 'Error: ' + e.message;
    }
}

export async function generateHashes() {
    const input = document.getElementById('hash-input').value;
    try {
        const r = await fetch(`${API_BASE}/tools/hash`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ input, algorithm: 'all' })
        });
        const d = await r.json();
        document.getElementById('hash-md5').textContent = d.md5 || '-';
        document.getElementById('hash-sha1').textContent = d.sha1 || '-';
        document.getElementById('hash-sha256').textContent = d.sha256 || '-';
        document.getElementById('hash-sha512').textContent = d.sha512 || '-';
    } catch (e) {
        console.error('Hash error:', e);
    }
}

function fallbackCopy(text) {
    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.cssText = 'position:fixed;opacity:0;left:-9999px;';
    document.body.appendChild(textarea);
    textarea.select();
    document.execCommand('copy');
    document.body.removeChild(textarea);
}

export function clipboardWrite(text) {
    if (navigator.clipboard && window.isSecureContext) {
        return navigator.clipboard.writeText(text);
    }
    fallbackCopy(text);
    return Promise.resolve();
}
window._copyText = clipboardWrite;

export function copyHashValue(elementId) {
    const el = document.getElementById(elementId);
    const text = el.textContent;
    if (text && text !== '-') {
        clipboardWrite(text).then(() => {
            const original = el.style.color;
            el.style.color = '#60a5fa';
            el.textContent = 'Copied!';
            setTimeout(() => {
                el.style.color = original;
                el.textContent = text;
            }, 1000);
        });
    }
}

export async function compareHashes() {
    const hash1 = document.getElementById('hash-compare1').value;
    const hash2 = document.getElementById('hash-compare2').value;
    const result = document.getElementById('hash-compare-result');
    try {
        const r = await fetch(`${API_BASE}/tools/hash/compare`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ hash1, hash2 })
        });
        const d = await r.json();
        result.style.display = 'block';
        if (d.match) {
            result.style.background = 'rgba(74,222,128,0.2)';
            result.style.color = '#4ade80';
            result.innerHTML = '<strong>Match!</strong> The hashes are identical.';
        } else {
            result.style.background = 'rgba(239,68,68,0.2)';
            result.style.color = '#ef4444';
            result.innerHTML = '<strong>No Match</strong> The hashes are different.';
        }
    } catch (e) {
        result.style.display = 'block';
        result.style.background = 'rgba(239,68,68,0.2)';
        result.style.color = '#ef4444';
        result.textContent = 'Error: ' + e.message;
    }
}

export async function checkCertificate() {
    const host = document.getElementById('cert-host').value;
    const port = parseInt(document.getElementById('cert-port').value) || 443;
    const result = document.getElementById('cert-result');
    result.innerHTML = '<p style="color:#8a8a9a;text-align:center;">Loading...</p>';
    try {
        const r = await fetch(`${API_BASE}/tools/cert/check`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ host, port, timeout: 10 })
        });
        const d = await r.json();
        if (d.error) {
            result.innerHTML = `<p style="color:#ef4444;">${d.error}</p>`;
            return;
        }
        result.innerHTML = formatCertResult(d);
    } catch (e) {
        result.innerHTML = `<p style="color:#ef4444;">Error: ${e.message}</p>`;
    }
}

export async function parseCertificate() {
    const pem = document.getElementById('cert-pem').value;
    const result = document.getElementById('cert-result');
    result.innerHTML = '<p style="color:#8a8a9a;text-align:center;">Parsing...</p>';
    try {
        const r = await fetch(`${API_BASE}/tools/cert/parse`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ pem })
        });
        const d = await r.json();
        if (d.error) {
            result.innerHTML = `<p style="color:#ef4444;">${d.error}</p>`;
            return;
        }
        result.innerHTML = formatCertResult(d);
    } catch (e) {
        result.innerHTML = `<p style="color:#ef4444;">Error: ${e.message}</p>`;
    }
}

function formatCertResult(data) {
    let html = '';
    if (data.host) {
        html += `<div style="margin-bottom:15px;padding:10px;background:#2a2a3e;border-radius:6px;">
            <span style="color:#8a8a9a;">Host:</span> <span style="color:#e0e0e0;">${data.host}:${data.port}</span>
            ${data.protocol ? `<span style="margin-left:10px;color:#8a8a9a;">Protocol:</span> <span style="color:#667eea;">${data.protocol}</span>` : ''}
            <span style="margin-left:10px;color:#8a8a9a;">Chain Valid:</span> <span style="color:${data.chain_valid ? '#4ade80' : '#ef4444'};">${data.chain_valid ? 'Yes' : 'No'}</span>
        </div>`;
    }
    if (data.certificates) {
        data.certificates.forEach((cert, i) => {
            const expiryColor = cert.is_expired ? '#ef4444' : (cert.days_until_expiry < 30 ? '#fbbf24' : '#4ade80');
            html += `<div style="margin-bottom:15px;padding:15px;background:#2a2a3e;border-radius:6px;border-left:3px solid ${i === 0 ? '#667eea' : '#3a3a4e'};">
                <div style="font-weight:500;color:#e0e0e0;margin-bottom:10px;">${i === 0 ? 'Leaf Certificate' : (cert.is_ca ? 'CA Certificate' : 'Intermediate')}</div>
                <div style="display:grid;gap:8px;font-size:12px;">
                    <div><span style="color:#8a8a9a;">Subject:</span> <span style="color:#e0e0e0;">${cert.subject}</span></div>
                    <div><span style="color:#8a8a9a;">Issuer:</span> <span style="color:#e0e0e0;">${cert.issuer}</span></div>
                    <div><span style="color:#8a8a9a;">Valid:</span> <span style="color:#e0e0e0;">${new Date(cert.not_before).toLocaleDateString()} - ${new Date(cert.not_after).toLocaleDateString()}</span></div>
                    <div><span style="color:#8a8a9a;">Expires:</span> <span style="color:${expiryColor};">${cert.is_expired ? 'EXPIRED' : cert.days_until_expiry + ' days'}</span></div>
                    ${cert.dns_names && cert.dns_names.length ? `<div><span style="color:#8a8a9a;">DNS Names:</span> <span style="color:#e0e0e0;">${cert.dns_names.join(', ')}</span></div>` : ''}
                    <div><span style="color:#8a8a9a;">SHA-256:</span> <span style="color:#667eea;font-family:monospace;font-size:10px;">${cert.fingerprint_sha256}</span></div>
                </div>
            </div>`;
        });
    }
    return html || '<p style="color:#8a8a9a;">No certificate data</p>';
}

export async function generateSSHKey() {
    const algorithm = document.getElementById('ssh-algorithm').value;
    const bitSize = parseInt(document.getElementById('ssh-bitsize').value) || 0;
    try {
        const r = await fetch(`${API_BASE}/tools/ssh/generate`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ algorithm, bit_size: bitSize })
        });
        const d = await r.json();
        if (d.error) {
            alert('Error: ' + d.error);
            return;
        }
        document.getElementById('ssh-private').value = d.private_key;
        document.getElementById('ssh-public').value = d.public_key;
        document.getElementById('ssh-fingerprint').querySelector('div').textContent = d.fingerprint;
    } catch (e) {
        alert('Error: ' + e.message);
    }
}

export async function validateSSHKey() {
    const key = document.getElementById('ssh-validate-input').value;
    const result = document.getElementById('ssh-validate-result');
    try {
        const r = await fetch(`${API_BASE}/tools/ssh/info`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ key })
        });
        const d = await r.json();
        result.style.display = 'block';
        if (d.valid) {
            result.innerHTML = `<div style="color:#4ade80;margin-bottom:5px;">Valid ${d.type} key</div>
                <div style="font-size:11px;color:#8a8a9a;">Fingerprint: <span style="color:#667eea;">${d.fingerprint}</span></div>
                ${d.comment ? `<div style="font-size:11px;color:#8a8a9a;">Comment: ${d.comment}</div>` : ''}`;
        } else {
            result.innerHTML = `<div style="color:#ef4444;">Invalid key: ${d.error}</div>`;
        }
    } catch (e) {
        result.style.display = 'block';
        result.innerHTML = `<div style="color:#ef4444;">Error: ${e.message}</div>`;
    }
}

export async function convertFormat() {
    const input = document.getElementById('convert-input').value;
    const from = document.getElementById('convert-from').value;
    const to = document.getElementById('convert-to').value;
    try {
        const r = await fetch(`${API_BASE}/tools/convert`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ input, from, to })
        });
        const d = await r.json();
        document.getElementById('convert-output').value = d.output || d.error || '';
    } catch (e) {
        document.getElementById('convert-output').value = 'Error: ' + e.message;
    }
}

export async function formatJSON() {
    const input = document.getElementById('convert-input').value;
    try {
        const r = await fetch(`${API_BASE}/tools/convert`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ input, from: 'json', to: 'json' })
        });
        const d = await r.json();
        document.getElementById('convert-output').value = d.output || d.error || '';
    } catch (e) {
        document.getElementById('convert-output').value = 'Error: ' + e.message;
    }
}

export async function minifyJSON() {
    const input = document.getElementById('convert-input').value;
    try {
        const r = await fetch(`${API_BASE}/tools/convert`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ input, from: 'json', to: 'minified' })
        });
        const d = await r.json();
        document.getElementById('convert-output').value = d.output || d.error || '';
    } catch (e) {
        document.getElementById('convert-output').value = 'Error: ' + e.message;
    }
}

export async function computeDiff() {
    const text1 = document.getElementById('diff-input1').value;
    const text2 = document.getElementById('diff-input2').value;
    const type = document.getElementById('diff-type').value;
    const output = document.getElementById('diff-output');
    const stats = document.getElementById('diff-stats');
    try {
        const r = await fetch(`${API_BASE}/tools/diff`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ text1, text2, type })
        });
        const d = await r.json();
        if (d.error) {
            output.innerHTML = `<span style="color:#ef4444;">${d.error}</span>`;
            stats.textContent = '';
            return;
        }
        if (d.identical) {
            output.innerHTML = '<span style="color:#4ade80;">Files are identical</span>';
            stats.textContent = 'No differences';
        } else {
            let html = '';
            if (d.diff_lines) {
                d.diff_lines.forEach(line => {
                    const color = line.type === 'add' ? '#4ade80' : (line.type === 'delete' ? '#ef4444' : '#8a8a9a');
                    const prefix = line.type === 'add' ? '+ ' : (line.type === 'delete' ? '- ' : '  ');
                    html += `<div style="color:${color};">${prefix}${escapeHtml(line.content)}</div>`;
                });
            } else {
                html = escapeHtml(d.diff);
            }
            output.innerHTML = html;
            stats.innerHTML = `<span style="color:#4ade80;">+${d.additions}</span> <span style="color:#ef4444;">-${d.deletions}</span>`;
        }
    } catch (e) {
        output.innerHTML = `<span style="color:#ef4444;">Error: ${e.message}</span>`;
        stats.textContent = '';
    }
}

export function copyToClipboard(elementId) {
    const el = document.getElementById(elementId);
    if (el) {
        clipboardWrite(el.value || el.textContent).catch(err => {
            console.error('Copy failed:', err);
        });
    }
}
