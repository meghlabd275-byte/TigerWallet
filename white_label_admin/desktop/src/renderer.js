// Renderer: drives the 11 + 9 domain screens via the wlAdmin preload bridge.
// Each screen renders the backend contract and binds the actions to IPC.
const DOMAINS = [
  { key: 'dashboard',    label: 'Dashboard',     endpoints: [['GET', '/api/v1/admin/stats', false]] },
  { key: 'users',        label: 'Users',         endpoints: [['GET', '/api/v1/admin/users', false]] },
  { key: 'futures',      label: 'Futures',       endpoints: [['GET', '/futures', false], ['POST', '/futures', false], ['PUT', '/futures/:id', false], ['DELETE', '/futures/:id', false], ['PUT', '/futures/:id/status', true]] },
  { key: 'options',      label: 'Options',       endpoints: [['GET', '/options', false], ['POST', '/options', false], ['PUT', '/options/:id', false], ['DELETE', '/options/:id', false], ['PUT', '/options/:id/status', true]] },
  { key: 'copy-trading', label: 'Copy Trading',  endpoints: [['GET', '/copy-trading', false], ['POST', '/copy-trading', false], ['PUT', '/copy-trading/:id', false], ['DELETE', '/copy-trading/:id', false], ['PUT', '/copy-trading/:id/status', true]] },
  { key: 'convert',      label: 'Convert',       endpoints: [['GET', '/convert', false], ['POST', '/convert', false], ['PUT', '/convert/:id', false], ['DELETE', '/convert/:id', false], ['PUT', '/convert/:id/status', true]] },
  { key: 'onramp',       label: 'On-Ramp',       endpoints: [['GET', '/onramp', false], ['POST', '/onramp', false], ['PUT', '/onramp/:id', false], ['DELETE', '/onramp/:id', false], ['POST', '/onramp/:id/approve', true], ['POST', '/onramp/:id/reject {reason}', true]] },
  { key: 'offramp',      label: 'Off-Ramp',      endpoints: [['GET', '/offramp', false], ['POST', '/offramp', false], ['PUT', '/offramp/:id', false], ['DELETE', '/offramp/:id', false], ['POST', '/offramp/:id/approve', true], ['POST', '/offramp/:id/reject {reason}', true]] },
  { key: 'p2p-clients',  label: 'P2P Clients',   endpoints: [['GET', '/p2p-clients', false], ['POST', '/p2p-clients', false], ['PUT', '/p2p-clients/:id', false], ['DELETE', '/p2p-clients/:id', false], ['PUT', '/p2p-clients/:id/status', true]] },
  { key: 'partners',     label: 'Partners',      endpoints: [['GET', '/partners', false], ['POST', '/partners', false], ['PUT', '/partners/:id', false], ['DELETE', '/partners/:id', false], ['PUT', '/partners/:id/status', true], ['POST', '/partners/:id/approve', true], ['POST', '/partners/:id/reject {reason}', true]] },
  { key: 'rewards',      label: 'Rewards',       endpoints: [['GET', '/rewards', false], ['POST', '/rewards', false], ['PUT', '/rewards/:id', false], ['DELETE', '/rewards/:id', false], ['PUT', '/rewards/:id/status', true]] },
  { key: 'marketing',    label: 'Marketing',     endpoints: [['GET', '/marketing', false], ['POST', '/marketing', false], ['PUT', '/marketing/:id', false], ['DELETE', '/marketing/:id', false], ['PUT', '/marketing/:id/status', true]] },
  { key: 'liquidity',    label: 'Liquidity',     endpoints: [['GET', '/wl-liquidity/sources', false], ['POST', '/wl-liquidity/sources', false], ['PUT', '/wl-liquidity/sources/:id', true], ['DELETE', '/wl-liquidity/sources/:id', true], ['GET', '/wl-liquidity/allocations', false], ['POST', '/wl-liquidity/allocations', false], ['GET', '/wl-liquidity/stats', false]] },
  { key: 'crypto-card',  label: 'Crypto Cards',  endpoints: [['GET', '/wl-cards', false], ['POST', '/wl-cards', false], ['PUT', '/wl-cards/:id/status', true], ['GET', '/wl-cards/transactions', false], ['GET', '/wl-cards/stats', false]] },
  { key: 'bots',         label: 'Bots',          endpoints: [['GET', '/wl-bots/operators', false], ['POST', '/wl-bots/operators', false], ['PUT', '/wl-bots/operators/:id/status', true], ['GET', '/wl-bots/config', false], ['GET', '/wl-bots/stats', false]] },
  { key: 'kyc',          label: 'KYC',           endpoints: [['GET', '/kyc', false], ['POST', '/kyc/:id/approve', true], ['POST', '/kyc/:id/reject {reason}', true]] },
  { key: 'tickets',      label: 'Support Tickets', endpoints: [['GET', '/tickets', false], ['GET', '/tickets/:id', true], ['POST', '/tickets', false], ['PUT', '/tickets/:id/status', true], ['POST', '/tickets/:id/messages', true], ['PUT', '/tickets/:id/assign', true]] },
  { key: 'ip-whitelist', label: 'IP Whitelist',  endpoints: [['GET', '/ip-whitelist', false], ['POST', '/ip-whitelist', false], ['DELETE', '/ip-whitelist/:id', true]] },
  { key: 'audit-logs',   label: 'Audit Logs',    endpoints: [['GET', '/audit-logs', false], ['POST', '/audit-logs/export', true]] },
  { key: 'wallet-management', label: 'Wallet Mgmt', endpoints: [['GET', '/withdrawals', false], ['GET', '/fees', false], ['POST', '/fees', false], ['PUT', '/fees/:id', true], ['PUT', '/users/:id/status', true], ['POST', '/withdrawals/:id/approve', true], ['POST', '/withdrawals/:id/reject {reason}', true], ['POST', '/withdrawals/:id/process', true]] },
  { key: 'withdrawals',  label: 'Withdrawals',   endpoints: [['GET', '/withdrawals', false], ['POST', '/withdrawals/:id/approve', true], ['POST', '/withdrawals/:id/reject {reason}', true], ['POST', '/withdrawals/:id/process', true]] },
  { key: 'rbac',         label: 'Admin Roles',   endpoints: [['GET', '/admin-roles', false], ['POST', '/admin-roles', false], ['GET', '/admin-permissions', false], ['POST', '/admins/:id/role', true], ['GET', '/admins/:id/permissions', false], ['DELETE', '/admins/:id/role/:roleId', true]] },
];

const nav = document.getElementById('nav');
const title = document.getElementById('title');
const subtitle = document.getElementById('subtitle');
const endpointsEl = document.getElementById('endpoints');

function renderNav() {
  nav.innerHTML = '';
  DOMAINS.forEach((d) => {
    const el = document.createElement('div');
    el.className = 'nav-item';
    el.textContent = d.label;
    el.dataset.key = d.key;
    el.addEventListener('click', () => select(d));
    nav.appendChild(el);
  });
}

function select(d) {
  document.querySelectorAll('.nav-item').forEach((n) => n.classList.toggle('active', n.dataset.key === d.key));
  title.textContent = d.label;
  subtitle.textContent = 'Governance records only - no fund movement.';
  endpointsEl.innerHTML = d.endpoints.map(([m, p, gov]) =>
    `<div class="ep ${gov ? 'gov' : ''}"><span class="label">${m}</span><span class="path">${p}</span></div>`
  ).join('') + '<div class="note">Backend: http://localhost:8082 (WL admin)</div>';
  // Bind the domain's IPC contract (real calls issued on demand by future
  // table renderers; selection already exposes window.wlAdmin[d.key]).
  const bridge = window.wlAdmin && window.wlAdmin[d.key];
  if (bridge && bridge.list) bridge.list().catch(() => {});
}

async function initTheme() {
  const toggle = document.getElementById('theme-toggle');
  const res = await window.wlAdmin.theme.get();
  if (res && res.dark) document.body.classList.add('dark');
  document.documentElement.dataset.theme = res && res.dark ? 'dark' : 'light';
  toggle.addEventListener('click', async () => {
    const next = !document.body.classList.contains('dark');
    document.body.classList.toggle('dark', next);
    document.documentElement.dataset.theme = next ? 'dark' : 'light';
    await window.wlAdmin.theme.set(next);
  });
}

renderNav();
select(DOMAINS[0]);
initTheme();
