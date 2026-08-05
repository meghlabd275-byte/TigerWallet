/**
 * TigerWallet Super Admin - Desktop Application
 * Electron-based cross-platform desktop app
 */

const { app, BrowserWindow, ipcMain, Menu, Tray, nativeTheme, dialog } = require('electron');
const path = require('path');

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
