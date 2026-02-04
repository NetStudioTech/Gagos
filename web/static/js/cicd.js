// CI/CD Module for GAGOS

import { API_BASE } from './app.js';
import { saveState } from './state.js';
import { escapeHtml, formatDuration, formatSize, formatTime } from './utils.js';
import {
    loadSSHHosts, showAddSSHHostModal, closeSSHHostModal, updateAuthFields, saveSSHHost,
    loadGitCredentials, showAddGitCredentialModal, closeGitCredentialModal, updateGitAuthFields, saveGitCredential,
    loadFreestyleJobs, showCreateFreestyleJobModal, closeFreestyleJobModal, saveFreestyleJob, addBuildStep,
    loadFreestyleBuilds, closeBuildConsole
} from './freestyle.js';
import {
    loadNotifications, showCreateNotificationModal, closeNotificationModal, saveNotification
} from './notifications.js';

let cicdLogWs = null;
let cicdPipelines = [];
let cicdRuns = [];

export function showCicdTab(tabId) {
    document.querySelectorAll('#window-cicd .tab-content').forEach(t => t.classList.remove('active'));
    document.querySelectorAll('#window-cicd .tab-btn').forEach(b => b.classList.remove('active'));
    document.getElementById('cicd-tab-' + tabId).classList.add('active');
    event.target.classList.add('active');

    if (tabId === 'overview') loadCicdStats();
    else if (tabId === 'pipelines') loadCicdPipelines();
    else if (tabId === 'runs') loadCicdRuns();
    else if (tabId === 'artifacts') loadCicdArtifacts();
    else if (tabId === 'ssh-hosts') loadSSHHosts();
    else if (tabId === 'git-credentials') loadGitCredentials();
    else if (tabId === 'freestyle-jobs') loadFreestyleJobs();
    else if (tabId === 'freestyle-builds') loadFreestyleBuilds();
    else if (tabId === 'notifications') loadNotifications();

    saveState();
}

// Export freestyle functions for global access
export {
    showAddSSHHostModal, closeSSHHostModal, updateAuthFields, saveSSHHost,
    showCreateFreestyleJobModal, closeFreestyleJobModal, saveFreestyleJob, addBuildStep,
    closeBuildConsole,
    showAddGitCredentialModal, closeGitCredentialModal, updateGitAuthFields, saveGitCredential, loadGitCredentials
};

// Export notification functions for global access
export {
    showCreateNotificationModal, closeNotificationModal, saveNotification
};

export async function loadCicdData() {
    await loadCicdStats();
}

export async function loadCicdStats() {
    try {
        const r = await fetch(`${API_BASE}/cicd/stats`);
        const d = await r.json();
        document.getElementById('stat-pipelines').textContent = d.total_pipelines || 0;
        document.getElementById('stat-running').textContent = d.running_runs || 0;
        document.getElementById('stat-succeeded').textContent = d.succeeded_24h || 0;
        document.getElementById('stat-failed').textContent = d.failed_24h || 0;

        // Load recent runs
        const r2 = await fetch(`${API_BASE}/cicd/runs?limit=10`);
        const d2 = await r2.json();
        renderRecentRuns(d2.runs || []);
    } catch (e) {
        console.error('Failed to load CI/CD stats:', e);
    }
}

function renderRecentRuns(runs) {
    const tbody = document.getElementById('cicd-recent-runs');
    tbody.innerHTML = '';
    if (runs.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" style="text-align:center;color:#6a6a7a;padding:30px;">No runs yet</td></tr>';
        return;
    }
    runs.forEach(run => {
        const statusClass = getRunStatusClass(run.status);
        const duration = run.duration_ms ? formatDuration(run.duration_ms) : '-';
        const started = run.started_at ? formatTime(run.started_at) : '-';
        tbody.innerHTML += `<tr>
            <td>#${run.run_number}</td>
            <td>${run.pipeline_name}</td>
            <td class="${statusClass}">${run.status}</td>
            <td>${run.trigger_type}</td>
            <td>${duration}</td>
            <td>${started}</td>
        </tr>`;
    });
}

export async function loadCicdPipelines() {
    try {
        const r = await fetch(`${API_BASE}/cicd/pipelines`);
        const d = await r.json();
        cicdPipelines = d.pipelines || [];
        renderPipelinesTable();
    } catch (e) {
        console.error('Failed to load pipelines:', e);
    }
}

function renderPipelinesTable() {
    const tbody = document.getElementById('cicd-pipelines-tbody');
    tbody.innerHTML = '';
    if (cicdPipelines.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" style="text-align:center;color:#6a6a7a;padding:30px;">No pipelines. Click "Create" to add one.</td></tr>';
        return;
    }
    cicdPipelines.forEach(p => {
        const lastRun = p.status.last_run_at ? formatTime(p.status.last_run_at) : 'Never';
        const triggers = getTriggerIcons(p.spec.triggers || []);
        tbody.innerHTML += `<tr>
            <td><strong>${p.name}</strong></td>
            <td style="color:#8a8a9a;">${p.description || '-'}</td>
            <td>${lastRun}</td>
            <td>${p.status.total_runs || 0}</td>
            <td>${triggers}</td>
            <td class="action-cell">
                <button class="row-action-btn" style="background:rgba(74,222,128,0.2);color:#4ade80;" onclick="triggerPipeline('${p.id}')" title="Run">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"/><path stroke-linecap="round" stroke-linejoin="round" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
                </button>
                <button class="row-action-btn describe" onclick="viewPipeline('${p.id}')" title="View">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/><path stroke-linecap="round" stroke-linejoin="round" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/></svg>
                </button>
                <button class="row-action-btn" style="background:rgba(167,139,250,0.2);color:#a78bfa;" onclick="clonePipeline('${p.id}')" title="Clone">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
                </button>
                <button class="row-action-btn" style="background:rgba(251,191,36,0.2);color:#fbbf24;" onclick="copyPipelineBadge('${p.id}', '${p.name}')" title="Copy Badge">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z"/></svg>
                </button>
                <button class="row-action-btn delete" onclick="deletePipeline('${p.id}')" title="Delete">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
                </button>
            </td>
        </tr>`;
    });
}

function getTriggerIcons(triggers) {
    if (!triggers || triggers.length === 0) return '<span style="color:#6a6a7a;">Manual only</span>';
    let icons = '';
    triggers.forEach(t => {
        if (t.type === 'webhook' && t.enabled !== false) {
            icons += '<span title="Webhook" style="color:#22d3ee;margin-right:8px;">&#128279;</span>';
        } else if (t.type === 'cron' && t.enabled) {
            icons += '<span title="Cron: ' + t.schedule + '" style="color:#a78bfa;margin-right:8px;">&#9200;</span>';
        }
    });
    return icons || '<span style="color:#6a6a7a;">Manual only</span>';
}

export async function loadCicdRuns() {
    try {
        const r = await fetch(`${API_BASE}/cicd/runs?limit=50`);
        const d = await r.json();
        cicdRuns = d.runs || [];
        renderRunsTable();
    } catch (e) {
        console.error('Failed to load runs:', e);
    }
}

function renderRunsTable() {
    const tbody = document.getElementById('cicd-runs-tbody');
    tbody.innerHTML = '';
    if (cicdRuns.length === 0) {
        tbody.innerHTML = '<tr><td colspan="8" style="text-align:center;color:#6a6a7a;padding:30px;">No runs yet</td></tr>';
        return;
    }
    cicdRuns.forEach(run => {
        const statusClass = getRunStatusClass(run.status);
        const duration = run.duration_ms ? formatDuration(run.duration_ms) : '-';
        const started = run.started_at ? formatTime(run.started_at) : '-';
        const jobsInfo = run.jobs ? `${run.jobs.filter(j => j.status === 'succeeded').length}/${run.jobs.length}` : '-';
        const canCancel = run.status === 'running' || run.status === 'pending';
        const canRetry = run.status === 'failed' || run.status === 'cancelled';
        tbody.innerHTML += `<tr>
            <td>#${run.run_number}</td>
            <td>${run.pipeline_name}</td>
            <td class="${statusClass}">${run.status}</td>
            <td>${run.trigger_type}</td>
            <td>${jobsInfo}</td>
            <td>${duration}</td>
            <td>${started}</td>
            <td class="action-cell">
                <button class="row-action-btn logs" onclick="viewRunJobs('${run.id}')" title="View Jobs">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M4 6h16M4 12h16M4 18h7"/></svg>
                </button>
                ${canRetry ? `<button class="row-action-btn" style="background:rgba(251,191,36,0.2);color:#fbbf24;" onclick="retryPipelineRun('${run.pipeline_id}')" title="Retry">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>
                </button>` : ''}
                ${canCancel ? `<button class="row-action-btn delete" onclick="cancelRun('${run.id}')" title="Cancel">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"/></svg>
                </button>` : ''}
            </td>
        </tr>`;
    });
}

export function getRunStatusClass(status) {
    switch (status) {
        case 'succeeded': return 'status-running';
        case 'failed': return 'status-failed';
        case 'running': case 'pending': return 'status-pending';
        case 'cancelled': return 'status-failed';
        case 'skipped': return 'status-pending';
        default: return '';
    }
}

export async function loadCicdArtifacts() {
    try {
        const r = await fetch(`${API_BASE}/cicd/artifacts`);
        const d = await r.json();
        renderArtifactsTable(d.artifacts || []);
    } catch (e) {
        console.error('Failed to load artifacts:', e);
    }
}

function renderArtifactsTable(artifacts) {
    const tbody = document.getElementById('cicd-artifacts-tbody');
    tbody.innerHTML = '';
    if (artifacts.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" style="text-align:center;color:#6a6a7a;padding:30px;">No artifacts</td></tr>';
        return;
    }
    artifacts.forEach(a => {
        const size = formatSize(a.size);
        const created = formatTime(a.created_at);
        tbody.innerHTML += `<tr>
            <td>${a.name}</td>
            <td>${a.pipeline_id.slice(0,12)}...</td>
            <td>${a.run_id.slice(0,12)}...</td>
            <td>${size}</td>
            <td>${created}</td>
            <td class="action-cell">
                <a href="${API_BASE}/cicd/artifacts/${a.id}/download" class="row-action-btn logs" style="text-decoration:none;" title="Download">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/></svg>
                </a>
                <button class="row-action-btn delete" onclick="deleteArtifact('${a.id}')" title="Delete">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
                </button>
            </td>
        </tr>`;
    });
}

export async function loadSamplePipeline() {
    try {
        const r = await fetch(`${API_BASE}/cicd/sample`);
        const d = await r.json();
        document.getElementById('pipeline-yaml-editor').value = d.yaml;
    } catch (e) {
        alert('Failed to load sample: ' + e.message);
    }
}

export async function validatePipeline() {
    const yaml = document.getElementById('pipeline-yaml-editor').value;
    const result = document.getElementById('pipeline-validation-result');
    if (!yaml.trim()) {
        result.style.display = 'block';
        result.style.background = 'rgba(239,68,68,0.1)';
        result.style.border = '1px solid rgba(239,68,68,0.3)';
        result.style.color = '#ef4444';
        result.textContent = 'Please enter pipeline YAML';
        return;
    }
    // Basic YAML validation by trying to create
    try {
        const r = await fetch(`${API_BASE}/cicd/pipelines`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ yaml })
        });
        const d = await r.json();
        if (d.error) {
            result.style.display = 'block';
            result.style.background = 'rgba(239,68,68,0.1)';
            result.style.border = '1px solid rgba(239,68,68,0.3)';
            result.style.color = '#ef4444';
            result.textContent = 'Validation error: ' + d.error;
        } else {
            // Delete the created pipeline (it was just for validation)
            await fetch(`${API_BASE}/cicd/pipelines/${d.id}`, { method: 'DELETE' });
            result.style.display = 'block';
            result.style.background = 'rgba(74,222,128,0.1)';
            result.style.border = '1px solid rgba(74,222,128,0.3)';
            result.style.color = '#4ade80';
            result.textContent = 'Valid pipeline YAML! Name: ' + d.name;
        }
    } catch (e) {
        result.style.display = 'block';
        result.style.background = 'rgba(239,68,68,0.1)';
        result.style.border = '1px solid rgba(239,68,68,0.3)';
        result.style.color = '#ef4444';
        result.textContent = 'Error: ' + e.message;
    }
}

export async function createPipeline() {
    const yaml = document.getElementById('pipeline-yaml-editor').value;
    if (!yaml.trim()) {
        alert('Please enter pipeline YAML');
        return;
    }
    try {
        const r = await fetch(`${API_BASE}/cicd/pipelines`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ yaml })
        });
        const d = await r.json();
        if (d.error) {
            alert('Error: ' + d.error);
        } else {
            document.getElementById('pipeline-yaml-editor').value = '';
            document.getElementById('pipeline-validation-result').style.display = 'none';
            showCicdTab('pipelines');
            loadCicdPipelines();
        }
    } catch (e) {
        alert('Failed to create pipeline: ' + e.message);
    }
}

export async function triggerPipeline(id) {
    if (!confirm('Run this pipeline?')) return;
    try {
        const r = await fetch(`${API_BASE}/cicd/pipelines/${id}/trigger`, { method: 'POST' });
        const d = await r.json();
        if (d.error) {
            alert('Error: ' + d.error);
        } else {
            alert(`Pipeline triggered! Run #${d.run_number}`);
            loadCicdStats();
            loadCicdPipelines();
        }
    } catch (e) {
        alert('Failed to trigger: ' + e.message);
    }
}

export async function viewPipeline(id) {
    try {
        const r = await fetch(`${API_BASE}/cicd/pipelines/${id}`);
        const p = await r.json();
        if (p.error) {
            alert('Error: ' + p.error);
            return;
        }
        // Show in create tab for editing
        document.getElementById('pipeline-yaml-editor').value = p.yaml;
        showCicdTab('create');
    } catch (e) {
        alert('Failed to load pipeline: ' + e.message);
    }
}

export async function deletePipeline(id) {
    if (!confirm('Delete this pipeline? This cannot be undone.')) return;
    try {
        const r = await fetch(`${API_BASE}/cicd/pipelines/${id}`, { method: 'DELETE' });
        const d = await r.json();
        if (d.error) {
            alert('Error: ' + d.error);
        } else {
            loadCicdPipelines();
            loadCicdStats();
        }
    } catch (e) {
        alert('Failed to delete: ' + e.message);
    }
}

export async function clonePipeline(id) {
    try {
        const r = await fetch(`${API_BASE}/cicd/pipelines/${id}`);
        const p = await r.json();
        if (p.error) {
            alert('Error: ' + p.error);
            return;
        }

        const newName = prompt('Enter name for cloned pipeline:', p.name + '-copy');
        if (!newName) return;

        // Replace the name in the YAML
        let yaml = p.yaml;
        yaml = yaml.replace(/^name:\s*.*$/m, `name: ${newName}`);

        // Create the cloned pipeline
        const createR = await fetch(`${API_BASE}/cicd/pipelines`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ yaml })
        });
        const createD = await createR.json();
        if (createD.error) {
            alert('Error cloning pipeline: ' + createD.error);
        } else {
            alert(`Pipeline "${newName}" created successfully!`);
            loadCicdPipelines();
            loadCicdStats();
        }
    } catch (e) {
        alert('Failed to clone pipeline: ' + e.message);
    }
}

export async function cancelRun(runId) {
    if (!confirm('Cancel this run?')) return;
    try {
        const r = await fetch(`${API_BASE}/cicd/runs/${runId}/cancel`, { method: 'POST' });
        const d = await r.json();
        if (d.error) {
            alert('Error: ' + d.error);
        } else {
            loadCicdRuns();
            loadCicdStats();
        }
    } catch (e) {
        alert('Failed to cancel: ' + e.message);
    }
}

export async function retryPipelineRun(pipelineId) {
    if (!confirm('Retry this pipeline?')) return;
    try {
        const r = await fetch(`${API_BASE}/cicd/pipelines/${pipelineId}/trigger`, { method: 'POST' });
        const d = await r.json();
        if (d.error) {
            alert('Error: ' + d.error);
        } else {
            alert(`Pipeline re-triggered! Run #${d.run_number}`);
            loadCicdRuns();
            loadCicdStats();
        }
    } catch (e) {
        alert('Failed to retry: ' + e.message);
    }
}

export async function viewRunJobs(runId) {
    try {
        const r = await fetch(`${API_BASE}/cicd/runs/${runId}`);
        const run = await r.json();
        if (run.error) {
            alert('Error: ' + run.error);
            return;
        }
        // Show first job's logs
        if (run.jobs && run.jobs.length > 0) {
            openCicdLogModal(runId, run.jobs[0].name);
        } else {
            alert('No jobs found in this run');
        }
    } catch (e) {
        alert('Failed to load run: ' + e.message);
    }
}

export function openCicdLogModal(runId, jobName) {
    document.getElementById('cicd-log-modal').style.display = 'flex';
    document.getElementById('cicd-log-title').textContent = `Job Logs: ${jobName}`;
    document.getElementById('cicd-log-run').textContent = runId.slice(0, 12) + '...';
    document.getElementById('cicd-log-job').textContent = jobName;
    document.getElementById('cicd-log-status').textContent = 'Loading...';
    document.getElementById('cicd-log-content').textContent = 'Fetching logs...\n';

    // Fetch logs via REST first
    fetchJobLogs(runId, jobName);
}

async function fetchJobLogs(runId, jobName) {
    try {
        const r = await fetch(`${API_BASE}/cicd/runs/${runId}/jobs/${jobName}/logs?tail=500`);
        const d = await r.json();
        if (d.error) {
            document.getElementById('cicd-log-content').textContent = 'Error: ' + d.error;
            document.getElementById('cicd-log-status').textContent = 'Error';
        } else {
            document.getElementById('cicd-log-content').textContent = d.logs || 'No logs available';
            document.getElementById('cicd-log-status').textContent = 'Loaded';
        }
    } catch (e) {
        document.getElementById('cicd-log-content').textContent = 'Failed to fetch logs: ' + e.message;
    }
}

export function closeCicdLogModal() {
    document.getElementById('cicd-log-modal').style.display = 'none';
    if (cicdLogWs) {
        cicdLogWs.close();
        cicdLogWs = null;
    }
}

export async function deleteArtifact(id) {
    if (!confirm('Delete this artifact?')) return;
    try {
        const r = await fetch(`${API_BASE}/cicd/artifacts/${id}`, { method: 'DELETE' });
        const d = await r.json();
        if (d.error) {
            alert('Error: ' + d.error);
        } else {
            loadCicdArtifacts();
        }
    } catch (e) {
        alert('Failed to delete: ' + e.message);
    }
}

export function copyPipelineBadge(id, name) {
    const baseUrl = window.location.origin;
    const badgeUrl = `${baseUrl}${API_BASE}/cicd/pipelines/${id}/badge`;
    const markdown = `![${name}](${badgeUrl})`;

    window._copyText(markdown).then(() => {
        alert(`Badge markdown copied!\n\n${markdown}\n\nPaste this in your README.md`);
    }).catch(() => {
        prompt('Copy this badge markdown:', markdown);
    });
}

export function copyFreestyleJobBadge(id, name) {
    const baseUrl = window.location.origin;
    const badgeUrl = `${baseUrl}${API_BASE}/cicd/freestyle/jobs/${id}/badge`;
    const markdown = `![${name}](${badgeUrl})`;

    window._copyText(markdown).then(() => {
        alert(`Badge markdown copied!\n\n${markdown}\n\nPaste this in your README.md`);
    }).catch(() => {
        prompt('Copy this badge markdown:', markdown);
    });
}
