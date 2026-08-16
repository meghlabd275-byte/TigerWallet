/**
 * TigerWallet Super Admin - Desktop Application
 * Electron-based cross-platform desktop app
 */

const { app, BrowserWindow, ipcMain, Menu, Tray, nativeTheme, dialog } = require('electron');
const path = require('path');

// Real super_admin/go backend on port 8082 (JWT bearer auth).
const ADMIN_API_BASE = 'http://localhost:8082/api/v1/admin';
// Token is held in-memory after the admin signs in via the renderer.
let adminToken = '';

// The 12 governance domains and the governance actions each supports.
// `resource` is the path segment under /api/v1/admin.
const ADMIN_DOMAINS = [
  { id: 'futures', label: 'Futures', resource: 'futures', actions: ['status'] },
  { id: 'options', label: 'Options', resource: 'options', actions: ['status'] },
  { id: 'copy-trading', label: 'Copy Trading', resource: 'copy-trading', actions: ['status'] },
  { id: 'convert', label: 'Convert', resource: 'convert', actions: ['status'] },
  { id: 'onramp', label: 'Onramp', resource: 'onramp', actions: ['approve', 'reject'] },
  { id: 'offramp', label: 'Offramp', resource: 'offramp', actions: ['approve', 'reject'] },
  { id: 'p2p-clients', label: 'P2P Clients', resource: 'p2p-clients', actions: ['status'] },
  { id: 'partners', label: 'Partners', resource: 'partners', actions: ['status', 'approve', 'reject'] },
  { id: 'rewards', label: 'Rewards', resource: 'rewards', actions: ['status'] },
  { id: 'marketing', label: 'Marketing', resource: 'marketing', actions: ['status'] },
  { id: 'admin-roles', label: 'Admin Roles', resource: 'admin-roles', actions: [] },
  { id: 'wl-control', label: 'WL Control', resource: 'wl-clients', actions: ['status'] }
];

// Main window
let mainWindow = null;
let tray = null;

// Theme state
let isDarkMode = false;

// Create main window
function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1400,
    height: 900,
    minWidth: 1024,
    minHeight: 700,
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
      preload: path.join(__dirname, 'preload.js')
    },
    show: false,
    backgroundColor: isDarkMode ? '#0f172a' : '#ffffff'
  });

  // Load the app
  mainWindow.loadFile(path.join(__dirname, 'index.html'));

  // Show when ready
  mainWindow.once('ready-to-show', () => {
    mainWindow.show();
  });

  // Create application menu
  createMenu();

  // Handle window close
  mainWindow.on('close', (event) => {
    if (!app.isQuitting) {
      event.preventDefault();
      mainWindow.hide();
    }
  });
}

// Create application menu
function createMenu() {
  const template = [
    {
      label: 'File',
      submenu: [
        { label: 'New Window', accelerator: 'CmdOrCtrl+N', click: () => createWindow() },
        { type: 'separator' },
        { label: 'Export Data', accelerator: 'CmdOrCtrl+E', click: () => exportData() },
        { type: 'separator' },
        { role: 'quit' }
      ]
    },
    {
      label: 'Edit',
      submenu: [
        { role: 'undo' },
        { role: 'redo' },
        { type: 'separator' },
        { role: 'cut' },
        { role: 'copy' },
        { role: 'paste' },
        { role: 'selectall' }
      ]
    },
    {
      label: 'View',
      submenu: [
        { role: 'reload' },
        { role: 'forceReload' },
        { role: 'toggleDevTools' },
        { type: 'separator' },
        { role: 'resetZoom' },
        { role: 'zoomin' },
        { role: 'zoomout' },
        { type: 'separator' },
        { role: 'togglefullscreen' }
      ]
    },
    {
      label: 'Window',
      submenu: [
        { role: 'minimize' },
        { role: 'zoom' },
        { type: 'separator' },
        { role: 'close' }
      ]
    },
    {
      label: 'Help',
      submenu: [
        { label: 'Documentation', click: () => openDocumentation() },
        { label: 'About', click: () => showAbout() }
      ]
    }
  ];

  // Add theme toggle to View menu
  template[2].submenu.push(
    { type: 'separator' },
    {
      label: isDarkMode ? 'Light Mode' : 'Dark Mode',
      accelerator: 'CmdOrCtrl+Shift+T',
      click: () => toggleTheme()
    }
  );

  const menu = Menu.buildFromTemplate(template);
  Menu.setApplicationMenu(menu);
}

// Create system tray
function createTray() {
  // Use a simple icon path - in production, use actual icon
  const iconPath = path.join(__dirname, 'assets', 'icon.png');
  
  try {
    tray = new Tray(iconPath);
  } catch (e) {
    console.log('Tray icon not found, skipping tray');
    return;
  }

  const contextMenu = Menu.buildFromTemplate([
    { label: 'Show App', click: () => mainWindow.show() },
    { type: 'separator' },
    { label: 'Dashboard', click: () => navigateTo('dashboard') },
    { label: 'Users', click: () => navigateTo('users') },
    { label: 'Transactions', click: () => navigateTo('transactions') },
    { type: 'separator' },
    { label: 'Toggle Theme', click: () => toggleTheme() },
    { type: 'separator' },
    { label: 'Quit', click: () => { app.isQuitting = true; app.quit(); } }
  ]);

  tray.setToolTip('TigerWallet Super Admin');
  tray.setContextMenu(contextMenu);

  tray.on('double-click', () => {
    mainWindow.show();
  });
}

// Navigate to a route
function navigateTo(route) {
  mainWindow.webContents.send('navigate', route);
}

// Toggle theme
function toggleTheme() {
  isDarkMode = !isDarkMode;
  nativeTheme.themeSource = isDarkMode ? 'dark' : 'light';
  mainWindow.webContents.send('theme-changed', isDarkMode);
  
  // Update menu
  createMenu();
}

// Export data
async function exportData() {
  const { filePath } = await dialog.showSaveDialog(mainWindow, {
    title: 'Export Data',
    defaultPath: 'export.csv',
    filters: [
      { name: 'CSV Files', extensions: ['csv'] },
      { name: 'JSON Files', extensions: ['json'] }
    ]
  });
  
  if (filePath) {
    mainWindow.webContents.send('export-data', filePath);
  }
}

// Open documentation
function openDocumentation() {
  require('electron').shell.openExternal('https://docs.tigerwallet.com');
}

// Show about dialog
function showAbout() {
  dialog.showMessageBox(mainWindow, {
    type: 'info',
    title: 'About TigerWallet Super Admin',
    message: 'TigerWallet Super Admin',
    detail: 'Version 1.0.0\n\nA comprehensive admin management system for the TigerWallet platform.'
  });
}

// IPC handlers
ipcMain.handle('get-theme', () => isDarkMode);
ipcMain.handle('set-theme', (event, darkMode) => {
  isDarkMode = darkMode;
  nativeTheme.themeSource = isDarkMode ? 'dark' : 'light';
  return isDarkMode;
});

ipcMain.handle('show-notification', (event, { title, body }) => {
  const notification = new (require('electron').Notification)({
    title,
    body
  });
  notification.show();
});

ipcMain.handle('get-app-info', () => ({
  name: app.getName(),
  version: app.getVersion(),
  platform: process.platform
}));

// --- Super-admin domain IPC handlers (real HTTP to :8082) -----------------

ipcMain.handle('admin:set-token', (_event, token) => {
  adminToken = typeof token === 'string' ? token : '';
  return { ok: true };
});

ipcMain.handle('admin:domains', () => ADMIN_DOMAINS);

// Generic CRUD: admin:domain-call { domain, op, id?, body? }
// op in: list | get | create | update | delete | status | approve | reject
ipcMain.handle('admin:domain-call', async (_event, args) => {
  return domainCall(args || {});
});

// Apply a governance sub-action (status/approve/reject) with its payload.
ipcMain.handle('admin:domain-action', async (_event, args) => {
  const { domain, action, id, reason, status } = args || {};
  const cfg = ADMIN_DOMAINS.find((d) => d.id === domain);
  if (!cfg) return { error: 'Unknown domain: ' + domain };
  if (!cfg.actions.includes(action)) {
    return { error: 'Action "' + action + '" not supported by ' + cfg.label };
  }
  if (action === 'status') {
    return domainCall({ domain, op: 'status', id, body: { status } });
  }
  if (action === 'approve') {
    return domainCall({ domain, op: 'approve', id, body: {} });
  }
  if (action === 'reject') {
    return domainCall({ domain, op: 'reject', id, body: { reason } });
  }
  return { error: 'Unsupported action: ' + action };
});

// Resolve a domain id -> resource path segment.
function resolveResource(domain) {
  const cfg = ADMIN_DOMAINS.find((d) => d.id === domain);
  return cfg ? cfg.resource : domain;
}

// Build the URL for a given op.
function buildUrl(domain, op, id) {
  const resource = resolveResource(domain);
  switch (op) {
    case 'list': return ADMIN_API_BASE + '/' + resource;
    case 'create': return ADMIN_API_BASE + '/' + resource;
    case 'get':
    case 'update':
    case 'delete': return ADMIN_API_BASE + '/' + resource + '/' + encodeURIComponent(id);
    case 'status': return ADMIN_API_BASE + '/' + resource + '/' + encodeURIComponent(id) + '/status';
    case 'approve': return ADMIN_API_BASE + '/' + resource + '/' + encodeURIComponent(id) + '/approve';
    case 'reject': return ADMIN_API_BASE + '/' + resource + '/' + encodeURIComponent(id) + '/reject';
    default: return ADMIN_API_BASE + '/' + resource;
  }
}

function methodFor(op) {
  switch (op) {
    case 'list':
    case 'get': return 'GET';
    case 'create':
    case 'approve':
    case 'reject': return 'POST';
    case 'update':
    case 'status': return 'PUT';
    case 'delete': return 'DELETE';
    default: return 'GET';
  }
}

// Real fetch against the Go backend. Returns { ok, status, data } or { error }.
async function domainCall({ domain, op, id, body }) {
  if (!domain) return { error: 'Missing domain' };
  if (op !== 'list' && op !== 'create' && !id) {
    return { error: 'Missing record id for op "' + op + '"' };
  }
  const url = buildUrl(domain, op, id);
  const method = methodFor(op);
  const headers = { 'Content-Type': 'application/json' };
  if (adminToken) headers['Authorization'] = 'Bearer ' + adminToken;
  const init = { method, headers };
  if (method !== 'GET' && method !== 'DELETE') {
    init.body = JSON.stringify(body || {});
  }
  try {
    const res = await fetch(url, init);
    const text = await res.text();
    let parsed = null;
    try { parsed = text ? JSON.parse(text) : null; }
    catch (_) { parsed = { raw: text }; }
    if (!res.ok) {
      const msg = (parsed && parsed.error) ? parsed.error : ('HTTP ' + res.status);
      return { error: msg, status: res.status };
    }
    return { ok: true, status: res.status, data: parsed };
  } catch (error) {
    return { error: (error && error.message) ? error.message : 'Failed to reach super-admin backend.' };
  }
}

// App ready
app.whenReady().then(() => {
  createWindow();
  createTray();
  
  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});

// Quit when all windows closed (except on macOS)
app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

// Before quit
app.on('before-quit', () => {
  app.isQuitting = true;
});
