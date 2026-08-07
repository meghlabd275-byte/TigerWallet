/**
 * TigerWallet Admin - Preload Script
 * Secure bridge between main and renderer processes
 */

const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('electronAPI', {
  getTheme: () => ipcRenderer.invoke('get-theme'),
  setTheme: (theme) => ipcRenderer.invoke('set-theme', theme),
  getAppVersion: () => ipcRenderer.invoke('get-app-version'),
  onOpenSettings: (callback) => ipcRenderer.on('open-settings', callback)
});
