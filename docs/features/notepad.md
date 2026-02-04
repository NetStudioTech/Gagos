# Notepad

GAGOS includes a simple multi-tab text editor for notes and temporary content.

## Features

- **Multi-tab Interface** - Work with multiple documents
- **Content Persistence** - Content saved to browser localStorage
- **Simple Editor** - Plain text editing without distractions

## Usage

### Opening Notepad

Click the Notepad icon on the desktop.

### Working with Tabs

#### New Tab
1. Click the "+" button in the tab bar
2. New untitled tab opens

#### Switch Tabs
Click on any tab to switch to it.

#### Close Tab
Click the "x" on a tab to close it.

#### Rename Tab
Double-click the tab name to rename it.

### Editing Content

1. Click in the text area
2. Type or paste content
3. Content auto-saves to browser storage

### Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| Ctrl+S | Save (manual save) |
| Ctrl+A | Select all |
| Ctrl+C | Copy |
| Ctrl+V | Paste |
| Ctrl+Z | Undo |
| Ctrl+Y | Redo |

## Use Cases

### Temporary Notes

Keep quick notes during troubleshooting:
- Error messages to investigate
- Command outputs to compare
- Configuration snippets

### YAML Staging

Draft Kubernetes YAML before applying:
1. Write YAML in Notepad
2. Copy to Kubernetes Create dialog
3. Apply to cluster

### Clipboard Buffer

Use as intermediate storage:
1. Copy content from one source
2. Paste into Notepad
3. Edit as needed
4. Copy to final destination

### Log Analysis

Paste log snippets for review:
1. Copy relevant log lines
2. Paste into Notepad tab
3. Search and analyze

## Storage

Content is stored in browser localStorage:
- Persists across page reloads
- Specific to browser/device
- Not synced across browsers
- Cleared if browser data is cleared

## Limitations

- **Plain Text Only** - No syntax highlighting
- **Browser Storage** - Limited to localStorage capacity
- **No File System** - Cannot save to/load from files
- **No Collaboration** - Single-user only

## Tips

1. **Use descriptive tab names** - Helps organize multiple notes
2. **Don't store sensitive data** - localStorage is not encrypted
3. **Export important content** - Copy to external storage if needed
4. **Keep it simple** - Best for temporary, short-term content
