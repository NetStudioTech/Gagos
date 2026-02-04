// Network Tools Module for GAGOS

import { API_BASE } from './app.js';
import { saveState } from './state.js';

// Tab switching
export function showNetTab(tabId) {
    document.querySelectorAll('#window-network .tab-btn').forEach(b => b.classList.remove('active'));
    document.querySelectorAll('#window-network .tab-content').forEach(c => c.classList.remove('active'));
    const tabBtn = document.querySelector(`#window-network .tab-btn[onclick="showNetTab('${tabId}')"]`);
    if (tabBtn) tabBtn.classList.add('active');
    const tabContent = document.getElementById('net-tab-' + tabId);
    if (tabContent) tabContent.classList.add('active');
    saveState();
}

export async function runPing() {
    const host = document.getElementById('ping-host').value;
    const count = parseInt(document.getElementById('ping-count').value) || 4;
    const result = document.getElementById('ping-result');
    if (!host) { result.textContent = 'Please enter a host'; return; }
    result.textContent = 'Pinging ' + host + '...';
    result.className = 'output-box';
    try {
        const r = await fetch(`${API_BASE}/network/ping`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ host, count, timeout: 10 })
        });
        const d = await r.json();
        if (d.error) { result.textContent = 'Error: ' + d.error; result.className = 'output-box error'; return; }
        let out = `PING ${host} (${d.ip || 'unknown'})\n\n`;
        if (d.rtts?.length > 0) d.rtts.forEach((rtt, i) => out += `Reply ${i+1}: time=${rtt.toFixed(2)}ms\n`);
        out += `\n--- ${host} ping statistics ---\n`;
        out += `${d.packets_sent || 0} transmitted, ${d.packets_recv || 0} received, ${(d.packet_loss || 0).toFixed(1)}% loss\n`;
        if (d.success) out += `rtt min/avg/max = ${(d.min_rtt||0).toFixed(2)}/${(d.avg_rtt||0).toFixed(2)}/${(d.max_rtt||0).toFixed(2)} ms`;
        result.textContent = out;
    } catch (e) { result.textContent = 'Error: ' + e.message; result.className = 'output-box error'; }
}

export async function runDNS() {
    const host = document.getElementById('dns-host').value;
    const type = document.getElementById('dns-type').value;
    const result = document.getElementById('dns-result');
    if (!host) { result.textContent = 'Please enter a domain'; return; }
    result.textContent = 'Looking up ' + host + '...';
    result.className = 'output-box';
    try {
        const r = await fetch(`${API_BASE}/network/dns`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ host, record_type: type })
        });
        const d = await r.json();
        if (d.error) { result.textContent = 'Error: ' + d.error; result.className = 'output-box error'; return; }
        let out = `DNS Lookup: ${host} (${type})\n\n`;
        if (d.addresses?.length > 0) { out += 'IP Addresses:\n'; d.addresses.forEach(ip => out += `  ${ip}\n`); }
        if (d.cname) out += `\nCNAME: ${d.cname}\n`;
        if (d.mx?.length > 0) { out += '\nMX Records:\n'; d.mx.forEach(mx => out += `  ${mx.host} (pri: ${mx.pref})\n`); }
        if (d.ns?.length > 0) { out += '\nNS Records:\n'; d.ns.forEach(ns => out += `  ${ns}\n`); }
        if (d.txt?.length > 0) { out += '\nTXT Records:\n'; d.txt.forEach(txt => out += `  ${txt}\n`); }
        result.textContent = out;
    } catch (e) { result.textContent = 'Error: ' + e.message; result.className = 'output-box error'; }
}

export async function runPortCheck() {
    const host = document.getElementById('port-host').value;
    const port = parseInt(document.getElementById('port-number').value);
    const result = document.getElementById('port-result');
    if (!host || !port) { result.textContent = 'Please enter host and port'; return; }
    result.textContent = 'Checking ' + host + ':' + port + '...';
    result.className = 'output-box';
    try {
        const r = await fetch(`${API_BASE}/network/port-check`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ host, port, timeout: 5 })
        });
        const d = await r.json();
        if (d.error) { result.textContent = 'Error: ' + d.error; result.className = 'output-box error'; return; }
        result.textContent = `Port ${port} on ${host}: ${d.open ? 'OPEN' : 'CLOSED'}\nResponse time: ${(d.response_time_ms||0).toFixed(2)}ms`;
        if (!d.open) result.className = 'output-box error';
    } catch (e) { result.textContent = 'Error: ' + e.message; result.className = 'output-box error'; }
}

export async function runTraceroute() {
    const host = document.getElementById('trace-host').value;
    const hops = parseInt(document.getElementById('trace-hops').value) || 15;
    const result = document.getElementById('trace-result');
    if (!host) { result.textContent = 'Please enter a host'; return; }
    result.textContent = 'Tracing route to ' + host + '...';
    result.className = 'output-box';
    try {
        const r = await fetch(`${API_BASE}/network/traceroute`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ host, max_hops: hops, timeout: 30 })
        });
        const d = await r.json();
        if (d.error) { result.textContent = 'Error: ' + d.error; result.className = 'output-box error'; return; }
        let out = `Traceroute to ${host} (${d.target_ip || '?'}), ${hops} hops max\n\n`;
        if (d.hops?.length > 0) {
            d.hops.forEach(h => {
                out += h.ip === '*' ? `${h.hop.toString().padStart(2)}  *  timeout\n` :
                    `${h.hop.toString().padStart(2)}  ${h.ip.padEnd(16)} ${(h.rtt_ms||0).toFixed(2)}ms\n`;
            });
        }
        out += d.reached ? '\nTrace complete.' : '\nTrace incomplete.';
        result.textContent = out;
    } catch (e) { result.textContent = 'Error: ' + e.message; result.className = 'output-box error'; }
}

export async function runTelnet() {
    const host = document.getElementById('telnet-host').value;
    const port = parseInt(document.getElementById('telnet-port').value);
    const command = document.getElementById('telnet-command').value;
    const result = document.getElementById('telnet-result');
    if (!host || !port) { result.textContent = 'Please enter host and port'; return; }
    result.textContent = `Connecting to ${host}:${port}...`;
    result.className = 'output-box';
    try {
        const r = await fetch(`${API_BASE}/network/telnet`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ host, port, command, timeout: 10 })
        });
        const d = await r.json();
        if (d.error) { result.textContent = 'Error: ' + d.error; result.className = 'output-box error'; return; }
        let out = `Telnet ${host}:${port}\n`;
        out += `Connected: ${d.connected ? 'Yes' : 'No'}\n`;
        out += `Duration: ${(d.duration_ms||0).toFixed(2)}ms\n`;
        if (d.response) out += `\n--- Response ---\n${d.response}`;
        result.textContent = out;
        if (!d.connected) result.className = 'output-box error';
    } catch (e) { result.textContent = 'Error: ' + e.message; result.className = 'output-box error'; }
}

export async function runWhois() {
    const query = document.getElementById('whois-query').value;
    const result = document.getElementById('whois-result');
    if (!query) { result.textContent = 'Please enter a domain or IP'; return; }
    result.textContent = `Looking up ${query}...`;
    result.className = 'output-box';
    try {
        const r = await fetch(`${API_BASE}/network/whois`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ query, timeout: 10 })
        });
        const d = await r.json();
        if (d.error) { result.textContent = 'Error: ' + d.error; result.className = 'output-box error'; return; }
        let out = `WHOIS: ${query}\n`;
        out += `Server: ${d.server}\n`;
        out += `Duration: ${(d.duration_ms||0).toFixed(2)}ms\n`;
        out += `\n${d.response || 'No data'}`;
        result.textContent = out;
    } catch (e) { result.textContent = 'Error: ' + e.message; result.className = 'output-box error'; }
}

export async function runSSLCheck() {
    const host = document.getElementById('ssl-host').value;
    const port = parseInt(document.getElementById('ssl-port').value) || 443;
    const result = document.getElementById('ssl-result');
    if (!host) { result.textContent = 'Please enter a host'; return; }
    result.textContent = `Checking SSL certificate for ${host}:${port}...`;
    result.className = 'output-box';
    try {
        const r = await fetch(`${API_BASE}/network/ssl-check`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ host, port, timeout: 10 })
        });
        const d = await r.json();
        if (d.error) { result.textContent = 'Error: ' + d.error; result.className = 'output-box error'; return; }
        let out = `SSL Certificate for ${host}:${port}\n`;
        out += `${'='.repeat(50)}\n\n`;
        out += `Valid:         ${d.valid ? 'Yes' : 'No'}\n`;
        out += `Subject:       ${d.subject || '-'}\n`;
        out += `Issuer:        ${d.issuer || '-'}\n`;
        out += `Not Before:    ${d.not_before || '-'}\n`;
        out += `Not After:     ${d.not_after || '-'}\n`;
        out += `Days Left:     ${d.days_left || '-'}\n`;
        out += `Version:       ${d.version || '-'}\n`;
        out += `Serial:        ${d.serial_number || '-'}\n`;
        if (d.dns_names?.length > 0) {
            out += `\nDNS Names:\n`;
            d.dns_names.forEach(n => out += `  - ${n}\n`);
        }
        out += `\nDuration: ${(d.duration_ms||0).toFixed(2)}ms`;
        result.textContent = out;
        if (!d.valid || d.days_left < 30) result.className = 'output-box error';
    } catch (e) { result.textContent = 'Error: ' + e.message; result.className = 'output-box error'; }
}

export async function runCurl() {
    const url = document.getElementById('curl-url').value;
    const method = document.getElementById('curl-method').value;
    const headersText = document.getElementById('curl-headers').value;
    const body = document.getElementById('curl-body').value;
    const followRedirects = document.getElementById('curl-follow').checked;
    const includeBody = document.getElementById('curl-body-include').checked;
    const result = document.getElementById('curl-result');

    if (!url) { result.textContent = 'Please enter a URL'; return; }
    result.textContent = `${method} ${url}...`;
    result.className = 'output-box';

    // Parse headers
    const headers = {};
    if (headersText) {
        headersText.split('\n').forEach(line => {
            const idx = line.indexOf(':');
            if (idx > 0) {
                headers[line.slice(0, idx).trim()] = line.slice(idx + 1).trim();
            }
        });
    }

    try {
        const r = await fetch(`${API_BASE}/network/curl`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ url, method, headers, body, follow_redirects: followRedirects, include_body: includeBody, timeout: 30 })
        });
        const d = await r.json();
        if (d.error) { result.textContent = 'Error: ' + d.error; result.className = 'output-box error'; return; }

        let out = `${method} ${url}\n`;
        out += `${'='.repeat(60)}\n\n`;
        out += `Status:        ${d.status_code} ${d.status}\n`;
        out += `Protocol:      ${d.protocol || '-'}\n`;
        out += `TLS Version:   ${d.tls_version || '-'}\n`;
        out += `Content-Type:  ${d.content_type || '-'}\n`;
        out += `Content-Len:   ${d.content_length || '-'}\n`;
        if (d.redirect_url) out += `Redirect:      ${d.redirect_url}\n`;
        out += `Duration:      ${(d.duration_ms||0).toFixed(2)}ms\n`;

        if (d.headers && Object.keys(d.headers).length > 0) {
            out += `\n--- Response Headers ---\n`;
            for (const [k, v] of Object.entries(d.headers)) {
                out += `${k}: ${Array.isArray(v) ? v.join(', ') : v}\n`;
            }
        }

        if (d.body) {
            out += `\n--- Response Body ---\n${d.body}`;
        }

        result.textContent = out;
        if (d.status_code >= 400) result.className = 'output-box error';
    } catch (e) { result.textContent = 'Error: ' + e.message; result.className = 'output-box error'; }
}

export async function loadInterfaces() {
    const result = document.getElementById('interfaces-result');
    result.textContent = 'Loading network interfaces...';
    result.className = 'output-box';
    try {
        const r = await fetch(`${API_BASE}/network/interfaces`);
        const d = await r.json();
        if (d.error) { result.textContent = 'Error: ' + d.error; result.className = 'output-box error'; return; }

        let out = `Network Interfaces\n${'='.repeat(50)}\n\n`;
        if (d.interfaces?.length > 0) {
            d.interfaces.forEach(iface => {
                out += `${iface.name}\n`;
                out += `  MTU:     ${iface.mtu}\n`;
                out += `  HW Addr: ${iface.hw_addr || '-'}\n`;
                out += `  Flags:   ${iface.flags}\n`;
                if (iface.addresses?.length > 0) {
                    out += `  Addresses:\n`;
                    iface.addresses.forEach(addr => out += `    - ${addr}\n`);
                }
                out += '\n';
            });
        } else {
            out += 'No interfaces found';
        }
        result.textContent = out;
    } catch (e) { result.textContent = 'Error: ' + e.message; result.className = 'output-box error'; }
}
