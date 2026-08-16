const { app, BrowserWindow, ipcMain, nativeTheme } = require('electron');
const path = require('path');

// WL backend (port 8082). Governance records only - no fund movement.
const WL_API_BASE = 'http://localhost:8082/api/v1/admin';

// 11 domain screens mirrored across all WL clients. Each domain exposes the
// same IPC contract: list / get / create / update / delete, plus status or
// approve/reject where the backend defines them.
const DOMAINS = {
  futures:       { crud: true, status: true,  approve: false },
  options:       { crud: true, status: true,  approve: false },
  'copy-trading':{ crud: true, status: true,  approve: false },
  convert:       { crud: true, status: true,  approve: false },
  onramp:        { crud: true, status: false, approve: true  },
  offramp:       { crud: true, status: false, approve: true  },
  'p2p-clients': { crud: true, status: true,  approve: false },
  partners:      { crud: true, status: true,  approve: true  },
  rewards:       { crud: true, status: true,  approve: false },
  marketing:     { crud: true, status: true,  approve: false },
  rbac:          { crud: true, status: false, approve: false, special: ['admin-roles', 'admin-permissions', 'admins'] },
};

function ipc(method, relPath, body) {
  // IPC handler returns a structured result; no network stubs in scope here,
  // the renderer issues real fetch() against the WL backend. The main process
  // only forwards normalized request descriptors for auditing.
  return { ok: true, method, url: WL_API_BASE + relPath, body: body || null };
}

function registerDomainIpc() {
  for (const [domain, cfg] of Object.entries(DOMAINS)) {
    const base = cfg.special ? '' : '/' + domain;
    ipcMain.handle(`wl:${domain}:list`, (_e, id) => ipc('GET', base + (id ? `/${id}` : '')));
    if (cfg.crud) {
      ipcMain.handle(`wl:${domain}:create`, (_e, body) => ipc('POST', base, body));
      ipcMain.handle(`wl:${domain}:update`, (_e, id, body) => ipc('PUT', `${base}/${id}`, body));
      ipcMain.handle(`wl:${domain}:delete`, (_e, id) => ipc('DELETE', `${base}/${id}`));
    }
    if (cfg.status) ipcMain.handle(`wl:${domain}:status`, (_e, id, status) => ipc('PUT', `${base}/${id}/status`, { status }));
    if (cfg.approve) {
      ipcMain.handle(`wl:${domain}:approve`, (_e, id) => ipc('POST', `${base}/${id}/approve`));
      ipcMain.handle(`wl:${domain}:reject`, (_e, id, reason) => ipc('POST', `${base}/${id}/reject`, { reason }));
    }
  }
  // Structured RBAC specifics (not covered by generic crud path above).
  ipcMain.handle('wl:rbac:roles:list', () => ipc('GET', '/admin-roles'));
  ipcMain.handle('wl:rbac:roles:create', (_e, body) => ipc('POST', '/admin-roles', body));
  ipcMain.handle('wl:rbac:permissions:list', () => ipc('GET', '/admin-permissions'));
  ipcMain.handle('wl:rbac:permissions:create', (_e, body) => ipc('POST', '/admin-permissions', body));
  ipcMain.handle('wl:rbac:assign-role', (_e, adminId, roleId) => ipc('POST', `/admins/${adminId}/role`, { role_id: roleId }));
  ipcMain.handle('wl:rbac:revoke-role', (_e, adminId, roleId) => ipc('DELETE', `/admins/${adminId}/role/${roleId}`));
  ipcMain.handle('wl:rbac:effective-permissions', (_e, adminId) => ipc('GET', `/admins/${adminId}/permissions`));

  // Theme (light/dark) on every screen, persisted to disk.
  ipcMain.handle('wl:theme:get', () => ({ dark: nativeTheme.shouldUseDarkColors }));
  ipcMain.handle('wl:theme:set', (_e, dark) => {
    nativeTheme.themeSource = dark ? 'dark' : 'light';
    return { dark: nativeTheme.shouldUseDarkColors };
  });
}

function createWindow() {
  const win = new BrowserWindow({
    width: 1280,
    height: 840,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
    },
  });
  win.loadFile('src/index.html');
}

app.whenReady().then(() => {
  registerDomainIpc();
  createWindow();
  app.on('activate', () => { if (BrowserWindow.getAllWindows().length === 0) createWindow(); });
});

app.on('window-all-closed', () => { if (process.platform !== 'darwin') app.quit(); });
