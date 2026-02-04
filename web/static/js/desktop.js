// Desktop Icon Management Module for GAGOS

import { API_BASE } from './app.js';
import { openWindow } from './windows.js';

// Desktop Icons Configuration
export const DESKTOP_ICONS = [
    { id: 'network', name: 'Network Tools', color: '#4ade80', svg: '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"></path>' },
    { id: 'kubernetes', name: 'Kubernetes', color: '#667eea', svg: '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path>' },
    { id: 'terminal', name: 'Terminal', color: '#fbbf24', svg: '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"></path>' },
    { id: 'about', name: 'About', color: '#a78bfa', svg: '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>' },
    { id: 'notepad', name: 'Notepad', color: '#f472b6', svg: '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"></path>' },
    { id: 'cicd', name: 'CI/CD', color: '#22d3ee', svg: '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path>' },
    { id: 'monitoring', name: 'Monitoring', color: '#10b981', svg: '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"></path>' },
    { id: 'devtools', name: 'Dev Tools', color: '#f97316', svg: '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"></path><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path>' },
    { id: 'postgres', name: 'PostgreSQL', color: '#336791', svg: '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"></path>' },
    { id: 'redis', name: 'Redis', color: '#dc382d', svg: '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M5 12h14M5 12l4-4m-4 4l4 4M19 12l-4-4m4 4l-4 4"></path><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 5v14"></path>' },
    { id: 'mysql', name: 'MySQL', color: '#00758f', svg: '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7"></path><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M20 7c0 2.21-3.582 4-8 4S4 9.21 4 7s3.582-4 8-4 8 1.79 8 4z"></path><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 12c0 2.21 3.582 4 8 4s8-1.79 8-4"></path>' },
    { id: 's3', name: 'S3 Storage', color: '#ff9900', svg: '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"/>' },
    { id: 'elasticsearch', name: 'Elasticsearch', color: '#f59e0b', svg: '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>' }
];

// Desktop preferences: slots array of 24 elements (icon id or null)
export let desktopPrefs = { slots: null };
export let iconEditMode = false;
let draggedIconId = null;

export function getDefaultSlots() {
    const slots = new Array(24).fill(null);
    DESKTOP_ICONS.forEach((icon, i) => { slots[i] = icon.id; });
    return slots;
}

export async function loadDesktopPreferences() {
    try {
        const r = await fetch(`${API_BASE}/preferences/desktop`);
        if (r.ok) {
            const data = await r.json();
            // Handle old format (icon_order) or new format (slots)
            if (data.slots) {
                desktopPrefs.slots = data.slots;
            } else if (data.icon_order) {
                // Convert old format to new slots format
                const slots = new Array(24).fill(null);
                const hidden = data.hidden || [];
                let slotIdx = 0;
                data.icon_order.forEach(id => {
                    if (!hidden.includes(id) && slotIdx < 24) {
                        slots[slotIdx++] = id;
                    }
                });
                desktopPrefs.slots = slots;
            }
        }
    } catch (e) {
        console.error('Failed to load desktop preferences:', e);
    }
    if (!desktopPrefs.slots) {
        desktopPrefs.slots = getDefaultSlots();
    }
    renderDesktopIcons();
}

export async function saveDesktopPreferences() {
    hideDesktopContextMenu();
    try {
        const r = await fetch(`${API_BASE}/preferences/desktop`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ slots: desktopPrefs.slots })
        });
        if (r.ok) {
            showNotification('Layout saved', 'success');
        } else {
            showNotification('Failed to save', 'error');
        }
    } catch (e) {
        console.error('Failed to save desktop preferences:', e);
        showNotification('Failed to save', 'error');
    }
}

export async function resetDesktopPreferences() {
    hideDesktopContextMenu();
    if (!confirm('Reset desktop layout to default?')) return;

    try {
        const r = await fetch(`${API_BASE}/preferences/desktop`, { method: 'DELETE' });
        if (r.ok) {
            desktopPrefs.slots = getDefaultSlots();
            renderDesktopIcons();
            showNotification('Layout reset', 'success');
        }
    } catch (e) {
        console.error('Failed to reset desktop preferences:', e);
    }
}

export function showNotification(message, type) {
    const toast = document.createElement('div');
    toast.style.cssText = `
        position: fixed;
        bottom: 60px;
        left: 50%;
        transform: translateX(-50%);
        padding: 10px 20px;
        background: ${type === 'success' ? '#10b981' : '#ef4444'};
        color: white;
        border-radius: 6px;
        font-size: 13px;
        z-index: 10001;
        box-shadow: 0 4px 12px rgba(0,0,0,0.3);
    `;
    toast.textContent = message;
    document.body.appendChild(toast);
    setTimeout(() => toast.remove(), 2000);
}

export function renderDesktopIcons() {
    const container = document.getElementById('desktop-icons-container');
    if (!container) return;

    const slots = desktopPrefs.slots || getDefaultSlots();
    container.innerHTML = '';

    // Create 24 slots
    for (let i = 0; i < 24; i++) {
        const iconId = slots[i];
        const slot = document.createElement('div');
        slot.className = 'desktop-slot' + (iconEditMode ? ' edit-mode' : '');
        slot.dataset.slotIndex = i;

        // Drag/drop on slots in edit mode
        if (iconEditMode) {
            slot.ondragover = (e) => {
                e.preventDefault();
                slot.classList.add('drag-over');
            };
            slot.ondragleave = () => {
                slot.classList.remove('drag-over');
            };
            slot.ondrop = (e) => {
                e.preventDefault();
                slot.classList.remove('drag-over');
                if (draggedIconId !== null) {
                    moveIconToSlot(draggedIconId, i);
                }
            };
        }

        // If slot has an icon, render it
        if (iconId) {
            const icon = DESKTOP_ICONS.find(ic => ic.id === iconId);
            if (icon) {
                const iconEl = createIconElement(icon);
                slot.appendChild(iconEl);
            }
        }

        container.appendChild(slot);
    }
}

export function createIconElement(icon) {
    const div = document.createElement('div');
    div.className = 'desktop-icon' + (iconEditMode ? ' edit-mode' : '');
    div.dataset.iconId = icon.id;
    div.style.position = 'relative';

    div.ondblclick = () => { if (!iconEditMode) openWindow(icon.id); };

    if (iconEditMode) {
        div.draggable = true;
        div.ondragstart = (e) => {
            draggedIconId = icon.id;
            div.classList.add('dragging');
            e.dataTransfer.effectAllowed = 'move';
        };
        div.ondragend = () => {
            div.classList.remove('dragging');
            draggedIconId = null;
        };
    }

    const hideBtn = iconEditMode ? `<button class="icon-hide-btn" onclick="event.stopPropagation(); window.hideIcon('${icon.id}')" title="Hide"><svg fill="none" stroke="white" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M6 18L18 6M6 6l12 12"></path></svg></button>` : '';
    div.innerHTML = `${hideBtn}<svg fill="none" stroke="${icon.color}" viewBox="0 0 24 24">${icon.svg}</svg><span>${icon.name}</span>`;
    return div;
}

export function hideIcon(iconId) {
    const slots = desktopPrefs.slots || getDefaultSlots();
    const idx = slots.indexOf(iconId);
    if (idx !== -1) {
        slots[idx] = null;
        desktopPrefs.slots = slots;
        renderDesktopIcons();
    }
}

export function showIcon(iconId) {
    const slots = desktopPrefs.slots || getDefaultSlots();
    // Find first empty slot
    const emptyIdx = slots.findIndex(s => s === null);
    if (emptyIdx !== -1) {
        slots[emptyIdx] = iconId;
        desktopPrefs.slots = slots;
        renderDesktopIcons();
        updateHiddenIconsMenu();
    }
}

export function getHiddenIcons() {
    const slots = desktopPrefs.slots || getDefaultSlots();
    const visibleIds = slots.filter(Boolean);
    return DESKTOP_ICONS.filter(icon => !visibleIds.includes(icon.id));
}

export function updateHiddenIconsMenu() {
    const submenu = document.getElementById('hidden-icons-menu');
    if (!submenu) return;
    const hidden = getHiddenIcons();
    if (hidden.length === 0) {
        submenu.innerHTML = '<div class="context-menu-item" style="color:#666;cursor:default;">No hidden icons</div>';
    } else {
        submenu.innerHTML = hidden.map(icon =>
            `<div class="context-menu-item" onclick="window.showIcon('${icon.id}'); window.hideDesktopContextMenu();">${icon.name}</div>`
        ).join('');
    }
}

export function moveIconToSlot(iconId, targetSlot) {
    const slots = desktopPrefs.slots || getDefaultSlots();

    // Find current slot of the icon
    const currentSlot = slots.indexOf(iconId);

    // Swap: if target has an icon, swap them; otherwise just move
    const targetIconId = slots[targetSlot];

    if (currentSlot !== -1) {
        slots[currentSlot] = targetIconId; // Put target's icon (or null) in source slot
    }
    slots[targetSlot] = iconId;

    desktopPrefs.slots = slots;
    renderDesktopIcons();
}

export function showDesktopContextMenu(e) {
    if (e.target.closest('.desktop-icon') || e.target.closest('.window')) return;

    e.preventDefault();
    const menu = document.getElementById('desktop-context-menu');
    if (!menu) return;

    let x = e.clientX;
    let y = e.clientY;

    menu.style.display = 'block';
    const menuRect = menu.getBoundingClientRect();
    if (x + menuRect.width > window.innerWidth) x = window.innerWidth - menuRect.width - 10;
    if (y + menuRect.height > window.innerHeight) y = window.innerHeight - menuRect.height - 10;

    menu.style.left = x + 'px';
    menu.style.top = y + 'px';

    document.getElementById('edit-mode-label').textContent = iconEditMode ? 'Done Editing' : 'Edit Layout';
    updateHiddenIconsMenu();
}

export function hideDesktopContextMenu() {
    const menu = document.getElementById('desktop-context-menu');
    if (menu) menu.style.display = 'none';
}

export function toggleIconEditMode() {
    iconEditMode = !iconEditMode;
    document.getElementById('edit-mode-label').textContent = iconEditMode ? 'Done Editing' : 'Edit Layout';
    renderDesktopIcons();
    hideDesktopContextMenu();
}
