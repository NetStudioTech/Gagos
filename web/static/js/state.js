// State Management Module for GAGOS

export const STORAGE_KEY = 'gagos_state';
export const STATE_VERSION = 2;

// Import DESKTOP_ICONS from desktop.js (circular dependency handled via late import)
let getDesktopIconsFunc = null;
export function setDesktopIconsGetter(fn) {
    getDesktopIconsFunc = fn;
}

export function getDefaultState() {
    const desktopIcons = getDesktopIconsFunc ? getDesktopIconsFunc() : [];
    return {
        version: STATE_VERSION,
        windows: {
            network: { left: 150, top: 50, width: 800, height: 600, visible: false, maximized: false },
            kubernetes: { left: 200, top: 80, width: 900, height: 650, visible: false, maximized: false },
            terminal: { left: 250, top: 100, width: 600, height: 400, visible: false, maximized: false },
            about: { left: 300, top: 150, width: 400, height: 300, visible: false, maximized: false },
            notepad: { left: 350, top: 120, width: 500, height: 400, visible: false, maximized: false },
            cicd: { left: 180, top: 60, width: 1000, height: 700, visible: false, maximized: false }
        },
        activeTabs: { network: 'ping', kubernetes: 'overview', cicd: 'overview' },
        activeWindow: null,
        notepadContent: '',
        desktopIcons: {
            order: desktopIcons.map(i => i.id),
            hidden: []
        }
    };
}

export function loadState() {
    try {
        const saved = localStorage.getItem(STORAGE_KEY);
        if (saved) {
            const state = JSON.parse(saved);
            if (state.version === STATE_VERSION) return state;
        }
    } catch (e) {
        console.warn('Failed to load state:', e);
    }
    return getDefaultState();
}

let saveTimeout = null;

export function saveState() {
    if (saveTimeout) clearTimeout(saveTimeout);
    saveTimeout = setTimeout(() => {
        const state = getCurrentState();
        localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
    }, 300);
}

// These will be set by app.js after all modules load
let getActiveWindowFunc = null;
let showNetTabFunc = null;
let showK8sTabFunc = null;

export function setStateHelpers(helpers) {
    getActiveWindowFunc = helpers.getActiveWindow;
    showNetTabFunc = helpers.showNetTab;
    showK8sTabFunc = helpers.showK8sTab;
}

export function getCurrentState() {
    const state = getDefaultState();
    ['network', 'kubernetes', 'terminal', 'about', 'notepad'].forEach(id => {
        const win = document.getElementById('window-' + id);
        if (win) {
            state.windows[id] = {
                left: parseInt(win.style.left) || state.windows[id].left,
                top: parseInt(win.style.top) || state.windows[id].top,
                width: parseInt(win.style.width) || state.windows[id].width,
                height: parseInt(win.style.height) || state.windows[id].height,
                visible: win.style.display !== 'none',
                maximized: win.dataset.maximized === 'true'
            };
        }
    });
    const netTab = document.querySelector('#window-network .tab-btn.active');
    if (netTab) {
        const match = netTab.getAttribute('onclick')?.match(/showNetTab\('(\w+)'\)/);
        if (match) state.activeTabs.network = match[1];
    }
    const k8sTab = document.querySelector('#window-kubernetes .tab-btn.active');
    if (k8sTab) {
        const match = k8sTab.getAttribute('onclick')?.match(/showK8sTab\('(\w+)'\)/);
        if (match) state.activeTabs.kubernetes = match[1];
    }
    state.activeWindow = getActiveWindowFunc ? getActiveWindowFunc() : null;
    const notepad = document.getElementById('notepad-content');
    if (notepad) state.notepadContent = notepad.value;
    return state;
}

export function restoreState() {
    const state = loadState();
    // Only restore window positions and sizes, not visibility
    // Windows should start closed on fresh page load
    ['network', 'kubernetes', 'terminal', 'about', 'notepad'].forEach(id => {
        const win = document.getElementById('window-' + id);
        const ws = state.windows[id];
        if (win && ws) {
            win.style.left = ws.left + 'px';
            win.style.top = ws.top + 'px';
            win.style.width = ws.width + 'px';
            win.style.height = ws.height + 'px';
            // Don't auto-open windows - let user open them manually
        }
    });
    if (state.activeTabs.network && showNetTabFunc) showNetTabFunc(state.activeTabs.network);
    if (state.activeTabs.kubernetes && showK8sTabFunc) showK8sTabFunc(state.activeTabs.kubernetes);
    // Desktop icons loaded from API via loadDesktopPreferences()
}
