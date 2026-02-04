// Notification Management Module for GAGOS CI/CD

import { API_BASE } from './app.js';
import { formatTime } from './utils.js';

let notificationConfigs = [];
let editingNotificationId = null;
let selectedEvents = [];

const ALL_EVENTS = [
    { value: 'build_started', label: 'Build Started', group: 'Freestyle' },
    { value: 'build_succeeded', label: 'Build Succeeded', group: 'Freestyle' },
    { value: 'build_failed', label: 'Build Failed', group: 'Freestyle' },
    { value: 'build_cancelled', label: 'Build Cancelled', group: 'Freestyle' },
    { value: 'run_started', label: 'Run Started', group: 'Pipeline' },
    { value: 'run_succeeded', label: 'Run Succeeded', group: 'Pipeline' },
    { value: 'run_failed', label: 'Run Failed', group: 'Pipeline' },
    { value: 'run_cancelled', label: 'Run Cancelled', group: 'Pipeline' }
];

export async function loadNotifications() {
    try {
        const r = await fetch(`${API_BASE}/cicd/notifications`);
        const d = await r.json();
        notificationConfigs = d.notifications || [];
        renderNotificationsTable();
    } catch (e) {
        console.error('Failed to load notifications:', e);
    }
}

function renderNotificationsTable() {
    const tbody = document.getElementById('notifications-tbody');
    tbody.innerHTML = '';

    if (notificationConfigs.length === 0) {
        tbody.innerHTML = `<tr><td colspan="6" style="text-align:center;color:#6a6a7a;padding:30px;">
            No notification configs. Click "New Notification" to create one.
        </td></tr>`;
        return;
    }

    notificationConfigs.forEach(n => {
        const eventCount = n.events ? n.events.length : 0;
        const scope = getScopeText(n);
        const enabledClass = n.enabled ? 'status-running' : 'status-failed';
        const enabledText = n.enabled ? 'Enabled' : 'Disabled';

        tbody.innerHTML += `<tr>
            <td><strong>${escapeHtml(n.name)}</strong></td>
            <td><span style="color:#22d3ee;">${n.type || 'webhook'}</span></td>
            <td>${eventCount} event${eventCount !== 1 ? 's' : ''}</td>
            <td style="color:#8a8a9a;">${scope}</td>
            <td class="${enabledClass}">${enabledText}</td>
            <td class="action-cell">
                <button class="row-action-btn describe" onclick="editNotification('${n.id}')" title="Edit">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/></svg>
                </button>
                <button class="row-action-btn ${n.enabled ? 'delete' : 'logs'}" onclick="toggleNotification('${n.id}')" title="${n.enabled ? 'Disable' : 'Enable'}">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;">
                        ${n.enabled
                            ? '<path stroke-linecap="round" stroke-linejoin="round" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636"/>'
                            : '<path stroke-linecap="round" stroke-linejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>'
                        }
                    </svg>
                </button>
                <button class="row-action-btn delete" onclick="deleteNotification('${n.id}')" title="Delete">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:14px;height:14px;"><path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
                </button>
            </td>
        </tr>`;
    });
}

function getScopeText(notification) {
    if (notification.job_ids && notification.job_ids.length > 0) {
        return `${notification.job_ids.length} job(s)`;
    }
    if (notification.pipeline_ids && notification.pipeline_ids.length > 0) {
        return `${notification.pipeline_ids.length} pipeline(s)`;
    }
    return 'All';
}

function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

export function showCreateNotificationModal() {
    editingNotificationId = null;
    selectedEvents = [];
    document.getElementById('notification-id').value = '';
    document.getElementById('notification-modal-title').textContent = 'Create Notification';
    renderNotificationForm({});
    document.getElementById('notification-modal').style.display = 'flex';
}

window.editNotification = async function(id) {
    try {
        const r = await fetch(`${API_BASE}/cicd/notifications/${id}`);
        const notification = await r.json();
        if (notification.error) {
            alert('Error: ' + notification.error);
            return;
        }
        editingNotificationId = id;
        selectedEvents = notification.events || [];
        document.getElementById('notification-id').value = id;
        document.getElementById('notification-modal-title').textContent = 'Edit Notification';
        renderNotificationForm(notification);
        document.getElementById('notification-modal').style.display = 'flex';
    } catch (e) {
        alert('Failed to load notification: ' + e.message);
    }
};

function renderNotificationForm(data) {
    const body = document.getElementById('notification-modal-body');

    const freestyleEvents = ALL_EVENTS.filter(e => e.group === 'Freestyle');
    const pipelineEvents = ALL_EVENTS.filter(e => e.group === 'Pipeline');

    body.innerHTML = `
        <!-- Webhook Configuration Section -->
        <div style="background:rgba(34,211,238,0.05);border:1px solid rgba(34,211,238,0.2);border-radius:8px;padding:16px;margin-bottom:16px;">
            <h4 style="color:#22d3ee;margin:0 0 14px 0;font-size:13px;display:flex;align-items:center;gap:8px;">
                <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:16px;height:16px;"><path stroke-linecap="round" stroke-linejoin="round" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"/></svg>
                Webhook Configuration
            </h4>
            <div class="form-group" style="margin-bottom:12px;">
                <label style="font-size:12px;color:#8a8a9a;margin-bottom:6px;display:block;">Name <span style="color:#ef4444;">*</span></label>
                <input type="text" id="notification-name" required placeholder="My Build Alerts" value="${escapeHtml(data.name || '')}"
                    style="width:100%;padding:10px 12px;background:#1a1a2e;border:1px solid #3a3a4e;border-radius:6px;color:#fff;font-size:14px;">
            </div>
            <div class="form-group" style="margin-bottom:12px;">
                <label style="font-size:12px;color:#8a8a9a;margin-bottom:6px;display:block;">Webhook URL <span style="color:#ef4444;">*</span></label>
                <input type="url" id="notification-url" required placeholder="https://example.com/webhook" value="${escapeHtml(data.url || '')}"
                    style="width:100%;padding:10px 12px;background:#1a1a2e;border:1px solid #3a3a4e;border-radius:6px;color:#fff;font-size:14px;">
            </div>
            <div class="form-group" style="margin-bottom:0;">
                <label style="font-size:12px;color:#8a8a9a;margin-bottom:6px;display:block;">
                    Secret
                    <span style="font-size:11px;color:#6a6a7a;margin-left:6px;">(optional, for HMAC signature)</span>
                </label>
                <input type="text" id="notification-secret" placeholder="hmac-secret-key" value="${escapeHtml(data.secret || '')}"
                    style="width:100%;padding:10px 12px;background:#1a1a2e;border:1px solid #3a3a4e;border-radius:6px;color:#fff;font-size:14px;font-family:monospace;">
            </div>
            <select id="notification-type" style="display:none;">
                <option value="webhook" ${data.type === 'webhook' || !data.type ? 'selected' : ''}>Webhook</option>
            </select>
        </div>

        <!-- Events Section -->
        <div style="background:rgba(251,191,36,0.05);border:1px solid rgba(251,191,36,0.2);border-radius:8px;padding:16px;margin-bottom:16px;">
            <h4 style="color:#fbbf24;margin:0 0 14px 0;font-size:13px;display:flex;align-items:center;gap:8px;">
                <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:16px;height:16px;"><path stroke-linecap="round" stroke-linejoin="round" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"/></svg>
                Events <span style="color:#ef4444;margin-left:4px;">*</span>
            </h4>

            <!-- Freestyle Events -->
            <div style="margin-bottom:14px;">
                <div style="font-size:11px;color:#6a6a7a;text-transform:uppercase;font-weight:600;margin-bottom:8px;display:flex;align-items:center;gap:6px;">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:12px;height:12px;"><path stroke-linecap="round" stroke-linejoin="round" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>
                    Freestyle Build Events
                </div>
                <div style="display:grid;grid-template-columns:1fr 1fr;gap:6px;">
                    ${freestyleEvents.map(e => `
                        <label class="event-label" style="display:flex;align-items:center;gap:10px;cursor:pointer;padding:10px 12px;background:#1a1a2e;border-radius:6px;border:2px solid ${selectedEvents.includes(e.value) ? '#fbbf24' : '#3a3a4e'};transition:all 0.2s;">
                            <input type="checkbox" class="event-checkbox" value="${e.value}" ${selectedEvents.includes(e.value) ? 'checked' : ''}
                                onchange="updateSelectedEvents();this.parentElement.style.borderColor=this.checked?'#fbbf24':'#3a3a4e';"
                                style="width:16px;height:16px;accent-color:#fbbf24;">
                            <span style="font-size:13px;color:#e0e0e0;">${e.label}</span>
                            ${e.value.includes('failed') ? '<span style="margin-left:auto;font-size:10px;background:rgba(239,68,68,0.2);color:#ef4444;padding:2px 6px;border-radius:4px;">Alert</span>' : ''}
                            ${e.value.includes('succeeded') ? '<span style="margin-left:auto;font-size:10px;background:rgba(74,222,128,0.2);color:#4ade80;padding:2px 6px;border-radius:4px;">Success</span>' : ''}
                        </label>
                    `).join('')}
                </div>
            </div>

            <!-- Pipeline Events -->
            <div>
                <div style="font-size:11px;color:#6a6a7a;text-transform:uppercase;font-weight:600;margin-bottom:8px;display:flex;align-items:center;gap:6px;">
                    <svg fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" style="width:12px;height:12px;"><path stroke-linecap="round" stroke-linejoin="round" d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z"/></svg>
                    Pipeline Run Events
                </div>
                <div style="display:grid;grid-template-columns:1fr 1fr;gap:6px;">
                    ${pipelineEvents.map(e => `
                        <label class="event-label" style="display:flex;align-items:center;gap:10px;cursor:pointer;padding:10px 12px;background:#1a1a2e;border-radius:6px;border:2px solid ${selectedEvents.includes(e.value) ? '#fbbf24' : '#3a3a4e'};transition:all 0.2s;">
                            <input type="checkbox" class="event-checkbox" value="${e.value}" ${selectedEvents.includes(e.value) ? 'checked' : ''}
                                onchange="updateSelectedEvents();this.parentElement.style.borderColor=this.checked?'#fbbf24':'#3a3a4e';"
                                style="width:16px;height:16px;accent-color:#fbbf24;">
                            <span style="font-size:13px;color:#e0e0e0;">${e.label}</span>
                            ${e.value.includes('failed') ? '<span style="margin-left:auto;font-size:10px;background:rgba(239,68,68,0.2);color:#ef4444;padding:2px 6px;border-radius:4px;">Alert</span>' : ''}
                            ${e.value.includes('succeeded') ? '<span style="margin-left:auto;font-size:10px;background:rgba(74,222,128,0.2);color:#4ade80;padding:2px 6px;border-radius:4px;">Success</span>' : ''}
                        </label>
                    `).join('')}
                </div>
            </div>
        </div>

        <!-- Status & Info Section -->
        <div style="display:grid;grid-template-columns:1fr 1fr;gap:12px;">
            <!-- Enabled Toggle -->
            <div style="background:rgba(74,222,128,0.05);border:1px solid rgba(74,222,128,0.2);border-radius:8px;padding:14px;">
                <label style="display:flex;align-items:center;gap:12px;cursor:pointer;">
                    <div style="position:relative;width:44px;height:24px;">
                        <input type="checkbox" id="notification-enabled" ${data.enabled !== false ? 'checked' : ''}
                            style="opacity:0;width:100%;height:100%;position:absolute;cursor:pointer;z-index:1;"
                            onchange="this.nextElementSibling.style.background=this.checked?'#4ade80':'#3a3a4e';this.nextElementSibling.querySelector('div').style.transform=this.checked?'translateX(20px)':'translateX(0)';">
                        <div style="position:absolute;top:0;left:0;right:0;bottom:0;background:${data.enabled !== false ? '#4ade80' : '#3a3a4e'};border-radius:12px;transition:0.3s;">
                            <div style="position:absolute;top:2px;left:2px;width:20px;height:20px;background:#fff;border-radius:50%;transition:0.3s;transform:${data.enabled !== false ? 'translateX(20px)' : 'translateX(0)'};"></div>
                        </div>
                    </div>
                    <div>
                        <div style="font-size:13px;color:#e0e0e0;font-weight:500;">Enabled</div>
                        <div style="font-size:11px;color:#6a6a7a;">Receive notifications</div>
                    </div>
                </label>
            </div>

            <!-- Payload Info -->
            <div style="background:rgba(139,92,246,0.05);border:1px solid rgba(139,92,246,0.2);border-radius:8px;padding:14px;">
                <div style="font-size:11px;color:#8b5cf6;text-transform:uppercase;font-weight:600;margin-bottom:6px;">Payload Format</div>
                <code style="font-size:10px;color:#8a8a9a;display:block;line-height:1.4;">
                    {"event": "...", "timestamp": "...", "build": {...}}
                </code>
            </div>
        </div>
    `;
}

window.updateSelectedEvents = function() {
    selectedEvents = [];
    document.querySelectorAll('.event-checkbox:checked').forEach(cb => {
        selectedEvents.push(cb.value);
    });
};

export function closeNotificationModal() {
    document.getElementById('notification-modal').style.display = 'none';
    editingNotificationId = null;
    selectedEvents = [];
}

export async function saveNotification(event) {
    event.preventDefault();

    const name = document.getElementById('notification-name').value.trim();
    const type = document.getElementById('notification-type').value;
    const url = document.getElementById('notification-url').value.trim();
    const secret = document.getElementById('notification-secret').value.trim();
    const enabled = document.getElementById('notification-enabled').checked;

    // Get selected events
    updateSelectedEvents();

    if (!name) {
        alert('Name is required');
        return;
    }
    if (!url) {
        alert('Webhook URL is required');
        return;
    }
    if (selectedEvents.length === 0) {
        alert('Please select at least one event');
        return;
    }

    const payload = {
        name,
        type,
        url,
        secret: secret || undefined,
        events: selectedEvents,
        enabled
    };

    try {
        let r;
        if (editingNotificationId) {
            r = await fetch(`${API_BASE}/cicd/notifications/${editingNotificationId}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
        } else {
            r = await fetch(`${API_BASE}/cicd/notifications`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
        }

        const d = await r.json();
        if (d.error) {
            alert('Error: ' + d.error);
            return;
        }

        closeNotificationModal();
        loadNotifications();
    } catch (e) {
        alert('Failed to save notification: ' + e.message);
    }
}

window.toggleNotification = async function(id) {
    const notification = notificationConfigs.find(n => n.id === id);
    if (!notification) return;

    const newEnabled = !notification.enabled;

    try {
        const r = await fetch(`${API_BASE}/cicd/notifications/${id}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...notification, enabled: newEnabled })
        });

        const d = await r.json();
        if (d.error) {
            alert('Error: ' + d.error);
            return;
        }

        loadNotifications();
    } catch (e) {
        alert('Failed to update notification: ' + e.message);
    }
};

window.deleteNotification = async function(id) {
    if (!confirm('Delete this notification config?')) return;

    try {
        const r = await fetch(`${API_BASE}/cicd/notifications/${id}`, { method: 'DELETE' });
        const d = await r.json();
        if (d.error) {
            alert('Error: ' + d.error);
            return;
        }

        loadNotifications();
    } catch (e) {
        alert('Failed to delete notification: ' + e.message);
    }
};

// Make functions available globally
window.showCreateNotificationModal = showCreateNotificationModal;
window.closeNotificationModal = closeNotificationModal;
window.saveNotification = saveNotification;
