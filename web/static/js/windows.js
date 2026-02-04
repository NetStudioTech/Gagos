// Window Management Module for GAGOS

import { saveState } from './state.js';

// Window state
export let windows = {};
export let activeWindow = null;
let dragState = null;
let resizeState = null;
let zIndex = 100;

// Callbacks for window open events
let onWindowOpenCallbacks = {};

export function setWindowOpenCallback(windowId, callback) {
    onWindowOpenCallbacks[windowId] = callback;
}

export function getActiveWindow() {
    return activeWindow;
}

export function openWindow(id) {
    const win = document.getElementById('window-' + id);
    win.style.display = 'flex';
    win.style.zIndex = ++zIndex;
    windows[id] = true;
    setActiveWindow(id);
    updateTaskbar();

    // Call window-specific open callback
    if (onWindowOpenCallbacks[id]) {
        onWindowOpenCallbacks[id]();
    }

    saveState();
}

export function closeWindow(id) {
    document.getElementById('window-' + id).style.display = 'none';
    delete windows[id];
    updateTaskbar();
    if (activeWindow === id) activeWindow = null;
    saveState();
}

export function minimizeWindow(id) {
    document.getElementById('window-' + id).style.display = 'none';
}

export function maximizeWindow(id) {
    const win = document.getElementById('window-' + id);
    const btn = document.getElementById('max-btn-' + id);
    if (win.dataset.maximized === 'true') {
        win.style.width = win.dataset.prevWidth;
        win.style.height = win.dataset.prevHeight;
        win.style.left = win.dataset.prevLeft;
        win.style.top = win.dataset.prevTop;
        win.dataset.maximized = 'false';
        if (btn) btn.classList.remove('maximized');
    } else {
        win.dataset.prevWidth = win.style.width;
        win.dataset.prevHeight = win.style.height;
        win.dataset.prevLeft = win.style.left;
        win.dataset.prevTop = win.style.top;
        win.style.width = '100%';
        win.style.height = 'calc(100vh - 48px)';
        win.style.left = '0';
        win.style.top = '0';
        win.dataset.maximized = 'true';
        if (btn) btn.classList.add('maximized');
    }
    saveState();
}

export function setActiveWindow(id) {
    document.querySelectorAll('.window').forEach(w => w.classList.remove('active'));
    document.querySelectorAll('.taskbar-item').forEach(t => t.classList.remove('active'));

    const win = document.getElementById('window-' + id);
    if (win) {
        win.classList.add('active');
        win.style.zIndex = ++zIndex;
    }

    const taskbarItem = document.querySelector(`.taskbar-item[data-id="${id}"]`);
    if (taskbarItem) taskbarItem.classList.add('active');

    activeWindow = id;
}

export function bringToFront(id) {
    setActiveWindow(id);
}

export function updateTaskbar() {
    const container = document.getElementById('taskbar-apps');
    container.innerHTML = '';

    const icons = {
        network: '<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"></path></svg>',
        kubernetes: '<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path></svg>',
        terminal: '<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"></path></svg>',
        about: '<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>',
        notepad: '<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"></path></svg>',
        cicd: '<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path></svg>',
        monitoring: '<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"></path></svg>',
        devtools: '<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"></path><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path></svg>',
        postgres: '<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"></path></svg>',
        redis: '<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12l4-4m-4 4l4 4M19 12l-4-4m4 4l-4 4"></path><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 5v14"></path></svg>',
        mysql: '<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7"></path><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7c0 2.21-3.582 4-8 4S4 9.21 4 7s3.582-4 8-4 8 1.79 8 4z"></path><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 12c0 2.21 3.582 4 8 4s8-1.79 8-4"></path></svg>',
        s3: '<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"/></svg>',
        elasticsearch: '<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/></svg>'
    };

    const names = {
        network: 'Network Tools',
        kubernetes: 'Kubernetes',
        terminal: 'Terminal',
        about: 'About',
        notepad: 'Notepad',
        cicd: 'CI/CD',
        monitoring: 'Monitoring',
        devtools: 'Dev Tools',
        postgres: 'PostgreSQL',
        redis: 'Redis',
        mysql: 'MySQL',
        s3: 'S3 Storage',
        elasticsearch: 'Elasticsearch'
    };

    for (const id in windows) {
        const btn = document.createElement('button');
        btn.className = 'taskbar-item' + (activeWindow === id ? ' active' : '');
        btn.dataset.id = id;
        btn.innerHTML = (icons[id] || '') + (names[id] || id);
        btn.onclick = () => {
            const win = document.getElementById('window-' + id);
            if (win.style.display === 'none') {
                win.style.display = 'flex';
            }
            setActiveWindow(id);
        };
        container.appendChild(btn);
    }
}

// Drag & Resize
export function startDrag(e, id) {
    if (e.target.classList.contains('window-btn')) return;
    const win = document.getElementById('window-' + id);
    if (win.dataset.maximized === 'true') return;

    setActiveWindow(id);
    dragState = {
        id,
        startX: e.clientX,
        startY: e.clientY,
        startLeft: parseInt(win.style.left),
        startTop: parseInt(win.style.top)
    };
    document.addEventListener('mousemove', onDrag);
    document.addEventListener('mouseup', stopDrag);
}

function onDrag(e) {
    if (!dragState) return;
    const win = document.getElementById('window-' + dragState.id);
    win.style.left = (dragState.startLeft + e.clientX - dragState.startX) + 'px';
    win.style.top = Math.max(0, dragState.startTop + e.clientY - dragState.startY) + 'px';
}

function stopDrag() {
    if (dragState) saveState();
    dragState = null;
    document.removeEventListener('mousemove', onDrag);
    document.removeEventListener('mouseup', stopDrag);
}

export function startResize(e, id) {
    const win = document.getElementById('window-' + id);
    resizeState = {
        id,
        startX: e.clientX,
        startY: e.clientY,
        startWidth: parseInt(win.style.width),
        startHeight: parseInt(win.style.height)
    };
    document.addEventListener('mousemove', onResize);
    document.addEventListener('mouseup', stopResize);
}

function onResize(e) {
    if (!resizeState) return;
    const win = document.getElementById('window-' + resizeState.id);
    win.style.width = Math.max(400, resizeState.startWidth + e.clientX - resizeState.startX) + 'px';
    win.style.height = Math.max(300, resizeState.startHeight + e.clientY - resizeState.startY) + 'px';
}

function stopResize() {
    if (resizeState) saveState();
    resizeState = null;
    document.removeEventListener('mousemove', onResize);
    document.removeEventListener('mouseup', stopResize);
}

export function toggleStartMenu() {
    // Minimize all windows (show desktop)
    let anyVisible = false;
    for (const id in windows) {
        const win = document.getElementById('window-' + id);
        if (win && win.style.display !== 'none') {
            anyVisible = true;
            break;
        }
    }

    if (anyVisible) {
        // Minimize all windows
        for (const id in windows) {
            const win = document.getElementById('window-' + id);
            if (win) win.style.display = 'none';
        }
    } else {
        // Restore all windows
        for (const id in windows) {
            const win = document.getElementById('window-' + id);
            if (win) win.style.display = 'flex';
        }
    }
}
