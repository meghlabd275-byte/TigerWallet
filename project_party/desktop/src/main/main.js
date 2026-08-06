const { app, BrowserWindow, ipcMain, Menu, nativeTheme } = require('electron');
const path = require('path');

let mainWindow;

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1200,
    height: 800,
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
      preload: path.join(__dirname, 'preload.js')
    },
    title: 'UserWallet',
    backgroundColor: nativeTheme.shouldUseDarkColors ? '#1a1a1a' : '#ffffff'
  });

  mainWindow.loadFile('dist/index.html');

  // Create menu
  const menuTemplate = [
    {
      label: 'File',
      submenu: [
        { role: 'quit' }
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
        { role: 'zoomIn' },
        { role: 'zoomOut' },
        { type: 'separator' },
        { role: 'togglefullscreen' }
      ]
    },
    {
      label: 'Window',
      submenu: [
        { role: 'minimize' },
        { role: 'close' }
      ]
    },
    {
      label: 'Theme',
      submenu: [
        {
          label: 'Light',
          type: 'radio',
          checked: !nativeTheme.shouldUseDarkColors,
          click: () => {
            nativeTheme.themeSource = 'light';
            mainWindow.webContents.send('theme-changed', 'light');
          }
        },
        {
          label: 'Dark',
          type: 'radio',
          checked: nativeTheme.shouldUseDarkColors,
          click: () => {
            nativeTheme.themeSource = 'dark';
            mainWindow.webContents.send('theme-changed', 'dark');
          }
        },
        {
          label: 'System',
          type: 'radio',
          click: () => {
            nativeTheme.themeSource = 'system';
            mainWindow.webContents.send('theme-changed', 'system');
          }
        }
      ]
    }
  ];

  const menu = Menu.buildFromTemplate(menuTemplate);
  Menu.setApplicationMenu(menu);
}

app.whenReady().then(() => {
  createWindow();

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

// IPC handlers
ipcMain.handle('get-theme', () => {
  return nativeTheme.shouldUseDarkColors ? 'dark' : 'light';
});

ipcMain.handle('set-theme', (event, theme) => {
  nativeTheme.themeSource = theme;
  return theme;
});
