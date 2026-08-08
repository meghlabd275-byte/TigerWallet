/**
 * TigerWallet Admin - Desktop Main Process
 * Complete Dark/Light Theme Support
 */

const { app, BrowserWindow, ipcMain, Menu, Tray, nativeTheme } = require('electron');
const path = require('path');
const log = require('electron-log');

log.transports.file.level = 'info';
log.transports.console.level = 'debug';

let mainWindow = null;
let tray = null;

process.on('uncaughtException', (error) => {
  log.error('Uncaught Exception:', error);
  app.exit(1);
});

process.on('unhandledRejection', (error) => {
  log.error('Unhandled Rejection:', error);
});

function createWindow() {
  log.info('Creating main window...');
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
    backgroundColor: nativeTheme.shouldUseDarkColors ? '#0A0A0F' : '#F8F9FA'
  });

  mainWindow.loadFile(path.join(__dirname, '../public/index.html'));
  mainWindow.once('ready-to-show', () => {
    mainWindow.show();
    log.info('Main window displayed');
  });

  mainWindow.on('closed', () => {
    mainWindow = null;
  });

  createMenu();
}

function createMenu() {
  const template = [
    { label: 'TigerAdmin', submenu: [
      { label: 'About TigerAdmin', role: 'about' },
      { type: 'separator' },
      { label: 'Quit', accelerator: 'CmdOrCtrl+Q', role: 'quit' }
    ]},
    { label: 'Edit', submenu: [
      { label: 'Undo', accelerator: 'CmdOrCtrl+Z', role: 'undo' },
      { label: 'Redo', accelerator: 'Shift+CmdOrCtrl+Z', role: 'redo' },
      { type: 'separator' },
      { label: 'Cut', accelerator: 'CmdOrCtrl+X', role: 'cut' },
      { label: 'Copy', accelerator: 'CmdOrCtrl+C', role: 'copy' },
      { label: 'Paste', accelerator: 'CmdOrCtrl+V', role: 'paste' }
    ]},
    { label: 'View', submenu: [
      { label: 'Reload', accelerator: 'CmdOrCtrl+R', role: 'reload' },
      { type: 'separator' },
      { label: 'Toggle Full Screen', accelerator: 'F11', role: 'togglefullscreen' },
      { label: 'Toggle Developer Tools', accelerator: 'F12', role: 'toggleDevTools' }
    ]},
    { label: 'Window', submenu: [
      { label: 'Minimize', accelerator: 'CmdOrCtrl+M', role: 'minimize' },
      { label: 'Close', accelerator: 'CmdOrCtrl+W', role: 'close' }
    ]}
  ];

  const menu = Menu.buildFromTemplate(template);
  Menu.setApplicationMenu(menu);
}

ipcMain.handle('get-theme', () => {
  return nativeTheme.shouldUseDarkColors ? 'dark' : 'light';
});

ipcMain.handle('set-theme', (event, theme) => {
  nativeTheme.themeSource = theme;
  return true;
});

ipcMain.handle('get-app-version', () => {
  return app.getVersion();
});

app.whenReady().then(() => {
  log.info('App ready, starting TigerAdmin...');
  createWindow();
  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow();
  });
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') app.quit();
});
