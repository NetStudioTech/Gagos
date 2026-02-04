// Notepad Module for GAGOS
// Persists notepad tabs and content to server-side storage via /api/v1/notepad/

const NOTEPAD_META_KEY = '__notepad_meta__';

let notepadTabs = [{ id: 'tab-1', name: 'Untitled 1', content: '' }];
let activeNotepadTab = 'tab-1';
let tabCounter = 1;
let saveContentTimeout = null;
let saveMetaTimeout = null;
let initialized = false;

export function getNotepadTabs() { return notepadTabs; }
export function getActiveNotepadTab() { return activeNotepadTab; }

// ---- API helpers ----

async function apiGet(key) {
    try {
        const res = await fetch(`/api/v1/notepad/${encodeURIComponent(key)}`);
        if (!res.ok) return null;
        return await res.json();
    } catch { return null; }
}

async function apiSave(key, content) {
    try {
        await fetch(`/api/v1/notepad/${encodeURIComponent(key)}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ content })
        });
    } catch (e) {
        console.warn('Notepad save failed:', e);
    }
}

async function apiDelete(key) {
    try {
        await fetch(`/api/v1/notepad/${encodeURIComponent(key)}`, { method: 'DELETE' });
    } catch (e) {
        console.warn('Notepad delete failed:', e);
    }
}

// ---- Meta persistence ----

function saveMetaToServer() {
    if (saveMetaTimeout) clearTimeout(saveMetaTimeout);
    saveMetaTimeout = setTimeout(() => {
        const meta = {
            tabs: notepadTabs.map(t => ({ id: t.id, name: t.name })),
            activeTab: activeNotepadTab,
            tabCounter: tabCounter
        };
        apiSave(NOTEPAD_META_KEY, JSON.stringify(meta));
    }, 500);
}

// ---- Content persistence (debounced) ----

function saveCurrentTabToServer() {
    if (saveContentTimeout) clearTimeout(saveContentTimeout);
    saveContentTimeout = setTimeout(() => {
        const tab = notepadTabs.find(t => t.id === activeNotepadTab);
        if (tab) {
            apiSave(tab.id, tab.content || '');
        }
    }, 800);
}

// ---- Init: load from server ----

export async function initNotepad() {
    if (initialized) {
        renderNotepadTabs();
        const textarea = document.getElementById('notepad-content');
        const activeTab = notepadTabs.find(t => t.id === activeNotepadTab);
        if (textarea && activeTab) textarea.value = activeTab.content || '';
        return;
    }
    try {
        const metaRes = await apiGet(NOTEPAD_META_KEY);
        if (metaRes && metaRes.content) {
            const meta = JSON.parse(metaRes.content);
            if (meta.tabs && meta.tabs.length > 0) {
                tabCounter = meta.tabCounter || meta.tabs.length;
                activeNotepadTab = meta.activeTab || meta.tabs[0].id;

                // Load each tab's content from server
                const loadedTabs = [];
                for (const tabMeta of meta.tabs) {
                    const tabData = await apiGet(tabMeta.id);
                    loadedTabs.push({
                        id: tabMeta.id,
                        name: tabMeta.name,
                        content: (tabData && tabData.content) || ''
                    });
                }
                notepadTabs = loadedTabs;

                // Verify active tab exists
                if (!notepadTabs.find(t => t.id === activeNotepadTab)) {
                    activeNotepadTab = notepadTabs[0].id;
                }
            }
        }
    } catch (e) {
        console.warn('Failed to load notepad from server, using defaults:', e);
    }

    // Set textarea to active tab content
    const textarea = document.getElementById('notepad-content');
    const activeTab = notepadTabs.find(t => t.id === activeNotepadTab);
    if (textarea && activeTab) {
        textarea.value = activeTab.content || '';
    }

    initialized = true;
    renderNotepadTabs();
}

// ---- Rename ----

export function renameNotepadTab(tabId) {
    const tab = notepadTabs.find(t => t.id === tabId);
    if (!tab) return;

    const tabEl = document.querySelector(`.notepad-tab[data-tab-id="${tabId}"] .tab-name`);
    if (!tabEl) return;

    const input = document.createElement('input');
    input.type = 'text';
    input.value = tab.name;
    input.style.cssText = 'background:#2a2a3a;border:1px solid #667eea;color:#fff;font-size:12px;padding:1px 4px;border-radius:3px;width:80px;outline:none;';

    const finish = () => {
        const newName = input.value.trim() || tab.name;
        tab.name = newName;
        renderNotepadTabs();
        saveMetaToServer();
    };

    input.onblur = finish;
    input.onkeydown = (e) => {
        if (e.key === 'Enter') { e.preventDefault(); input.blur(); }
        if (e.key === 'Escape') { input.value = tab.name; input.blur(); }
    };

    tabEl.replaceWith(input);
    input.focus();
    input.select();
}

// ---- Render tabs ----

export function renderNotepadTabs() {
    const tabsContainer = document.getElementById('notepad-tabs');
    if (!tabsContainer) return;

    tabsContainer.innerHTML = '';
    notepadTabs.forEach(tab => {
        const tabEl = document.createElement('button');
        tabEl.className = 'notepad-tab' + (tab.id === activeNotepadTab ? ' active' : '');
        tabEl.setAttribute('data-tab-id', tab.id);
        tabEl.innerHTML = `<span class="tab-name">${tab.name}</span>${notepadTabs.length > 1 ? '<span class="tab-close" onclick="event.stopPropagation(); window.closeNotepadTab(\'' + tab.id + '\')">x</span>' : ''}`;
        tabEl.onclick = () => switchNotepadTab(tab.id);
        tabEl.ondblclick = (e) => { e.stopPropagation(); renameNotepadTab(tab.id); };
        tabsContainer.appendChild(tabEl);
    });

    // Add "new tab" button
    const addBtn = document.createElement('button');
    addBtn.className = 'notepad-tab-add';
    addBtn.textContent = '+';
    addBtn.onclick = addNotepadTab;
    tabsContainer.appendChild(addBtn);
}

// ---- Switch tab ----

export function switchNotepadTab(tabId) {
    // Save current content
    const currentTab = notepadTabs.find(t => t.id === activeNotepadTab);
    const textarea = document.getElementById('notepad-content');
    if (currentTab && textarea) {
        currentTab.content = textarea.value;
        apiSave(currentTab.id, currentTab.content);
    }

    // Switch to new tab
    activeNotepadTab = tabId;
    const newTab = notepadTabs.find(t => t.id === tabId);
    if (newTab && textarea) {
        textarea.value = newTab.content || '';
    }

    renderNotepadTabs();
    saveMetaToServer();
}

// ---- Add tab ----

export function addNotepadTab() {
    tabCounter++;
    const newTab = { id: 'tab-' + tabCounter, name: 'Untitled ' + tabCounter, content: '' };
    notepadTabs.push(newTab);
    switchNotepadTab(newTab.id);
}

// ---- Close tab ----

export function closeNotepadTab(tabId) {
    if (notepadTabs.length <= 1) return;

    const idx = notepadTabs.findIndex(t => t.id === tabId);
    notepadTabs.splice(idx, 1);

    // Delete tab content from server
    apiDelete(tabId);

    if (activeNotepadTab === tabId) {
        activeNotepadTab = notepadTabs[Math.min(idx, notepadTabs.length - 1)].id;
        const textarea = document.getElementById('notepad-content');
        const newTab = notepadTabs.find(t => t.id === activeNotepadTab);
        if (textarea && newTab) {
            textarea.value = newTab.content || '';
        }
    }

    renderNotepadTabs();
    saveMetaToServer();
}

// ---- Save content (called on textarea input) ----

export function saveNotepadContent() {
    const currentTab = notepadTabs.find(t => t.id === activeNotepadTab);
    const textarea = document.getElementById('notepad-content');
    if (currentTab && textarea) {
        currentTab.content = textarea.value;
    }
    saveCurrentTabToServer();
}
