// Taskbar Module for GAGOS

export function updateClock() {
    const now = new Date();
    const clockEl = document.getElementById('clock');
    if (clockEl) {
        clockEl.textContent = now.toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'});
    }
}

export async function checkHealth() {
    try {
        const r = await fetch('/api/health');
        const d = await r.json();
        const dot = document.getElementById('health-dot');
        const text = document.getElementById('health-text');
        if (d.status === 'healthy') {
            dot.className = 'status-dot';
            text.textContent = 'Connected';
        } else {
            dot.className = 'status-dot error';
            text.textContent = 'Error';
        }
    } catch {
        document.getElementById('health-dot').className = 'status-dot error';
        document.getElementById('health-text').textContent = 'Offline';
    }
}

export async function logout() {
    try {
        await fetch('/api/auth/logout', { method: 'POST' });
    } catch (e) {
        console.error('Logout error:', e);
    }
    window.location.href = '/login';
}
