// Terminal Module for GAGOS
/* global Terminal, FitAddon */

let term = null;
let termWs = null;
let fitAddon = null;
let termInitialized = false;

export function initTerminal() {
    if (termInitialized) return;

    const container = document.getElementById('terminal-container');
    if (!container) {
        console.error('Terminal container not found');
        return;
    }

    // Check if xterm libraries are loaded
    if (typeof Terminal === 'undefined') {
        container.innerHTML = '<div style="color:#ef4444;padding:20px;">Error: xterm.js library failed to load. Check your internet connection.</div>';
        return;
    }
    if (typeof FitAddon === 'undefined') {
        container.innerHTML = '<div style="color:#ef4444;padding:20px;">Error: xterm-addon-fit library failed to load.</div>';
        return;
    }

    // Clear loading message
    container.innerHTML = '';

    try {
        term = new Terminal({
            cursorBlink: true,
            fontSize: 14,
            fontFamily: 'Menlo, Monaco, "Courier New", monospace',
            theme: {
                background: '#0d0d14',
                foreground: '#e4e4e7',
                cursor: '#4ade80',
                cursorAccent: '#0d0d14',
                black: '#1a1a2e',
                red: '#f87171',
                green: '#4ade80',
                yellow: '#fbbf24',
                blue: '#667eea',
                magenta: '#a78bfa',
                cyan: '#22d3ee',
                white: '#e4e4e7',
                brightBlack: '#6a6a7a',
                brightRed: '#fca5a5',
                brightGreen: '#86efac',
                brightYellow: '#fde047',
                brightBlue: '#818cf8',
                brightMagenta: '#c4b5fd',
                brightCyan: '#67e8f9',
                brightWhite: '#ffffff'
            }
        });

        fitAddon = new FitAddon.FitAddon();
        term.loadAddon(fitAddon);

        term.open(container);

        // Delay fit to ensure container has dimensions
        setTimeout(() => {
            if (fitAddon) fitAddon.fit();
        }, 100);

        connectTerminalWs();
        termInitialized = true;

        // Handle input
        term.onData(data => {
            if (termWs && termWs.readyState === WebSocket.OPEN) {
                termWs.send(JSON.stringify({ type: 'input', data: data }));
            }
        });

        // Handle resize
        const resizeObserver = new ResizeObserver(() => {
            if (fitAddon && term) {
                fitAddon.fit();
                if (termWs && termWs.readyState === WebSocket.OPEN) {
                    termWs.send(JSON.stringify({
                        type: 'resize',
                        cols: term.cols,
                        rows: term.rows
                    }));
                }
            }
        });
        resizeObserver.observe(container);

    } catch (e) {
        console.error('Terminal initialization error:', e);
        container.innerHTML = `<div style="color:#ef4444;padding:20px;">Error initializing terminal: ${e.message}</div>`;
    }
}

function connectTerminalWs() {
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${wsProtocol}//${window.location.host}/api/v1/terminal/ws`;

    termWs = new WebSocket(wsUrl);

    termWs.onopen = () => {
        // Send initial size
        termWs.send(JSON.stringify({
            type: 'resize',
            cols: term.cols,
            rows: term.rows
        }));
    };

    termWs.onmessage = (event) => {
        const msg = JSON.parse(event.data);
        if (msg.type === 'output' && msg.data) {
            term.write(msg.data);
        }
    };

    termWs.onerror = () => {
        term.writeln('\x1b[1;31mConnection error\x1b[0m');
    };

    termWs.onclose = () => {
        term.writeln('');
        term.writeln('\x1b[1;33mConnection closed. Press Enter to reconnect...\x1b[0m');
    };
}

export function reconnectTerminal() {
    if (termWs) {
        termWs.close();
    }
    term.clear();
    connectTerminalWs();
}
