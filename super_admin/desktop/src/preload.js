/**
 * TigerWallet Super Admin - Electron Preload
 * Bridges the sandboxed renderer to the main-process IPC handlers that talk
 * to the real super_admin/go backend on :8082.
 */
const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('tiger', {
  // Theme
  getTheme: () => ipcRenderer.invoke('get-theme'),
  setTheme: (dark) => ipcRenderer.invoke('set-theme', dark),
  onThemeChanged: (cb) => ipcRenderer.on('theme-changed', (_e, dark) => cb(dark)),
  onNavigate: (cb) => ipcRenderer.on('navigate', (_e, route) => cb(route)),

  // Auth token (JWT) for the admin backend
  setToken: (token) => ipcRenderer.invoke('admin:set-token', token),

  // Domain metadata + generic CRUD
  domains: () => ipcRenderer.invoke('admin:domains'),
  domainCall: (args) => ipcRenderer.invoke('admin:domain-call', args),
  domainAction: (args) => ipcRenderer.invoke('admin:domain-action', args),

  // Misc
  getAppInfo: () => ipcRenderer.invoke('get-app-info')
});
