// Preload: exposes the WL admin domain IPC contract to the renderer under
// contextIsolation. Each domain screen uses window.wlAdmin.<domain>.* .
const { contextBridge, ipcRenderer } = require('electron');

const DOMAINS = [
  'futures', 'options', 'copy-trading', 'convert', 'onramp', 'offramp',
  'p2p-clients', 'partners', 'rewards', 'marketing', 'rbac',
  // 9 scoped admin domains (real main.go routes on port 8082)
  'liquidity', 'crypto-card', 'bots', 'kyc', 'tickets',
  'ip-whitelist', 'audit-logs', 'wallet-management', 'withdrawals',
];

const api = { theme: {
  get: () => ipcRenderer.invoke('wl:theme:get'),
  set: (dark) => ipcRenderer.invoke('wl:theme:set', dark),
}};

for (const d of DOMAINS) {
  api[d] = {
    list: (id) => ipcRenderer.invoke(`wl:${d}:list`, id),
    create: (body) => ipcRenderer.invoke(`wl:${d}:create`, body),
    update: (id, body) => ipcRenderer.invoke(`wl:${d}:update`, id, body),
    remove: (id) => ipcRenderer.invoke(`wl:${d}:delete`, id),
    status: (id, s) => ipcRenderer.invoke(`wl:${d}:status`, id, s),
    approve: (id) => ipcRenderer.invoke(`wl:${d}:approve`, id),
    reject: (id, reason) => ipcRenderer.invoke(`wl:${d}:reject`, id, reason),
    // Non-uniform real backend routes (allocations/stats/transactions/...).
    extra: (i, id, body) => ipcRenderer.invoke(`wl:${d}:extra:${i}`, id, body),
  };
}

// Structured RBAC specifics beyond the generic CRUD contract.
api.rbac.roles = {
  list: () => ipcRenderer.invoke('wl:rbac:roles:list'),
  create: (body) => ipcRenderer.invoke('wl:rbac:roles:create', body),
};
api.rbac.permissions = {
  list: () => ipcRenderer.invoke('wl:rbac:permissions:list'),
  create: (body) => ipcRenderer.invoke('wl:rbac:permissions:create', body),
};
api.rbac.assignRole = (adminId, roleId) => ipcRenderer.invoke('wl:rbac:assign-role', adminId, roleId);
api.rbac.revokeRole = (adminId, roleId) => ipcRenderer.invoke('wl:rbac:revoke-role', adminId, roleId);
api.rbac.effectivePermissions = (adminId) => ipcRenderer.invoke('wl:rbac:effective-permissions', adminId);

contextBridge.exposeInMainWorld('wlAdmin', api);
