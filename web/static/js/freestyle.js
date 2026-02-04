// Freestyle Jobs and SSH Hosts Module for GAGOS
// Improved UI with tabbed interface

import { API_BASE } from './app.js';
import { formatDuration, formatTime } from './utils.js';
import { getRunStatusClass } from './cicd.js';

let sshHosts = [];
let freestyleJobs = [];
let freestyleBuilds = [];
let gitCredentials = [];
let buildLogWs = null;

// Job editor state
let editingJob = null;
let buildSteps = [];
let jobParameters = [];
let envVariables = [];
let currentTab = 'general';
let jobName = '';
let jobDesc = '';
let jobEnabled = true;
let webhookEnabled = false;
let cronEnabled = false;
let cronSchedule = '';

// SCM state
let scmType = 'none';
let scmRepositories = [];
let scmBranches = [];
let scmCloneDepth = 0;
let scmSubmodules = false;
let scmCleanBefore = false;

// ============ SSH Hosts ============

export async function loadSSHHosts() {
    try {
        const r = await fetch(`${API_BASE}/cicd/ssh/hosts`);
        const d = await r.json();
        sshHosts = d.hosts || [];
        renderSSHHostsTable();
    } catch (e) {
        console.error('Failed to load SSH hosts:', e);
    }
}

// ============ Git Credentials ============

export async function loadGitCredentials() {
    try {
        const r = await fetch(`${API_BASE}/cicd/git/credentials`);
        const d = await r.json();
        gitCredentials = d.credentials || [];
        renderGitCredentialsTable();
    } catch (e) {
        console.error('Failed to load Git credentials:', e);
    }
}

function renderGitCredentialsTable() {
    const tbody = document.getElementById('git-credentials-tbody');
    if (!tbody) return;
    tbody.innerHTML = '';
    if (gitCredentials.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;color:#6a6a7a;padding:30px;">No Git credentials. Click "Add Credential" to add one.</td></tr>';
        return;
    }
    gitCredentials.forEach(c => {
        const statusIcon = getCredStatusIcon(c.test_status);
        const authIcon = c.auth_method === 'ssh_key' ? '&#128273;' : c.auth_method === 'token' ? '&#127919;' : '&#128274;';
        tbody.innerHTML += `<tr>
            <td><strong>${c.name}</strong></td>
            <td>${c.description || '-'}</td>
            <td title="${c.auth_method}">${authIcon} ${c.auth_method}</td>
            <td>${statusIcon}</td>
            <td class="action-cell">
                <button class="row-action-btn" style="background:rgba(34,211,238,0.2);color:#22d3ee;" onclick="testGitCredential('${c.id}')" title="Test Credential">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
                </button>
                <button class="row-action-btn describe" onclick="editGitCredential('${c.id}')" title="Edit">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/></svg>
                </button>
                <button class="row-action-btn delete" onclick="deleteGitCredential('${c.id}')" title="Delete">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
                </button>
            </td>
        </tr>`;
    });
}

function getCredStatusIcon(status) {
    switch (status) {
        case 'success': return '<span style="color:#4ade80;" title="Test Passed">&#10004;</span>';
        case 'failed': return '<span style="color:#ef4444;" title="Test Failed">&#10008;</span>';
        default: return '<span style="color:#6a6a7a;" title="Not Tested">&#8226;</span>';
    }
}

export function showAddGitCredentialModal() {
    document.getElementById('git-credential-modal').style.display = 'flex';
    document.getElementById('git-credential-modal-title').textContent = 'Add Git Credential';
    document.getElementById('git-credential-form').reset();
    document.getElementById('git-credential-id').value = '';
    updateGitAuthFields();
}

export function closeGitCredentialModal() {
    document.getElementById('git-credential-modal').style.display = 'none';
}

export function updateGitAuthFields() {
    const method = document.getElementById('git-credential-auth').value;
    document.getElementById('git-token-group').style.display = method === 'token' ? 'block' : 'none';
    document.getElementById('git-password-group').style.display = method === 'password' ? 'block' : 'none';
    document.getElementById('git-key-group').style.display = method === 'ssh_key' ? 'block' : 'none';
}

export async function saveGitCredential(e) {
    e.preventDefault();
    const id = document.getElementById('git-credential-id').value;
    const data = {
        name: document.getElementById('git-credential-name').value,
        description: document.getElementById('git-credential-desc').value,
        auth_method: document.getElementById('git-credential-auth').value
    };

    if (data.auth_method === 'token') {
        data.token = document.getElementById('git-credential-token').value;
    } else if (data.auth_method === 'password') {
        data.username = document.getElementById('git-credential-username').value;
        data.password = document.getElementById('git-credential-password').value;
    } else if (data.auth_method === 'ssh_key') {
        data.private_key = document.getElementById('git-credential-key').value;
        data.passphrase = document.getElementById('git-credential-passphrase').value;
    }

    try {
        const url = id ? `${API_BASE}/cicd/git/credentials/${id}` : `${API_BASE}/cicd/git/credentials`;
        const method = id ? 'PUT' : 'POST';
        const r = await fetch(url, {
            method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        const d = await r.json();
        if (d.error) {
            alert('Error: ' + d.error);
        } else {
            closeGitCredentialModal();
            loadGitCredentials();
        }
    } catch (e) {
        alert('Failed to save: ' + e.message);
    }
}

export async function editGitCredential(id) {
    const cred = gitCredentials.find(c => c.id === id);
    if (!cred) return;

    document.getElementById('git-credential-modal').style.display = 'flex';
    document.getElementById('git-credential-modal-title').textContent = 'Edit Git Credential';
    document.getElementById('git-credential-id').value = cred.id;
    document.getElementById('git-credential-name').value = cred.name;
    document.getElementById('git-credential-desc').value = cred.description || '';
    document.getElementById('git-credential-auth').value = cred.auth_method;
    updateGitAuthFields();
}

export async function deleteGitCredential(id) {
    if (!confirm('Delete this Git credential?')) return;
    try {
        const r = await fetch(`${API_BASE}/cicd/git/credentials/${id}`, { method: 'DELETE' });
        const d = await r.json();
        if (d.error) {
            alert('Error: ' + d.error);
        } else {
            loadGitCredentials();
        }
    } catch (e) {
        alert('Failed to delete: ' + e.message);
    }
}

export async function testGitCredential(id) {
    const url = prompt('Enter a Git repository URL to test with:', 'https://github.com/user/repo.git');
    if (!url) return;

    try {
        const r = await fetch(`${API_BASE}/cicd/git/credentials/${id}/test`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ url })
        });
        const d = await r.json();
        if (d.success) {
            alert('Credential test passed!');
        } else {
            alert('Credential test failed: ' + d.error);
        }
        loadGitCredentials();
    } catch (e) {
        alert('Test failed: ' + e.message);
    }
}

function renderSSHHostsTable() {
    const tbody = document.getElementById('ssh-hosts-tbody');
    tbody.innerHTML = '';
    if (sshHosts.length === 0) {
        tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;color:#6a6a7a;padding:30px;">No SSH hosts. Click "Add Host" to add one.</td></tr>';
        return;
    }
    sshHosts.forEach(h => {
        const statusIcon = getHostStatusIcon(h.test_status);
        const groups = h.host_groups && h.host_groups.length > 0 ? h.host_groups.join(', ') : '-';
        const authIcon = h.auth_method === 'key' ? '&#128273;' : '&#128274;';
        tbody.innerHTML += `<tr>
            <td><strong>${h.name}</strong></td>
            <td>${h.host}:${h.port}</td>
            <td>${h.username}</td>
            <td title="${h.auth_method}">${authIcon} ${h.auth_method}</td>
            <td style="color:#8a8a9a;">${groups}</td>
            <td>${statusIcon}</td>
            <td class="action-cell">
                <button class="row-action-btn" style="background:rgba(34,211,238,0.2);color:#22d3ee;" onclick="testSSHHost('${h.id}')" title="Test Connection">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
                </button>
                <button class="row-action-btn describe" onclick="editSSHHost('${h.id}')" title="Edit">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/></svg>
                </button>
                <button class="row-action-btn" style="background:rgba(167,139,250,0.2);color:#a78bfa;" onclick="cloneSSHHost('${h.id}')" title="Clone">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
                </button>
                <button class="row-action-btn delete" onclick="deleteSSHHost('${h.id}')" title="Delete">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
                </button>
            </td>
        </tr>`;
    });
}

function getHostStatusIcon(status) {
    switch (status) {
        case 'success': return '<span style="color:#4ade80;" title="Connection OK">&#10004;</span>';
        case 'failed': return '<span style="color:#ef4444;" title="Connection Failed">&#10008;</span>';
        default: return '<span style="color:#6a6a7a;" title="Not Tested">&#8226;</span>';
    }
}

export function showAddSSHHostModal() {
    document.getElementById('ssh-host-modal').style.display = 'flex';
    document.getElementById('ssh-host-modal-title').textContent = 'Add SSH Host';
    document.getElementById('ssh-host-form').reset();
    document.getElementById('ssh-host-id').value = '';
    document.getElementById('ssh-host-port').value = '22';
    updateAuthFields();
}

export function closeSSHHostModal() {
    document.getElementById('ssh-host-modal').style.display = 'none';
}

export function updateAuthFields() {
    const method = document.getElementById('ssh-host-auth').value;
    document.getElementById('ssh-password-group').style.display = method === 'password' ? 'block' : 'none';
    document.getElementById('ssh-key-group').style.display = method === 'key' ? 'block' : 'none';
}

export async function saveSSHHost(e) {
    e.preventDefault();
    const id = document.getElementById('ssh-host-id').value;
    const data = {
        name: document.getElementById('ssh-host-name').value,
        host: document.getElementById('ssh-host-host').value,
        port: parseInt(document.getElementById('ssh-host-port').value) || 22,
        username: document.getElementById('ssh-host-user').value,
        auth_method: document.getElementById('ssh-host-auth').value,
        host_groups: document.getElementById('ssh-host-groups').value.split(',').map(g => g.trim()).filter(g => g),
        description: document.getElementById('ssh-host-desc').value
    };

    if (data.auth_method === 'password') {
        data.password = document.getElementById('ssh-host-password').value;
    } else {
        data.private_key = document.getElementById('ssh-host-key').value;
        data.passphrase = document.getElementById('ssh-host-passphrase').value;
    }

    try {
        const url = id ? `${API_BASE}/cicd/ssh/hosts/${id}` : `${API_BASE}/cicd/ssh/hosts`;
        const method = id ? 'PUT' : 'POST';
        const r = await fetch(url, {
            method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        const d = await r.json();
        if (d.error) {
            alert('Error: ' + d.error);
        } else {
            closeSSHHostModal();
            loadSSHHosts();
        }
    } catch (e) {
        alert('Failed to save: ' + e.message);
    }
}

export async function editSSHHost(id) {
    const host = sshHosts.find(h => h.id === id);
    if (!host) return;

    document.getElementById('ssh-host-modal').style.display = 'flex';
    document.getElementById('ssh-host-modal-title').textContent = 'Edit SSH Host';
    document.getElementById('ssh-host-id').value = host.id;
    document.getElementById('ssh-host-name').value = host.name;
    document.getElementById('ssh-host-host').value = host.host;
    document.getElementById('ssh-host-port').value = host.port;
    document.getElementById('ssh-host-user').value = host.username;
    document.getElementById('ssh-host-auth').value = host.auth_method;
    document.getElementById('ssh-host-groups').value = (host.host_groups || []).join(', ');
    document.getElementById('ssh-host-desc').value = host.description || '';
    updateAuthFields();
}

export async function deleteSSHHost(id) {
    if (!confirm('Delete this SSH host? This cannot be undone.')) return;
    try {
        const r = await fetch(`${API_BASE}/cicd/ssh/hosts/${id}`, { method: 'DELETE' });
        const d = await r.json();
        if (d.error) {
            alert('Error: ' + d.error);
        } else {
            loadSSHHosts();
        }
    } catch (e) {
        alert('Failed to delete: ' + e.message);
    }
}

export async function cloneSSHHost(id) {
    const host = sshHosts.find(h => h.id === id);
    if (!host) return;

    const newName = prompt('Enter name for cloned SSH host:', host.name + '-copy');
    if (!newName) return;

    try {
        const cloneData = {
            name: newName,
            host: host.host,
            port: host.port,
            username: host.username,
            auth_method: host.auth_method,
            host_groups: host.host_groups || []
        };

        const r = await fetch(`${API_BASE}/cicd/ssh/hosts`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(cloneData)
        });
        const d = await r.json();
        if (d.error) {
            alert('Error cloning SSH host: ' + d.error);
        } else {
            alert(`SSH host "${newName}" created. Please edit it to set credentials.`);
            loadSSHHosts();
        }
    } catch (e) {
        alert('Failed to clone SSH host: ' + e.message);
    }
}

export async function testSSHHost(id) {
    const btn = event.target.closest('button');
    btn.disabled = true;
    btn.innerHTML = '&#8987;';
    try {
        const r = await fetch(`${API_BASE}/cicd/ssh/hosts/${id}/test`, { method: 'POST' });
        const d = await r.json();
        if (d.success) {
            alert('Connection test passed!');
        } else {
            alert('Connection test failed: ' + d.error);
        }
        loadSSHHosts();
    } catch (e) {
        alert('Test failed: ' + e.message);
    } finally {
        btn.disabled = false;
        btn.innerHTML = '<svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>';
    }
}

// ============ Freestyle Jobs ============

export async function loadFreestyleJobs() {
    try {
        const r = await fetch(`${API_BASE}/cicd/freestyle/jobs`);
        const d = await r.json();
        freestyleJobs = d.jobs || [];
        renderFreestyleJobsTable();
    } catch (e) {
        console.error('Failed to load freestyle jobs:', e);
    }
}

function renderFreestyleJobsTable() {
    const tbody = document.getElementById('freestyle-jobs-tbody');
    tbody.innerHTML = '';
    if (freestyleJobs.length === 0) {
        tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;color:#6a6a7a;padding:30px;">No freestyle jobs. Click "Create Job" to add one.</td></tr>';
        return;
    }
    freestyleJobs.forEach(j => {
        const statusClass = j.status.last_status ? getRunStatusClass(j.status.last_status) : '';
        const lastBuild = j.status.last_build_at ? formatTime(j.status.last_build_at) : 'Never';
        const triggers = getFreestyleTriggerIcons(j.triggers || []);
        tbody.innerHTML += `<tr>
            <td><strong>${j.name}</strong></td>
            <td style="color:#8a8a9a;">${j.description || '-'}</td>
            <td>${j.build_steps ? j.build_steps.length : 0} steps</td>
            <td class="${statusClass}">${j.status.last_status || '-'}</td>
            <td>${lastBuild}</td>
            <td>${triggers}</td>
            <td class="action-cell">
                <button class="row-action-btn" style="background:rgba(74,222,128,0.2);color:#4ade80;" onclick="runFreestyleJob('${j.id}')" title="Run">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"/><path stroke-linecap="round" stroke-linejoin="round" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
                </button>
                <button class="row-action-btn describe" onclick="editFreestyleJob('${j.id}')" title="Edit">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/></svg>
                </button>
                <button class="row-action-btn" style="background:rgba(167,139,250,0.2);color:#a78bfa;" onclick="cloneFreestyleJob('${j.id}')" title="Clone">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
                </button>
                <button class="row-action-btn" style="background:rgba(251,191,36,0.2);color:#fbbf24;" onclick="copyFreestyleJobBadge('${j.id}', '${j.name}')" title="Copy Badge">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z"/></svg>
                </button>
                <button class="row-action-btn logs" onclick="viewFreestyleJobBuilds('${j.id}')" title="Builds">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M4 6h16M4 12h16M4 18h7"/></svg>
                </button>
                <button class="row-action-btn delete" onclick="deleteFreestyleJob('${j.id}')" title="Delete">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
                </button>
            </td>
        </tr>`;
    });
}

function getFreestyleTriggerIcons(triggers) {
    if (!triggers || triggers.length === 0) return '<span style="color:#6a6a7a;">Manual only</span>';
    let icons = '';
    triggers.forEach(t => {
        if (t.type === 'webhook' && t.enabled) {
            icons += '<span title="Webhook" style="color:#22d3ee;margin-right:8px;">&#128279;</span>';
        } else if (t.type === 'cron' && t.enabled) {
            icons += `<span title="Cron: ${t.schedule}" style="color:#a78bfa;margin-right:8px;">&#9200;</span>`;
        }
    });
    return icons || '<span style="color:#6a6a7a;">Manual only</span>';
}

// ============ Improved Job Modal ============

export async function showCreateFreestyleJobModal() {
    editingJob = null;
    buildSteps = [];
    jobParameters = [];
    envVariables = [];
    currentTab = 'general';
    jobName = '';
    jobDesc = '';
    jobEnabled = true;
    webhookEnabled = false;
    cronEnabled = false;
    cronSchedule = '';

    // Reset SCM state
    scmType = 'none';
    scmRepositories = [];
    scmBranches = [];
    scmCloneDepth = 0;
    scmSubmodules = false;
    scmCleanBefore = false;

    await loadSSHHostsForSelect();
    await loadGitCredentialsForSelect();
    renderJobModal();
    document.getElementById('freestyle-job-modal').style.display = 'flex';
}

export function closeFreestyleJobModal() {
    document.getElementById('freestyle-job-modal').style.display = 'none';
}

async function loadSSHHostsForSelect() {
    if (sshHosts.length === 0) {
        await loadSSHHosts();
    }
}

async function loadGitCredentialsForSelect() {
    if (gitCredentials.length === 0) {
        await loadGitCredentials();
    }
}

function renderJobModal() {
    const modal = document.getElementById('freestyle-job-modal');
    const modalContent = modal.querySelector('.k8s-modal');

    // Update title
    document.getElementById('freestyle-job-modal-title').textContent = editingJob ? 'Edit Freestyle Job' : 'Create Freestyle Job';

    // Render modal body with tabs
    const body = modal.querySelector('.k8s-modal-body');
    body.innerHTML = `
        <!-- Tab Navigation -->
        <div class="job-tabs" style="display:flex;gap:4px;border-bottom:1px solid rgba(255,255,255,0.1);margin-bottom:20px;padding-bottom:0;">
            <button type="button" class="job-tab ${currentTab === 'general' ? 'active' : ''}" onclick="switchJobTab('general')" style="padding:10px 20px;background:${currentTab === 'general' ? 'rgba(34,211,238,0.2)' : 'transparent'};border:none;border-bottom:2px solid ${currentTab === 'general' ? '#22d3ee' : 'transparent'};color:${currentTab === 'general' ? '#22d3ee' : '#8a8a9a'};cursor:pointer;font-size:13px;font-weight:500;transition:all 0.2s;">
                General
            </button>
            <button type="button" class="job-tab ${currentTab === 'scm' ? 'active' : ''}" onclick="switchJobTab('scm')" style="padding:10px 20px;background:${currentTab === 'scm' ? 'rgba(34,211,238,0.2)' : 'transparent'};border:none;border-bottom:2px solid ${currentTab === 'scm' ? '#22d3ee' : 'transparent'};color:${currentTab === 'scm' ? '#22d3ee' : '#8a8a9a'};cursor:pointer;font-size:13px;font-weight:500;transition:all 0.2s;">
                Source Code ${scmType === 'git' ? '<span style="background:rgba(74,222,128,0.2);color:#4ade80;padding:2px 6px;border-radius:10px;font-size:10px;margin-left:4px;">Git</span>' : ''}
            </button>
            <button type="button" class="job-tab ${currentTab === 'steps' ? 'active' : ''}" onclick="switchJobTab('steps')" style="padding:10px 20px;background:${currentTab === 'steps' ? 'rgba(34,211,238,0.2)' : 'transparent'};border:none;border-bottom:2px solid ${currentTab === 'steps' ? '#22d3ee' : 'transparent'};color:${currentTab === 'steps' ? '#22d3ee' : '#8a8a9a'};cursor:pointer;font-size:13px;font-weight:500;transition:all 0.2s;">
                Build Steps <span style="background:rgba(255,255,255,0.1);padding:2px 8px;border-radius:10px;font-size:11px;margin-left:6px;">${buildSteps.length}</span>
            </button>
            <button type="button" class="job-tab ${currentTab === 'params' ? 'active' : ''}" onclick="switchJobTab('params')" style="padding:10px 20px;background:${currentTab === 'params' ? 'rgba(34,211,238,0.2)' : 'transparent'};border:none;border-bottom:2px solid ${currentTab === 'params' ? '#22d3ee' : 'transparent'};color:${currentTab === 'params' ? '#22d3ee' : '#8a8a9a'};cursor:pointer;font-size:13px;font-weight:500;transition:all 0.2s;">
                Parameters <span style="background:rgba(255,255,255,0.1);padding:2px 8px;border-radius:10px;font-size:11px;margin-left:6px;">${jobParameters.length}</span>
            </button>
            <button type="button" class="job-tab ${currentTab === 'triggers' ? 'active' : ''}" onclick="switchJobTab('triggers')" style="padding:10px 20px;background:${currentTab === 'triggers' ? 'rgba(34,211,238,0.2)' : 'transparent'};border:none;border-bottom:2px solid ${currentTab === 'triggers' ? '#22d3ee' : 'transparent'};color:${currentTab === 'triggers' ? '#22d3ee' : '#8a8a9a'};cursor:pointer;font-size:13px;font-weight:500;transition:all 0.2s;">
                Triggers
            </button>
        </div>

        <!-- Tab Content -->
        <div class="job-tab-content" style="min-height:400px;">
            ${renderCurrentTabContent()}
        </div>
    `;
}

function renderCurrentTabContent() {
    switch (currentTab) {
        case 'general': return renderGeneralTab();
        case 'scm': return renderSCMTab();
        case 'steps': return renderStepsTab();
        case 'params': return renderParamsTab();
        case 'triggers': return renderTriggersTab();
        default: return renderGeneralTab();
    }
}

export function switchJobTab(tab) {
    // Save current tab data before switching
    saveCurrentTabData();
    currentTab = tab;
    renderJobModal();
}

function saveCurrentTabData() {
    if (currentTab === 'general') {
        const nameEl = document.getElementById('freestyle-job-name');
        const descEl = document.getElementById('freestyle-job-desc');
        const enabledEl = document.getElementById('freestyle-job-enabled');
        if (nameEl) jobName = nameEl.value;
        if (descEl) jobDesc = descEl.value;
        if (enabledEl) jobEnabled = enabledEl.checked;
    } else if (currentTab === 'scm') {
        const scmTypeEl = document.querySelector('input[name="scm-type"]:checked');
        const cloneDepthEl = document.getElementById('scm-clone-depth');
        const submodulesEl = document.getElementById('scm-submodules');
        const cleanBeforeEl = document.getElementById('scm-clean-before');
        if (scmTypeEl) scmType = scmTypeEl.value;
        if (cloneDepthEl) scmCloneDepth = parseInt(cloneDepthEl.value) || 0;
        if (submodulesEl) scmSubmodules = submodulesEl.checked;
        if (cleanBeforeEl) scmCleanBefore = cleanBeforeEl.checked;
    } else if (currentTab === 'triggers') {
        const webhookEl = document.getElementById('freestyle-job-webhook');
        const cronEnabledEl = document.getElementById('freestyle-job-cron-enabled');
        const cronScheduleEl = document.getElementById('freestyle-job-cron');
        if (webhookEl) webhookEnabled = webhookEl.checked;
        if (cronEnabledEl) cronEnabled = cronEnabledEl.checked;
        if (cronScheduleEl) cronSchedule = cronScheduleEl.value;
    }
}

// ============ General Tab ============

function renderGeneralTab() {
    return `
        <div class="form-group" style="margin-bottom:20px;">
            <label style="display:block;margin-bottom:8px;font-weight:500;color:#e0e0e0;">
                Job Name <span style="color:#ef4444;">*</span>
            </label>
            <input type="text" id="freestyle-job-name" required placeholder="my-build-job" value="${jobName}"
                style="width:100%;padding:12px;background:rgba(20,20,30,0.8);border:1px solid rgba(255,255,255,0.15);border-radius:6px;color:#fff;font-size:14px;">
            <small style="color:#6a6a7a;font-size:11px;margin-top:4px;display:block;">Unique identifier for this job</small>
        </div>

        <div class="form-group" style="margin-bottom:20px;">
            <label style="display:block;margin-bottom:8px;font-weight:500;color:#e0e0e0;">Description</label>
            <input type="text" id="freestyle-job-desc" placeholder="Brief description of what this job does" value="${jobDesc}"
                style="width:100%;padding:12px;background:rgba(20,20,30,0.8);border:1px solid rgba(255,255,255,0.15);border-radius:6px;color:#fff;font-size:14px;">
        </div>

        <div class="form-group" style="margin-bottom:24px;">
            <label style="display:flex;align-items:center;gap:10px;cursor:pointer;">
                <input type="checkbox" id="freestyle-job-enabled" ${jobEnabled ? 'checked' : ''}
                    style="width:18px;height:18px;accent-color:#22d3ee;">
                <span style="font-weight:500;color:#e0e0e0;">Enabled</span>
            </label>
            <small style="color:#6a6a7a;font-size:11px;margin-top:4px;display:block;margin-left:28px;">Disabled jobs cannot be triggered</small>
        </div>

        <div style="border-top:1px solid rgba(255,255,255,0.1);padding-top:20px;">
            <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:12px;">
                <label style="font-weight:600;color:#e0e0e0;">Environment Variables</label>
                <button type="button" onclick="addEnvVariable()"
                    style="padding:6px 12px;font-size:12px;background:rgba(34,211,238,0.15);border:1px solid rgba(34,211,238,0.3);border-radius:4px;color:#22d3ee;cursor:pointer;">
                    + Add Variable
                </button>
            </div>
            <div id="env-variables-list" style="background:rgba(20,20,30,0.5);border-radius:8px;overflow:hidden;">
                ${renderEnvVariables()}
            </div>
        </div>
    `;
}

function renderEnvVariables() {
    if (envVariables.length === 0) {
        return `<div style="padding:24px;text-align:center;color:#6a6a7a;font-size:13px;">
            No environment variables defined.<br>
            <small style="color:#4a4a5a;">Click "Add Variable" to define environment variables available to all steps.</small>
        </div>`;
    }

    let html = `<table style="width:100%;border-collapse:collapse;">
        <thead>
            <tr style="background:rgba(255,255,255,0.05);">
                <th style="text-align:left;padding:10px 12px;font-size:11px;font-weight:600;color:#8a8a9a;text-transform:uppercase;">Key</th>
                <th style="text-align:left;padding:10px 12px;font-size:11px;font-weight:600;color:#8a8a9a;text-transform:uppercase;">Value</th>
                <th style="width:50px;"></th>
            </tr>
        </thead>
        <tbody>`;

    envVariables.forEach((env, i) => {
        html += `<tr style="border-top:1px solid rgba(255,255,255,0.05);">
            <td style="padding:8px 12px;">
                <input type="text" value="${env.key}" onchange="updateEnvVariable(${i}, 'key', this.value)" placeholder="KEY"
                    style="width:100%;padding:8px;background:rgba(20,20,30,0.8);border:1px solid rgba(255,255,255,0.1);border-radius:4px;color:#fff;font-size:13px;font-family:monospace;">
            </td>
            <td style="padding:8px 12px;">
                <input type="text" value="${env.value}" onchange="updateEnvVariable(${i}, 'value', this.value)" placeholder="value"
                    style="width:100%;padding:8px;background:rgba(20,20,30,0.8);border:1px solid rgba(255,255,255,0.1);border-radius:4px;color:#fff;font-size:13px;font-family:monospace;">
            </td>
            <td style="padding:8px 12px;text-align:center;">
                <button type="button" onclick="removeEnvVariable(${i})" title="Remove"
                    style="background:rgba(239,68,68,0.2);border:none;color:#ef4444;padding:6px 8px;border-radius:4px;cursor:pointer;">
                    &#10005;
                </button>
            </td>
        </tr>`;
    });

    html += `</tbody></table>`;
    return html;
}

export function addEnvVariable() {
    envVariables.push({ key: '', value: '' });
    renderJobModal();
}

export function updateEnvVariable(index, field, value) {
    if (envVariables[index]) {
        envVariables[index][field] = value;
    }
}

export function removeEnvVariable(index) {
    envVariables.splice(index, 1);
    renderJobModal();
}

// ============ SCM (Source Code) Tab ============

function renderSCMTab() {
    // Build credential dropdown options
    const credOptions = gitCredentials.map(c =>
        `<option value="${c.id}">${c.name} (${c.auth_method})</option>`
    ).join('');

    // Render repositories
    let reposHtml = '';
    if (scmRepositories.length === 0) {
        reposHtml = `<div style="padding:20px;text-align:center;color:#6a6a7a;font-size:13px;">
            No repositories configured.
        </div>`;
    } else {
        reposHtml = scmRepositories.map((repo, i) => `
            <div style="padding:12px;border-bottom:1px solid rgba(255,255,255,0.05);">
                <div style="display:flex;gap:10px;align-items:flex-start;">
                    <div style="flex:1;">
                        <label style="display:block;font-size:11px;color:#8a8a9a;margin-bottom:4px;">Repository URL</label>
                        <input type="text" value="${repo.url || ''}" placeholder="https://github.com/user/repo.git"
                            onchange="updateSCMRepo(${i}, 'url', this.value)"
                            style="width:100%;padding:8px;background:rgba(20,20,30,0.8);border:1px solid rgba(255,255,255,0.15);border-radius:4px;color:#fff;font-size:13px;">
                    </div>
                    <div style="width:200px;">
                        <label style="display:block;font-size:11px;color:#8a8a9a;margin-bottom:4px;">Credentials</label>
                        <select onchange="updateSCMRepo(${i}, 'credential_id', this.value)"
                            style="width:100%;padding:8px;background:rgba(20,20,30,0.8);border:1px solid rgba(255,255,255,0.15);border-radius:4px;color:#fff;font-size:13px;">
                            <option value="">- none (public) -</option>
                            ${credOptions}
                        </select>
                    </div>
                    <button type="button" onclick="removeSCMRepo(${i})" title="Remove"
                        style="margin-top:20px;padding:8px;background:rgba(239,68,68,0.2);border:none;border-radius:4px;color:#ef4444;cursor:pointer;">
                        &#10006;
                    </button>
                </div>
            </div>
        `).join('');
    }

    // Render branches
    let branchesHtml = '';
    if (scmBranches.length === 0) {
        branchesHtml = `<div style="padding:12px;text-align:center;color:#6a6a7a;font-size:12px;">
            No branch specifier (will use default branch)
        </div>`;
    } else {
        branchesHtml = scmBranches.map((br, i) => `
            <div style="display:flex;gap:8px;align-items:center;padding:8px;">
                <input type="text" value="${br.specifier || ''}" placeholder="*/main"
                    onchange="updateSCMBranch(${i}, this.value)"
                    style="flex:1;padding:8px;background:rgba(20,20,30,0.8);border:1px solid rgba(255,255,255,0.15);border-radius:4px;color:#fff;font-size:13px;">
                <button type="button" onclick="removeSCMBranch(${i})" title="Remove"
                    style="padding:8px;background:rgba(239,68,68,0.2);border:none;border-radius:4px;color:#ef4444;cursor:pointer;">
                    &#10006;
                </button>
            </div>
        `).join('');
    }

    return `
        <div style="margin-bottom:24px;">
            <label style="display:block;font-weight:600;color:#e0e0e0;margin-bottom:12px;">Source Code Management</label>
            <div style="display:flex;gap:20px;padding:16px;background:rgba(20,20,30,0.5);border-radius:8px;">
                <label style="display:flex;align-items:center;gap:8px;cursor:pointer;">
                    <input type="radio" name="scm-type" value="none" ${scmType === 'none' ? 'checked' : ''} onchange="setSCMType('none')"
                        style="width:18px;height:18px;accent-color:#22d3ee;">
                    <span style="color:#e0e0e0;">None</span>
                </label>
                <label style="display:flex;align-items:center;gap:8px;cursor:pointer;">
                    <input type="radio" name="scm-type" value="git" ${scmType === 'git' ? 'checked' : ''} onchange="setSCMType('git')"
                        style="width:18px;height:18px;accent-color:#22d3ee;">
                    <span style="color:#e0e0e0;">Git</span>
                </label>
            </div>
        </div>

        <div id="scm-git-config" style="display:${scmType === 'git' ? 'block' : 'none'};">
            <!-- Repositories Section -->
            <div style="margin-bottom:20px;border:1px solid rgba(255,255,255,0.1);border-radius:8px;overflow:hidden;">
                <div style="display:flex;justify-content:space-between;align-items:center;padding:12px 16px;background:rgba(255,255,255,0.05);">
                    <span style="font-weight:600;color:#e0e0e0;">Repositories</span>
                    <button type="button" onclick="addSCMRepo()"
                        style="padding:6px 12px;font-size:12px;background:rgba(34,211,238,0.15);border:1px solid rgba(34,211,238,0.3);border-radius:4px;color:#22d3ee;cursor:pointer;">
                        + Add Repository
                    </button>
                </div>
                <div>${reposHtml}</div>
            </div>

            <!-- Branches Section -->
            <div style="margin-bottom:20px;border:1px solid rgba(255,255,255,0.1);border-radius:8px;overflow:hidden;">
                <div style="display:flex;justify-content:space-between;align-items:center;padding:12px 16px;background:rgba(255,255,255,0.05);">
                    <span style="font-weight:600;color:#e0e0e0;">Branches to Build</span>
                    <button type="button" onclick="addSCMBranch()"
                        style="padding:6px 12px;font-size:12px;background:rgba(34,211,238,0.15);border:1px solid rgba(34,211,238,0.3);border-radius:4px;color:#22d3ee;cursor:pointer;">
                        + Add Branch
                    </button>
                </div>
                <div>${branchesHtml}</div>
                <div style="padding:8px 12px;background:rgba(20,20,30,0.3);font-size:11px;color:#6a6a7a;">
                    Branch specifiers: <code style="background:rgba(0,0,0,0.3);padding:2px 4px;border-radius:3px;">*/main</code>,
                    <code style="background:rgba(0,0,0,0.3);padding:2px 4px;border-radius:3px;">*/develop</code>,
                    <code style="background:rgba(0,0,0,0.3);padding:2px 4px;border-radius:3px;">refs/heads/*</code>
                </div>
            </div>

            <!-- Advanced Options -->
            <details style="margin-bottom:16px;">
                <summary style="cursor:pointer;padding:12px;background:rgba(255,255,255,0.05);border-radius:8px;color:#e0e0e0;font-weight:500;">
                    &#9654; Advanced Options
                </summary>
                <div style="padding:16px;background:rgba(20,20,30,0.5);border-radius:0 0 8px 8px;border:1px solid rgba(255,255,255,0.1);border-top:none;">
                    <div style="margin-bottom:16px;">
                        <label style="display:flex;align-items:center;gap:10px;cursor:pointer;">
                            <input type="checkbox" id="scm-clean-before" ${scmCleanBefore ? 'checked' : ''}
                                style="width:18px;height:18px;accent-color:#22d3ee;">
                            <span style="color:#e0e0e0;">Clean workspace before checkout</span>
                        </label>
                        <small style="color:#6a6a7a;font-size:11px;margin-left:28px;display:block;margin-top:4px;">
                            Delete existing workspace directory before cloning
                        </small>
                    </div>
                    <div style="margin-bottom:16px;">
                        <label style="display:flex;align-items:center;gap:10px;cursor:pointer;">
                            <input type="checkbox" id="scm-submodules" ${scmSubmodules ? 'checked' : ''}
                                style="width:18px;height:18px;accent-color:#22d3ee;">
                            <span style="color:#e0e0e0;">Clone submodules</span>
                        </label>
                        <small style="color:#6a6a7a;font-size:11px;margin-left:28px;display:block;margin-top:4px;">
                            Recursively clone Git submodules
                        </small>
                    </div>
                    <div>
                        <label style="display:block;margin-bottom:8px;color:#e0e0e0;">Clone Depth</label>
                        <input type="number" id="scm-clone-depth" value="${scmCloneDepth}" min="0" placeholder="0 (full clone)"
                            style="width:120px;padding:8px;background:rgba(20,20,30,0.8);border:1px solid rgba(255,255,255,0.15);border-radius:4px;color:#fff;font-size:13px;">
                        <small style="color:#6a6a7a;font-size:11px;display:block;margin-top:4px;">
                            0 = full clone, otherwise shallow clone with specified depth
                        </small>
                    </div>
                </div>
            </details>
        </div>
    `;
}

export function setSCMType(type) {
    scmType = type;
    if (type === 'git' && scmRepositories.length === 0) {
        // Add one empty repository when switching to Git
        scmRepositories.push({ url: '', credential_id: '' });
    }
    renderJobModal();
}

export function addSCMRepo() {
    scmRepositories.push({ url: '', credential_id: '' });
    renderJobModal();
}

export function updateSCMRepo(index, field, value) {
    if (scmRepositories[index]) {
        scmRepositories[index][field] = value;
    }
}

export function removeSCMRepo(index) {
    scmRepositories.splice(index, 1);
    renderJobModal();
}

export function addSCMBranch() {
    scmBranches.push({ specifier: '' });
    renderJobModal();
}

export function updateSCMBranch(index, value) {
    if (scmBranches[index]) {
        scmBranches[index].specifier = value;
    }
}

export function removeSCMBranch(index) {
    scmBranches.splice(index, 1);
    renderJobModal();
}

// ============ Build Steps Tab ============

function renderStepsTab() {
    const hostOptions = sshHosts.map(h =>
        `<option value="${h.id}">${h.name} (${h.host})</option>`
    ).join('');

    let stepsHtml = '';

    if (buildSteps.length === 0) {
        stepsHtml = `<div style="padding:40px;text-align:center;color:#6a6a7a;background:rgba(20,20,30,0.5);border-radius:8px;border:2px dashed rgba(255,255,255,0.1);">
            <div style="font-size:40px;margin-bottom:10px;">📋</div>
            <div style="font-size:14px;margin-bottom:8px;">No build steps defined</div>
            <div style="font-size:12px;color:#4a4a5a;">Click "Add Step" to create your first build step</div>
        </div>`;
    } else {
        buildSteps.forEach((step, i) => {
            const typeLabels = {
                'shell': '💻 Shell Command',
                'script': '📜 Script',
                'scp_push': '📤 SCP Push (to remote)',
                'scp_pull': '📥 SCP Pull (from remote)'
            };

            const selectedHost = sshHosts.find(h => h.id === step.host_id);
            const hostName = selectedHost ? selectedHost.name : 'Select host...';

            stepsHtml += `
            <div class="build-step-card" style="background:rgba(30,30,40,0.5);border:1px solid rgba(255,255,255,0.1);border-radius:8px;margin-bottom:12px;overflow:hidden;">
                <!-- Step Header -->
                <div style="display:flex;justify-content:space-between;align-items:center;padding:12px 16px;background:rgba(255,255,255,0.03);border-bottom:1px solid rgba(255,255,255,0.05);">
                    <div style="display:flex;align-items:center;gap:12px;">
                        <span style="background:rgba(34,211,238,0.2);color:#22d3ee;padding:4px 10px;border-radius:4px;font-size:12px;font-weight:600;">Step ${i + 1}</span>
                        <div style="display:flex;align-items:center;gap:8px;">
                            <svg fill="none" stroke="#6a6a7a" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"/></svg>
                            <input type="text" value="${step.name || ''}" onchange="updateStep(${i}, 'name', this.value)" placeholder="Enter step name..."
                                style="background:rgba(20,20,30,0.6);border:1px solid rgba(255,255,255,0.1);border-radius:4px;color:#fff;font-size:13px;font-weight:500;padding:6px 10px;width:200px;">
                        </div>
                    </div>
                    <div style="display:flex;gap:6px;">
                        <button type="button" onclick="moveStepUp(${i})" title="Move up" ${i === 0 ? 'disabled' : ''}
                            style="background:rgba(255,255,255,0.05);border:none;color:#8a8a9a;padding:6px 8px;border-radius:4px;cursor:${i === 0 ? 'not-allowed' : 'pointer'};opacity:${i === 0 ? '0.5' : '1'};">▲</button>
                        <button type="button" onclick="moveStepDown(${i})" title="Move down" ${i === buildSteps.length - 1 ? 'disabled' : ''}
                            style="background:rgba(255,255,255,0.05);border:none;color:#8a8a9a;padding:6px 8px;border-radius:4px;cursor:${i === buildSteps.length - 1 ? 'not-allowed' : 'pointer'};opacity:${i === buildSteps.length - 1 ? '0.5' : '1'};">▼</button>
                        <button type="button" onclick="removeStep(${i})" title="Remove step"
                            style="background:rgba(239,68,68,0.2);border:none;color:#ef4444;padding:6px 10px;border-radius:4px;cursor:pointer;">&#10005;</button>
                    </div>
                </div>

                <!-- Step Configuration -->
                <div style="padding:16px;">
                    <div style="display:grid;grid-template-columns:1fr 1fr;gap:16px;margin-bottom:16px;">
                        <div>
                            <label style="display:block;font-size:11px;color:#8a8a9a;margin-bottom:6px;text-transform:uppercase;font-weight:600;">Host</label>
                            <select onchange="updateStep(${i}, 'host_id', this.value)"
                                style="width:100%;padding:10px;background:rgba(20,20,30,0.8);border:1px solid rgba(255,255,255,0.15);border-radius:6px;color:#fff;font-size:13px;">
                                <option value="local" ${step.host_id === 'local' || step.host_id === '' ? 'selected' : ''}>🖥️ Local (container)</option>
                                ${sshHosts.map(h => `<option value="${h.id}" ${step.host_id === h.id ? 'selected' : ''}>${h.name} (${h.host})</option>`).join('')}
                            </select>
                            <small style="color:#6a6a7a;font-size:11px;margin-top:4px;display:block;">${step.host_id && step.host_id !== 'local' ? 'Execute on remote SSH host' : 'Execute inside GAGOS container'}</small>
                        </div>
                        <div>
                            <label style="display:block;font-size:11px;color:#8a8a9a;margin-bottom:6px;text-transform:uppercase;font-weight:600;">Type</label>
                            <select onchange="updateStep(${i}, 'type', this.value)"
                                style="width:100%;padding:10px;background:rgba(20,20,30,0.8);border:1px solid rgba(255,255,255,0.15);border-radius:6px;color:#fff;font-size:13px;">
                                <option value="shell" ${step.type === 'shell' ? 'selected' : ''}>💻 Shell Command</option>
                                <option value="script" ${step.type === 'script' ? 'selected' : ''}>📜 Script</option>
                                ${step.host_id && step.host_id !== 'local' ? `
                                <option value="scp_push" ${step.type === 'scp_push' ? 'selected' : ''}>📤 SCP Push (to remote)</option>
                                <option value="scp_pull" ${step.type === 'scp_pull' ? 'selected' : ''}>📥 SCP Pull (from remote)</option>
                                ` : ''}
                            </select>
                        </div>
                    </div>

                    ${step.type === 'scp_push' || step.type === 'scp_pull' ? `
                    <div style="display:grid;grid-template-columns:1fr 1fr;gap:16px;margin-bottom:16px;">
                        <div>
                            <label style="display:block;font-size:11px;color:#8a8a9a;margin-bottom:6px;text-transform:uppercase;font-weight:600;">Local Path</label>
                            <input type="text" value="${step.local_path || ''}" onchange="updateStep(${i}, 'local_path', this.value)" placeholder="/local/path/file.txt"
                                style="width:100%;padding:10px;background:rgba(20,20,30,0.8);border:1px solid rgba(255,255,255,0.15);border-radius:6px;color:#fff;font-size:13px;font-family:monospace;">
                        </div>
                        <div>
                            <label style="display:block;font-size:11px;color:#8a8a9a;margin-bottom:6px;text-transform:uppercase;font-weight:600;">Remote Path</label>
                            <input type="text" value="${step.remote_path || ''}" onchange="updateStep(${i}, 'remote_path', this.value)" placeholder="/remote/path/file.txt"
                                style="width:100%;padding:10px;background:rgba(20,20,30,0.8);border:1px solid rgba(255,255,255,0.15);border-radius:6px;color:#fff;font-size:13px;font-family:monospace;">
                        </div>
                    </div>
                    ` : `
                    <div style="margin-bottom:16px;">
                        <label style="display:block;font-size:11px;color:#8a8a9a;margin-bottom:6px;text-transform:uppercase;font-weight:600;">${step.type === 'script' ? 'Script Content' : 'Command'}</label>
                        <textarea onchange="updateStep(${i}, '${step.type === 'script' ? 'script' : 'command'}', this.value)" placeholder="${step.type === 'script' ? '#!/bin/bash\necho \"Hello World\"' : 'echo \"Hello World\"'}"
                            style="width:100%;height:100px;padding:10px;background:rgba(20,20,30,0.8);border:1px solid rgba(255,255,255,0.15);border-radius:6px;color:#fff;font-size:13px;font-family:monospace;resize:vertical;">${step.type === 'script' ? (step.script || '') : (step.command || '')}</textarea>
                    </div>
                    `}

                    <div style="display:flex;gap:24px;align-items:center;">
                        <div style="display:flex;align-items:center;gap:8px;">
                            <label style="font-size:12px;color:#8a8a9a;">Timeout:</label>
                            <input type="number" value="${step.timeout || 300}" onchange="updateStep(${i}, 'timeout', parseInt(this.value))"
                                style="width:80px;padding:6px 10px;background:rgba(20,20,30,0.8);border:1px solid rgba(255,255,255,0.15);border-radius:4px;color:#fff;font-size:13px;">
                            <span style="font-size:12px;color:#6a6a7a;">seconds</span>
                        </div>
                        <label style="display:flex;align-items:center;gap:8px;cursor:pointer;">
                            <input type="checkbox" ${step.continue_on_error ? 'checked' : ''} onchange="updateStep(${i}, 'continue_on_error', this.checked)"
                                style="width:16px;height:16px;accent-color:#22d3ee;">
                            <span style="font-size:12px;color:#8a8a9a;">Continue on error</span>
                        </label>
                    </div>
                </div>
            </div>`;
        });
    }

    return `
        <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:16px;">
            <div>
                <span style="font-size:13px;color:#6a6a7a;">${buildSteps.length} step${buildSteps.length !== 1 ? 's' : ''} configured</span>
            </div>
            <button type="button" onclick="addBuildStep()"
                style="padding:10px 20px;font-size:13px;background:rgba(34,211,238,0.2);border:1px solid rgba(34,211,238,0.3);border-radius:6px;color:#22d3ee;cursor:pointer;font-weight:500;">
                + Add Step
            </button>
        </div>
        <div id="build-steps-container" style="max-height:400px;overflow-y:auto;">
            ${stepsHtml}
        </div>
    `;
}

export function addBuildStep() {
    buildSteps.push({
        name: `Step ${buildSteps.length + 1}`,
        type: 'shell',
        host_id: 'local',  // Default to local execution
        command: '',
        timeout: 300,
        continue_on_error: false
    });
    renderJobModal();
}

export function updateStep(index, field, value) {
    if (buildSteps[index]) {
        buildSteps[index][field] = value;

        // When host changes, re-render to update available step types
        if (field === 'host_id') {
            // If switching to local, reset SCP types to shell (SCP requires remote host)
            if (value === 'local' || value === '') {
                const currentType = buildSteps[index].type;
                if (currentType === 'scp_push' || currentType === 'scp_pull') {
                    buildSteps[index].type = 'shell';
                }
            }
            renderJobModal();
        }

        // Re-render if type changed to update UI
        if (field === 'type') {
            renderJobModal();
        }
    }
}

export function removeStep(index) {
    buildSteps.splice(index, 1);
    renderJobModal();
}

export function moveStepUp(index) {
    if (index > 0) {
        [buildSteps[index], buildSteps[index - 1]] = [buildSteps[index - 1], buildSteps[index]];
        renderJobModal();
    }
}

export function moveStepDown(index) {
    if (index < buildSteps.length - 1) {
        [buildSteps[index], buildSteps[index + 1]] = [buildSteps[index + 1], buildSteps[index]];
        renderJobModal();
    }
}

// ============ Parameters Tab ============

function renderParamsTab() {
    let paramsHtml = '';

    if (jobParameters.length === 0) {
        paramsHtml = `<div style="padding:40px;text-align:center;color:#6a6a7a;background:rgba(20,20,30,0.5);border-radius:8px;border:2px dashed rgba(255,255,255,0.1);">
            <div style="font-size:40px;margin-bottom:10px;">📝</div>
            <div style="font-size:14px;margin-bottom:8px;">No parameters defined</div>
            <div style="font-size:12px;color:#4a4a5a;">Parameters allow users to provide input when triggering a build</div>
        </div>`;
    } else {
        jobParameters.forEach((param, i) => {
            paramsHtml += `
            <div style="background:rgba(30,30,40,0.5);border:1px solid rgba(255,255,255,0.1);border-radius:8px;margin-bottom:12px;padding:16px;">
                <div style="display:flex;justify-content:space-between;align-items:start;margin-bottom:12px;">
                    <div style="flex:1;margin-right:16px;">
                        <label style="display:block;font-size:11px;color:#8a8a9a;margin-bottom:6px;text-transform:uppercase;font-weight:600;">Parameter Name</label>
                        <input type="text" value="${param.name || ''}" onchange="updateParameter(${i}, 'name', this.value)" placeholder="VERSION"
                            style="width:100%;padding:10px;background:rgba(20,20,30,0.8);border:1px solid rgba(255,255,255,0.15);border-radius:6px;color:#fff;font-size:13px;font-family:monospace;">
                    </div>
                    <button type="button" onclick="removeParameter(${i})" title="Remove"
                        style="background:rgba(239,68,68,0.2);border:none;color:#ef4444;padding:8px 12px;border-radius:4px;cursor:pointer;margin-top:20px;">&#10005;</button>
                </div>

                <div style="display:grid;grid-template-columns:1fr 1fr;gap:16px;margin-bottom:12px;">
                    <div>
                        <label style="display:block;font-size:11px;color:#8a8a9a;margin-bottom:6px;text-transform:uppercase;font-weight:600;">Type</label>
                        <select onchange="updateParameter(${i}, 'type', this.value)"
                            style="width:100%;padding:10px;background:rgba(20,20,30,0.8);border:1px solid rgba(255,255,255,0.15);border-radius:6px;color:#fff;font-size:13px;">
                            <option value="string" ${param.type === 'string' ? 'selected' : ''}>Text (String)</option>
                            <option value="boolean" ${param.type === 'boolean' ? 'selected' : ''}>Checkbox (Boolean)</option>
                            <option value="choice" ${param.type === 'choice' ? 'selected' : ''}>Dropdown (Choice)</option>
                        </select>
                    </div>
                    <div>
                        <label style="display:block;font-size:11px;color:#8a8a9a;margin-bottom:6px;text-transform:uppercase;font-weight:600;">Default Value</label>
                        <input type="text" value="${param.default_value || ''}" onchange="updateParameter(${i}, 'default_value', this.value)" placeholder="v1.0.0"
                            style="width:100%;padding:10px;background:rgba(20,20,30,0.8);border:1px solid rgba(255,255,255,0.15);border-radius:6px;color:#fff;font-size:13px;">
                    </div>
                </div>

                ${param.type === 'choice' ? `
                <div style="margin-bottom:12px;">
                    <label style="display:block;font-size:11px;color:#8a8a9a;margin-bottom:6px;text-transform:uppercase;font-weight:600;">Choices (comma-separated)</label>
                    <input type="text" value="${(param.choices || []).join(', ')}" onchange="updateParameter(${i}, 'choices', this.value.split(',').map(s=>s.trim()).filter(s=>s))" placeholder="option1, option2, option3"
                        style="width:100%;padding:10px;background:rgba(20,20,30,0.8);border:1px solid rgba(255,255,255,0.15);border-radius:6px;color:#fff;font-size:13px;">
                </div>
                ` : ''}

                <div style="margin-bottom:12px;">
                    <label style="display:block;font-size:11px;color:#8a8a9a;margin-bottom:6px;text-transform:uppercase;font-weight:600;">Description</label>
                    <input type="text" value="${param.description || ''}" onchange="updateParameter(${i}, 'description', this.value)" placeholder="Parameter description"
                        style="width:100%;padding:10px;background:rgba(20,20,30,0.8);border:1px solid rgba(255,255,255,0.15);border-radius:6px;color:#fff;font-size:13px;">
                </div>

                <label style="display:flex;align-items:center;gap:8px;cursor:pointer;">
                    <input type="checkbox" ${param.required ? 'checked' : ''} onchange="updateParameter(${i}, 'required', this.checked)"
                        style="width:16px;height:16px;accent-color:#22d3ee;">
                    <span style="font-size:12px;color:#8a8a9a;">Required parameter</span>
                </label>
            </div>`;
        });
    }

    return `
        <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:16px;">
            <div>
                <span style="font-size:13px;color:#6a6a7a;">${jobParameters.length} parameter${jobParameters.length !== 1 ? 's' : ''} defined</span>
            </div>
            <button type="button" onclick="addParameter()"
                style="padding:10px 20px;font-size:13px;background:rgba(34,211,238,0.2);border:1px solid rgba(34,211,238,0.3);border-radius:6px;color:#22d3ee;cursor:pointer;font-weight:500;">
                + Add Parameter
            </button>
        </div>
        <div style="max-height:400px;overflow-y:auto;">
            ${paramsHtml}
        </div>
    `;
}

export function addParameter() {
    jobParameters.push({
        name: '',
        type: 'string',
        description: '',
        default_value: '',
        choices: [],
        required: false
    });
    renderJobModal();
}

export function updateParameter(index, field, value) {
    if (jobParameters[index]) {
        jobParameters[index][field] = value;
        if (field === 'type') {
            renderJobModal();
        }
    }
}

export function removeParameter(index) {
    jobParameters.splice(index, 1);
    renderJobModal();
}

// ============ Triggers Tab ============

function renderTriggersTab() {
    // Use state variables (webhookEnabled, cronEnabled, cronSchedule are module-level)

    // Common cron presets
    const cronPresets = [
        { label: 'Every hour', value: '0 * * * *' },
        { label: 'Every 4 hours', value: '0 */4 * * *' },
        { label: 'Daily at midnight', value: '0 0 * * *' },
        { label: 'Daily at 2 AM', value: '0 2 * * *' },
        { label: 'Daily at 6 AM', value: '0 6 * * *' },
        { label: 'Weekdays at 9 AM', value: '0 9 * * 1-5' },
        { label: 'Weekly on Sunday', value: '0 0 * * 0' },
        { label: 'Monthly on 1st', value: '0 0 1 * *' },
    ];

    return `
        <div style="background:rgba(30,30,40,0.5);border:1px solid rgba(255,255,255,0.1);border-radius:8px;padding:20px;margin-bottom:16px;">
            <div style="display:flex;align-items:center;gap:12px;margin-bottom:16px;">
                <span style="font-size:24px;">🔗</span>
                <div style="flex:1;">
                    <h4 style="margin:0;color:#e0e0e0;font-size:15px;">Webhook Trigger</h4>
                    <p style="margin:4px 0 0;color:#6a6a7a;font-size:12px;">Trigger builds via HTTP POST request</p>
                </div>
                <label style="display:flex;align-items:center;gap:8px;cursor:pointer;">
                    <input type="checkbox" id="freestyle-job-webhook" ${webhookEnabled ? 'checked' : ''} onchange="toggleWebhook(this.checked)"
                        style="width:20px;height:20px;accent-color:#22d3ee;">
                    <span style="font-size:13px;color:#8a8a9a;">Enable</span>
                </label>
            </div>
            ${webhookEnabled && editingJob?.status?.webhook_url ? `
            <div style="background:rgba(20,20,30,0.8);border-radius:6px;padding:12px;">
                <label style="display:block;font-size:11px;color:#8a8a9a;margin-bottom:6px;text-transform:uppercase;">Webhook URL</label>
                <div style="display:flex;gap:8px;">
                    <input type="text" value="${editingJob.status.webhook_url}" readonly
                        style="flex:1;padding:10px;background:rgba(0,0,0,0.3);border:1px solid rgba(255,255,255,0.1);border-radius:4px;color:#8a8a9a;font-size:12px;font-family:monospace;">
                    <button type="button" onclick="copyWebhookUrl('${editingJob.status.webhook_url}')"
                        style="padding:10px 16px;background:rgba(34,211,238,0.2);border:1px solid rgba(34,211,238,0.3);border-radius:4px;color:#22d3ee;cursor:pointer;font-size:12px;">
                        Copy
                    </button>
                </div>
            </div>
            ` : ''}
        </div>

        <div style="background:rgba(30,30,40,0.5);border:1px solid rgba(255,255,255,0.1);border-radius:8px;padding:20px;">
            <div style="display:flex;align-items:center;gap:12px;margin-bottom:16px;">
                <span style="font-size:24px;">⏰</span>
                <div style="flex:1;">
                    <h4 style="margin:0;color:#e0e0e0;font-size:15px;">Scheduled Trigger (Cron)</h4>
                    <p style="margin:4px 0 0;color:#6a6a7a;font-size:12px;">Run automatically on a schedule</p>
                </div>
                <label style="display:flex;align-items:center;gap:8px;cursor:pointer;">
                    <input type="checkbox" id="freestyle-job-cron-enabled" ${cronEnabled ? 'checked' : ''} onchange="toggleCron(this.checked)"
                        style="width:20px;height:20px;accent-color:#22d3ee;">
                    <span style="font-size:13px;color:#8a8a9a;">Enable</span>
                </label>
            </div>

            <div id="cron-settings" style="display:${cronEnabled ? 'block' : 'none'};">
                <div style="margin-bottom:16px;">
                    <label style="display:block;font-size:11px;color:#8a8a9a;margin-bottom:8px;text-transform:uppercase;">Quick Presets</label>
                    <div style="display:flex;flex-wrap:wrap;gap:8px;">
                        ${cronPresets.map(p => `
                            <button type="button" onclick="setCronPreset('${p.value}')"
                                style="padding:8px 12px;background:${cronSchedule === p.value ? 'rgba(34,211,238,0.3)' : 'rgba(255,255,255,0.05)'};border:1px solid ${cronSchedule === p.value ? 'rgba(34,211,238,0.5)' : 'rgba(255,255,255,0.1)'};border-radius:4px;color:${cronSchedule === p.value ? '#22d3ee' : '#8a8a9a'};cursor:pointer;font-size:12px;transition:all 0.2s;">
                                ${p.label}
                            </button>
                        `).join('')}
                    </div>
                </div>

                <div>
                    <label style="display:block;font-size:11px;color:#8a8a9a;margin-bottom:8px;text-transform:uppercase;">Custom Cron Expression</label>
                    <div style="display:flex;gap:12px;align-items:center;">
                        <input type="text" id="freestyle-job-cron" value="${cronSchedule}" placeholder="0 2 * * *" onchange="updateCronPreview()"
                            style="flex:1;padding:12px;background:rgba(20,20,30,0.8);border:1px solid rgba(255,255,255,0.15);border-radius:6px;color:#fff;font-size:14px;font-family:monospace;">
                    </div>
                    <div id="cron-preview" style="margin-top:10px;padding:10px;background:rgba(74,222,128,0.1);border-radius:4px;color:#4ade80;font-size:12px;display:${cronSchedule ? 'block' : 'none'};">
                        ${getCronDescription(cronSchedule)}
                    </div>
                    <div style="margin-top:10px;font-size:11px;color:#6a6a7a;">
                        Format: minute hour day month weekday<br>
                        Example: <code style="background:rgba(255,255,255,0.1);padding:2px 6px;border-radius:3px;">0 2 * * *</code> = Every day at 2:00 AM
                    </div>
                </div>
            </div>
        </div>

        <div style="margin-top:16px;padding:16px;background:rgba(34,211,238,0.1);border:1px solid rgba(34,211,238,0.2);border-radius:8px;">
            <div style="display:flex;align-items:center;gap:10px;">
                <span style="font-size:18px;">💡</span>
                <div style="font-size:12px;color:#8a8a9a;">
                    <strong style="color:#22d3ee;">Manual trigger</strong> is always available. Use the ▶ button in the jobs list to run manually.
                </div>
            </div>
        </div>
    `;
}

function getCronDescription(cron) {
    if (!cron) return '';
    const parts = cron.split(' ');
    if (parts.length < 5) return 'Invalid cron expression';

    const [min, hour, day, month, weekday] = parts;

    // Simple descriptions for common patterns
    if (cron === '0 * * * *') return '🕐 Runs every hour at minute 0';
    if (cron === '0 */4 * * *') return '🕐 Runs every 4 hours';
    if (cron === '0 0 * * *') return '🕛 Runs daily at midnight';
    if (cron === '0 2 * * *') return '🕑 Runs daily at 2:00 AM';
    if (cron === '0 6 * * *') return '🕕 Runs daily at 6:00 AM';
    if (cron === '0 9 * * 1-5') return '🕘 Runs weekdays (Mon-Fri) at 9:00 AM';
    if (cron === '0 0 * * 0') return '📅 Runs weekly on Sunday at midnight';
    if (cron === '0 0 1 * *') return '📅 Runs monthly on the 1st at midnight';

    return `⏰ Runs at minute ${min}, hour ${hour}, day ${day}, month ${month}, weekday ${weekday}`;
}

export function toggleWebhook(enabled) {
    webhookEnabled = enabled;
}

export function toggleCron(enabled) {
    cronEnabled = enabled;
    document.getElementById('cron-settings').style.display = enabled ? 'block' : 'none';
}

export function setCronPreset(value) {
    cronSchedule = value;
    cronEnabled = true;
    document.getElementById('freestyle-job-cron').value = value;
    document.getElementById('freestyle-job-cron-enabled').checked = true;
    document.getElementById('cron-settings').style.display = 'block';
    updateCronPreview();
    renderJobModal();
}

export function updateCronPreview() {
    const cronInput = document.getElementById('freestyle-job-cron');
    const preview = document.getElementById('cron-preview');
    if (cronInput && preview) {
        cronSchedule = cronInput.value; // Update state
        const desc = getCronDescription(cronInput.value);
        preview.innerHTML = desc;
        preview.style.display = cronInput.value ? 'block' : 'none';
    }
}

export function copyWebhookUrl(url) {
    window._copyText(url).then(() => {
        alert('Webhook URL copied to clipboard!');
    });
}

// ============ Save & Edit ============

export async function saveFreestyleJob(e) {
    if (e && e.preventDefault) {
        e.preventDefault();
    }

    // Save current tab data before collecting all data
    saveCurrentTabData();

    console.log('saveFreestyleJob called', { jobName, jobDesc, jobEnabled, buildSteps, jobParameters, envVariables });

    const data = {
        name: jobName,
        description: jobDesc,
        enabled: jobEnabled,
        build_steps: buildSteps,
        parameters: jobParameters,
        triggers: [],
        environment: {}
    };

    // Add SCM configuration if Git is enabled
    if (scmType === 'git') {
        data.scm = {
            type: 'git',
            repositories: scmRepositories.filter(r => r.url && r.url.trim()),
            branches: scmBranches.filter(b => b.specifier && b.specifier.trim()),
            clone_depth: scmCloneDepth,
            submodules: scmSubmodules,
            clean_before: scmCleanBefore
        };
    }

    // Convert environment variables array to object
    envVariables.forEach(env => {
        if (env.key && env.key.trim()) {
            data.environment[env.key.trim()] = env.value || '';
        }
    });

    // Triggers - use state variables
    if (webhookEnabled) {
        data.triggers.push({ type: 'webhook', enabled: true });
    }

    if (cronEnabled && cronSchedule) {
        data.triggers.push({ type: 'cron', schedule: cronSchedule, enabled: true });
    }

    // Validation
    if (!data.name) {
        alert('Job name is required');
        switchJobTab('general');
        return;
    }

    if (buildSteps.length === 0) {
        alert('At least one build step is required');
        switchJobTab('steps');
        return;
    }

    // Check if SCP steps have remote hosts (SCP requires SSH)
    const scpStepsWithoutHost = buildSteps.filter(s =>
        (s.type === 'scp_push' || s.type === 'scp_pull') &&
        (!s.host_id || s.host_id === 'local')
    );
    if (scpStepsWithoutHost.length > 0) {
        alert(`SCP step "${scpStepsWithoutHost[0].name || 'unnamed'}" requires a remote SSH host.`);
        switchJobTab('steps');
        return;
    }

    try {
        const url = editingJob ? `${API_BASE}/cicd/freestyle/jobs/${editingJob.id}` : `${API_BASE}/cicd/freestyle/jobs`;
        const method = editingJob ? 'PUT' : 'POST';
        const r = await fetch(url, {
            method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        const d = await r.json();
        if (d.error) {
            alert('Error: ' + d.error);
        } else {
            closeFreestyleJobModal();
            loadFreestyleJobs();
        }
    } catch (e) {
        alert('Failed to save: ' + e.message);
    }
}

export async function editFreestyleJob(id) {
    const job = freestyleJobs.find(j => j.id === id);
    if (!job) return;

    editingJob = job;
    buildSteps = job.build_steps ? [...job.build_steps] : [];
    jobParameters = job.parameters ? [...job.parameters] : [];
    currentTab = 'general';

    // Initialize state variables from job
    jobName = job.name || '';
    jobDesc = job.description || '';
    jobEnabled = job.enabled !== false;

    // Initialize trigger state from job
    const triggers = job.triggers || [];
    const webhookTrigger = triggers.find(t => t.type === 'webhook');
    const cronTrigger = triggers.find(t => t.type === 'cron');
    webhookEnabled = webhookTrigger?.enabled || false;
    cronEnabled = cronTrigger?.enabled || false;
    cronSchedule = cronTrigger?.schedule || '';

    // Initialize SCM state from job
    if (job.scm && job.scm.type === 'git') {
        scmType = 'git';
        scmRepositories = job.scm.repositories ? [...job.scm.repositories] : [];
        scmBranches = job.scm.branches ? [...job.scm.branches] : [];
        scmCloneDepth = job.scm.clone_depth || 0;
        scmSubmodules = job.scm.submodules || false;
        scmCleanBefore = job.scm.clean_before || false;
    } else {
        scmType = 'none';
        scmRepositories = [];
        scmBranches = [];
        scmCloneDepth = 0;
        scmSubmodules = false;
        scmCleanBefore = false;
    }

    // Convert environment object to array
    envVariables = Object.entries(job.environment || {}).map(([key, value]) => ({ key, value }));

    await loadSSHHostsForSelect();
    await loadGitCredentialsForSelect();
    renderJobModal();
    document.getElementById('freestyle-job-modal').style.display = 'flex';
}

export async function deleteFreestyleJob(id) {
    if (!confirm('Delete this job and all its builds? This cannot be undone.')) return;
    try {
        const r = await fetch(`${API_BASE}/cicd/freestyle/jobs/${id}`, { method: 'DELETE' });
        const d = await r.json();
        if (d.error) {
            alert('Error: ' + d.error);
        } else {
            loadFreestyleJobs();
        }
    } catch (e) {
        alert('Failed to delete: ' + e.message);
    }
}

export async function cloneFreestyleJob(id) {
    try {
        const r = await fetch(`${API_BASE}/cicd/freestyle/jobs/${id}`);
        const job = await r.json();
        if (job.error) {
            alert('Error: ' + job.error);
            return;
        }

        const newName = prompt('Enter name for cloned job:', job.name + '-copy');
        if (!newName) return;

        // Create clone data without id and status
        const cloneData = {
            name: newName,
            description: job.description || '',
            enabled: job.enabled,
            ssh_host_ids: job.ssh_host_ids || [],
            build_steps: job.build_steps || [],
            parameters: job.parameters || [],
            env_vars: job.env_vars || [],
            triggers: [],  // Don't clone triggers to avoid duplicate webhooks
            scm: job.scm || null,
            notifications: job.notifications || []
        };

        const createR = await fetch(`${API_BASE}/cicd/freestyle/jobs`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(cloneData)
        });
        const createD = await createR.json();
        if (createD.error) {
            alert('Error cloning job: ' + createD.error);
        } else {
            alert(`Job "${newName}" created successfully! Triggers were not cloned.`);
            loadFreestyleJobs();
        }
    } catch (e) {
        alert('Failed to clone job: ' + e.message);
    }
}

export async function runFreestyleJob(id) {
    const job = freestyleJobs.find(j => j.id === id);

    // If job has parameters, show a dialog to collect them
    if (job && job.parameters && job.parameters.length > 0) {
        showRunJobModal(job);
        return;
    }

    if (!confirm('Run this job?')) return;
    await triggerJobBuild(id, {});
}

function showRunJobModal(job) {
    let paramsHtml = job.parameters.map((p, i) => {
        if (p.type === 'boolean') {
            return `<div style="margin-bottom:16px;">
                <label style="display:flex;align-items:center;gap:10px;cursor:pointer;">
                    <input type="checkbox" id="run-param-${i}" ${p.default_value === 'true' ? 'checked' : ''} style="width:18px;height:18px;accent-color:#22d3ee;">
                    <span style="font-weight:500;color:#e0e0e0;">${p.name}${p.required ? ' *' : ''}</span>
                </label>
                ${p.description ? `<small style="color:#6a6a7a;margin-left:28px;display:block;margin-top:4px;">${p.description}</small>` : ''}
            </div>`;
        } else if (p.type === 'choice' && p.choices && p.choices.length > 0) {
            return `<div style="margin-bottom:16px;">
                <label style="display:block;margin-bottom:6px;font-weight:500;color:#e0e0e0;">${p.name}${p.required ? ' *' : ''}</label>
                <select id="run-param-${i}" style="width:100%;padding:10px;background:rgba(20,20,30,0.8);border:1px solid rgba(255,255,255,0.15);border-radius:6px;color:#fff;font-size:14px;">
                    ${p.choices.map(c => `<option value="${c}" ${c === p.default_value ? 'selected' : ''}>${c}</option>`).join('')}
                </select>
                ${p.description ? `<small style="color:#6a6a7a;margin-top:4px;display:block;">${p.description}</small>` : ''}
            </div>`;
        } else {
            return `<div style="margin-bottom:16px;">
                <label style="display:block;margin-bottom:6px;font-weight:500;color:#e0e0e0;">${p.name}${p.required ? ' *' : ''}</label>
                <input type="text" id="run-param-${i}" value="${p.default_value || ''}" ${p.required ? 'required' : ''}
                    style="width:100%;padding:10px;background:rgba(20,20,30,0.8);border:1px solid rgba(255,255,255,0.15);border-radius:6px;color:#fff;font-size:14px;">
                ${p.description ? `<small style="color:#6a6a7a;margin-top:4px;display:block;">${p.description}</small>` : ''}
            </div>`;
        }
    }).join('');

    // Create modal dynamically
    const existingModal = document.getElementById('run-job-modal');
    if (existingModal) existingModal.remove();

    const modal = document.createElement('div');
    modal.id = 'run-job-modal';
    modal.className = 'k8s-modal-overlay';
    modal.style.display = 'flex';
    modal.innerHTML = `
        <div class="k8s-modal" style="max-width:500px;">
            <div class="k8s-modal-header">
                <h3>
                    <svg fill="none" stroke="#4ade80" viewBox="0 0 24 24" style="width:20px;height:20px;">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"/>
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
                    </svg>
                    <span>Run: ${job.name}</span>
                </h3>
                <button class="k8s-modal-close" onclick="document.getElementById('run-job-modal').remove()">&times;</button>
            </div>
            <div class="k8s-modal-body">
                <p style="color:#8a8a9a;margin-bottom:20px;">Provide values for the build parameters:</p>
                ${paramsHtml}
            </div>
            <div class="k8s-modal-footer">
                <button type="button" class="modal-btn cancel" onclick="document.getElementById('run-job-modal').remove()">Cancel</button>
                <button type="button" class="modal-btn apply" onclick="submitJobRun('${job.id}', ${JSON.stringify(job.parameters).replace(/"/g, '&quot;')})">Run Build</button>
            </div>
        </div>
    `;
    document.body.appendChild(modal);
}

window.submitJobRun = async function(jobId, parameters) {
    const params = {};
    parameters.forEach((p, i) => {
        const el = document.getElementById(`run-param-${i}`);
        if (el) {
            if (p.type === 'boolean') {
                params[p.name] = el.checked ? 'true' : 'false';
            } else {
                params[p.name] = el.value;
            }
        }
    });

    document.getElementById('run-job-modal').remove();
    await triggerJobBuild(jobId, params);
};

async function triggerJobBuild(jobId, parameters) {
    try {
        const r = await fetch(`${API_BASE}/cicd/freestyle/jobs/${jobId}/build`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ parameters })
        });
        const d = await r.json();
        if (d.error) {
            alert('Error: ' + d.error);
        } else {
            // Open build console
            openBuildConsole(d.id);
        }
    } catch (e) {
        alert('Failed to start build: ' + e.message);
    }
}

// ============ Freestyle Builds ============

export async function loadFreestyleBuilds() {
    try {
        const r = await fetch(`${API_BASE}/cicd/freestyle/builds`);
        const d = await r.json();
        freestyleBuilds = d.builds || [];
        renderFreestyleBuildsTable();
    } catch (e) {
        console.error('Failed to load freestyle builds:', e);
    }
}

function renderFreestyleBuildsTable() {
    const tbody = document.getElementById('freestyle-builds-tbody');
    tbody.innerHTML = '';
    if (freestyleBuilds.length === 0) {
        tbody.innerHTML = '<tr><td colspan="8" style="text-align:center;color:#6a6a7a;padding:30px;">No builds yet</td></tr>';
        return;
    }
    freestyleBuilds.forEach(b => {
        const statusClass = getRunStatusClass(b.status);
        const duration = b.duration_ms ? formatDuration(b.duration_ms) : '-';
        const started = b.started_at ? formatTime(b.started_at) : '-';
        const stepsInfo = b.steps ? `${b.steps.filter(s => s.status === 'succeeded').length}/${b.steps.length}` : '-';
        const canCancel = b.status === 'running' || b.status === 'pending';
        const canRetry = b.status === 'failed' || b.status === 'cancelled';
        tbody.innerHTML += `<tr>
            <td>#${b.build_number}</td>
            <td>${b.job_name}</td>
            <td class="${statusClass}">${b.status}</td>
            <td>${b.trigger_type}</td>
            <td>${stepsInfo}</td>
            <td>${duration}</td>
            <td>${started}</td>
            <td class="action-cell">
                <button class="row-action-btn logs" onclick="openBuildConsole('${b.id}')" title="Console">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>
                </button>
                ${canRetry ? `<button class="row-action-btn" style="background:rgba(251,191,36,0.2);color:#fbbf24;" onclick="retryFreestyleBuild('${b.job_id}')" title="Retry">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>
                </button>` : ''}
                ${canCancel ? `<button class="row-action-btn delete" onclick="cancelFreestyleBuild('${b.id}')" title="Cancel">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"/></svg>
                </button>` : ''}
            </td>
        </tr>`;
    });
}

export async function viewFreestyleJobBuilds(jobId) {
    try {
        const r = await fetch(`${API_BASE}/cicd/freestyle/jobs/${jobId}/builds`);
        const d = await r.json();
        freestyleBuilds = d.builds || [];
        renderFreestyleBuildsTable();
        // Switch to builds tab
        document.querySelectorAll('#window-cicd .tab-content').forEach(t => t.classList.remove('active'));
        document.querySelectorAll('#window-cicd .tab-btn').forEach(b => b.classList.remove('active'));
        document.getElementById('cicd-tab-freestyle-builds').classList.add('active');
    } catch (e) {
        alert('Failed to load builds: ' + e.message);
    }
}

export async function cancelFreestyleBuild(id) {
    if (!confirm('Cancel this build?')) return;
    try {
        const r = await fetch(`${API_BASE}/cicd/freestyle/builds/${id}/cancel`, { method: 'POST' });
        const d = await r.json();
        if (d.error) {
            alert('Error: ' + d.error);
        } else {
            loadFreestyleBuilds();
        }
    } catch (e) {
        alert('Failed to cancel: ' + e.message);
    }
}

export async function retryFreestyleBuild(jobId) {
    if (!confirm('Retry this job?')) return;
    try {
        const r = await fetch(`${API_BASE}/cicd/freestyle/jobs/${jobId}/trigger`, { method: 'POST' });
        const d = await r.json();
        if (d.error) {
            alert('Error: ' + d.error);
        } else {
            alert(`Job re-triggered! Build #${d.build_number}`);
            loadFreestyleBuilds();
        }
    } catch (e) {
        alert('Failed to retry: ' + e.message);
    }
}

// ============ Build Console ============

export async function openBuildConsole(buildId) {
    document.getElementById('freestyle-build-modal').style.display = 'flex';
    document.getElementById('freestyle-build-content').textContent = 'Loading...';
    document.getElementById('freestyle-build-status').textContent = 'Loading';

    try {
        const r = await fetch(`${API_BASE}/cicd/freestyle/builds/${buildId}`);
        const build = await r.json();

        document.getElementById('freestyle-build-title').textContent = `Build #${build.build_number} - ${build.job_name}`;
        document.getElementById('freestyle-build-status').textContent = build.status;
        document.getElementById('freestyle-build-status').className = getRunStatusClass(build.status);

        // Render steps
        renderBuildSteps(build.steps || []);

        // Fetch logs
        const logsR = await fetch(`${API_BASE}/cicd/freestyle/builds/${buildId}/logs`);
        const logsD = await logsR.json();
        document.getElementById('freestyle-build-content').textContent = logsD.logs || 'No logs available';

        // If running, connect WebSocket for live updates
        if (build.status === 'running' || build.status === 'pending') {
            connectBuildLogStream(buildId);
        }
    } catch (e) {
        document.getElementById('freestyle-build-content').textContent = 'Error loading build: ' + e.message;
    }
}

function renderBuildSteps(steps) {
    const container = document.getElementById('freestyle-build-steps');
    container.innerHTML = '';
    steps.forEach(s => {
        const statusIcon = s.status === 'succeeded' ? '&#10004;' :
                          s.status === 'failed' ? '&#10008;' :
                          s.status === 'running' ? '&#9679;' : '&#9675;';
        const statusColor = s.status === 'succeeded' ? '#4ade80' :
                           s.status === 'failed' ? '#ef4444' :
                           s.status === 'running' ? '#22d3ee' : '#6a6a7a';
        container.innerHTML += `<div style="display:flex;align-items:center;gap:8px;padding:6px 10px;background:rgba(30,30,40,0.3);border-radius:4px;margin-bottom:4px;">
            <span style="color:${statusColor};">${statusIcon}</span>
            <span style="flex:1;font-size:12px;">${s.name}</span>
            <span style="font-size:11px;color:#6a6a7a;">${s.host_name || ''}</span>
        </div>`;
    });
}

function connectBuildLogStream(buildId) {
    if (buildLogWs) {
        buildLogWs.close();
    }

    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${wsProtocol}//${window.location.host}/api/v1/cicd/freestyle/builds/${buildId}/logs/stream`;

    buildLogWs = new WebSocket(wsUrl);

    buildLogWs.onmessage = (e) => {
        const content = document.getElementById('freestyle-build-content');
        content.textContent += e.data;
        content.scrollTop = content.scrollHeight;
    };

    buildLogWs.onerror = (e) => {
        console.error('WebSocket error:', e);
    };

    buildLogWs.onclose = () => {
        buildLogWs = null;
        // Refresh build status
        loadFreestyleBuilds();
    };
}

export function closeBuildConsole() {
    document.getElementById('freestyle-build-modal').style.display = 'none';
    if (buildLogWs) {
        buildLogWs.close();
        buildLogWs = null;
    }
}

// Make functions available globally
window.testSSHHost = testSSHHost;
window.editSSHHost = editSSHHost;
window.deleteSSHHost = deleteSSHHost;
window.cloneSSHHost = cloneSSHHost;
window.runFreestyleJob = runFreestyleJob;
window.editFreestyleJob = editFreestyleJob;
window.deleteFreestyleJob = deleteFreestyleJob;
window.cloneFreestyleJob = cloneFreestyleJob;
window.viewFreestyleJobBuilds = viewFreestyleJobBuilds;
window.openBuildConsole = openBuildConsole;
window.cancelFreestyleBuild = cancelFreestyleBuild;
window.retryFreestyleBuild = retryFreestyleBuild;
window.addBuildStep = addBuildStep;
window.updateStep = updateStep;
window.removeStep = removeStep;
window.moveStepUp = moveStepUp;
window.moveStepDown = moveStepDown;
window.switchJobTab = switchJobTab;
window.addEnvVariable = addEnvVariable;
window.updateEnvVariable = updateEnvVariable;
window.removeEnvVariable = removeEnvVariable;
window.addParameter = addParameter;
window.updateParameter = updateParameter;
window.removeParameter = removeParameter;
window.toggleWebhook = toggleWebhook;
window.toggleCron = toggleCron;
window.setCronPreset = setCronPreset;
window.updateCronPreview = updateCronPreview;
window.copyWebhookUrl = copyWebhookUrl;
window.saveFreestyleJob = saveFreestyleJob;
window.closeFreestyleJobModal = closeFreestyleJobModal;

// Git credential window exports
window.testGitCredential = testGitCredential;
window.editGitCredential = editGitCredential;
window.deleteGitCredential = deleteGitCredential;

// SCM tab window exports
window.setSCMType = setSCMType;
window.addSCMRepo = addSCMRepo;
window.updateSCMRepo = updateSCMRepo;
window.removeSCMRepo = removeSCMRepo;
window.addSCMBranch = addSCMBranch;
window.updateSCMBranch = updateSCMBranch;
window.removeSCMBranch = removeSCMBranch;
