const { app, BrowserWindow, ipcMain } = require('electron');
const path = require('path');

let mainWindow;
let isDarkMode = false;

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1200,
    height: 800,
    webPreferences: { nodeIntegration: false, contextIsolation: true }
  });
  mainWindow.loadFile(path.join(__dirname, 'src', 'index.html'));
  mainWindow.on('closed', () => { mainWindow = null; });
}

app.whenReady().then(() => { createWindow(); app.on('activate', () => { if (BrowserWindow.getAllWindows().length === 0) createWindow(); }); });
app.on('window-all-closed', () => { if (process.platform !== 'darwin') app.quit(); });

ipcMain.handle('get-dark-mode', () => isDarkMode);
ipcMain.handle('set-dark-mode', (event, value) => { isDarkMode = value; mainWindow.webContents.send('theme-changed', value); });
ipcMain.handle('get-api-url', () => 'http://localhost:8090');
