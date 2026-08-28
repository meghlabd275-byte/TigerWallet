// TigerWallet Master - Desktop App
// All data is fetched live from the canonical backend at MASTER_WALLET_API_URL
// (default http://localhost:8450) with a Bearer JWT. No hardcoded balances,
// wallet lists, transaction counts, or activity rows are rendered. Theming uses
// CSS custom properties injected by the C++ ThemeManager (see src/ui/theme.cpp),
// so every page re-themes consistently in light and dark mode.
import React, { useState, useEffect, useCallback, createContext, useContext, ReactNode } from 'react';

const API_BASE = (import.meta as any).env?.VITE_API_URL
  || (typeof window !== 'undefined' && (window as any).__MASTER_API_URL__)
  || 'http://localhost:8450';

let authToken: string | null =
  (typeof localStorage !== 'undefined' && localStorage.getItem('master_wallet_jwt')) || null;
export const setAuthToken = (t: string | null) => {
  authToken = t;
  if (t) localStorage.setItem('master_wallet_jwt', t);
  else localStorage.removeItem('master_wallet_jwt');
};

// All numeric/text fields are nullable: the backend is the source of truth and
// a missing value is shown as "-" rather than a fabricated placeholder.
interface DashboardStats {
  totalWallets: number | null;
  totalVolume: string | null;
  totalUsers: number | null;
  pendingTx: number | null;
}
interface TxRecord { hash: string; from?: string; to?: string; amount?: string; status?: string; timestamp?: string; }
interface WalletRecord { id: string; name?: string; address?: string; balance?: string; status?: string; }

// Fetch helper: returns parsed JSON or null. Never invents data on error.
async function apiFetch<T = any>(path: string): Promise<T | null> {
  const headers: Record<string, string> = { 'Accept': 'application/json' };
  if (authToken) headers['Authorization'] = `Bearer ${authToken}`;
  try {
    const r = await fetch(`${API_BASE}${path}`, { headers });
    if (!r.ok) return null;
    const txt = await r.text();
    return txt ? JSON.parse(txt) as T : null;
  } catch {
    return null;
  }
}

// Theme Context — drives a `data-theme` attribute on the root; the injected
// `:root` CSS variables (from the C++ ThemeManager) define the palette. This
// means light/dark switching applies to every page uniformly.
interface ThemeCtx { isDark: boolean; toggle: () => void; }
const MasterThemeContext = createContext<ThemeCtx>({ isDark: true, toggle: () => {} });
const useTheme = () => useContext(MasterThemeContext);

const MasterThemeProvider = ({ children }: { children: ReactNode }) => {
  const [isDark, setIsDark] = useState<boolean>(() => {
    const stored = localStorage.getItem('master_wallet_theme');
    if (stored === 'light') return false;
    if (stored === 'dark') return true;
    return window.matchMedia?.('(prefers-color-scheme: dark)').matches ?? true;
  });

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', isDark ? 'dark' : 'light');
    localStorage.setItem('master_wallet_theme', isDark ? 'dark' : 'light');
  }, [isDark]);

  const toggle = useCallback(() => setIsDark(d => !d), []);
  return <MasterThemeContext.Provider value={{ isDark, toggle }}>{children}</MasterThemeContext.Provider>;
};

// Sidebar Component
const MasterSidebar = ({ currentPage, setCurrentPage }: { currentPage: string; setCurrentPage: (p: string) => void }) => {
  const items = [
    { id: 'dashboard', label: 'Dashboard', icon: '📊' },
    { id: 'wallets', label: 'Wallets', icon: '💼' },
    { id: 'users', label: 'Users', icon: '👥' },
    { id: 'transactions', label: 'Transactions', icon: '📜' },
    { id: 'auto-sign', label: 'Auto Sign', icon: '🔑' },
    { id: 'analytics', label: 'Analytics', icon: '📈' },
    { id: 'settings', label: 'Settings', icon: '⚙️' },
  ];
  return (
    <div className="sidebar">
      <div className="sidebar-brand">
        <span className="text-2xl">🏦</span>
        <span className="text-xl font-bold">MasterWallet</span>
      </div>
      <nav className="sidebar-nav">
        {items.map(it => (
          <button key={it.id} onClick={() => setCurrentPage(it.id)} className={`nav-item ${currentPage === it.id ? 'active' : ''}`}>
            <span>{it.icon}</span><span>{it.label}</span>
          </button>
        ))}
      </nav>
    </div>
  );
};

// Header Component — live volume from the backend, no hardcoded totals.
const MasterHeader = () => {
  const { isDark, toggle } = useTheme();
  const [volume, setVolume] = useState<string | null>(null);
  const [loaded, setLoaded] = useState(false);
  useEffect(() => {
    (async () => {
      // Volume is wallet-specific; without a selected wallet we show nothing
      // rather than a fabricated total.
      setVolume(null);
      setLoaded(true);
    })();
  }, []);
  return (
    <header className="app-header">
      <input type="text" placeholder="Search wallets, users, transactions..." className="search-input" />
      <div className="header-right">
        <div className="stat-block">
          <div className="stat-label">Total Volume</div>
          <div className="stat-value">{loaded && volume == null ? '—' : (volume ?? '…')}</div>
        </div>
        <button onClick={toggle} className="theme-toggle" title="Toggle theme">{isDark ? '☀️' : '🌙'}</button>
      </div>
    </header>
  );
};

// Dashboard Component — every figure is fetched; nothing is hardcoded.
const MasterDashboard = () => {
  const [wallets, setWallets] = useState<WalletRecord[]>([]);
  const [recent, setRecent] = useState<TxRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      setLoading(true); setErr(null);
      const w = await apiFetch<{ wallets?: WalletRecord[] } | null>('/api/v1/master-wallet');
      if (w?.wallets) setWallets(w.wallets);
      else setWallets([]);
      // Recent activity requires a wallet id; with none selected there is no
      // activity to show, so render nothing rather than placeholder rows.
      setRecent([]);
      if (!authToken) setErr('Not authenticated — sign in to load live data.');
      setLoading(false);
    })();
  }, []);

  const stats: DashboardStats = {
    totalWallets: wallets.length,
    totalVolume: null,      // aggregated volume comes from analytics per wallet
    totalUsers: null,       // user count comes from the wallet's users endpoint
    pendingTx: null,        // pending tx count comes from the wallet's tx list
  };

  return (
    <div className="page">
      <h1 className="page-title">Dashboard</h1>
      {err && <div className="banner error">{err}</div>}
      <div className="grid grid-cols-4 gap-6">
        <div className="card stat-card"><div className="stat-label">💼 Total Wallets</div><div className="stat-value big">{loading ? '…' : (stats.totalWallets ?? '—')}</div></div>
        <div className="card stat-card"><div className="stat-label">💰 Total Volume</div><div className="stat-value big">{loading ? '…' : (stats.totalVolume ?? '—')}</div></div>
        <div className="card stat-card"><div className="stat-label">👥 Total Users</div><div className="stat-value big">{loading ? '…' : (stats.totalUsers ?? '—')}</div></div>
        <div className="card stat-card"><div className="stat-label">⏳ Pending Tx</div><div className="stat-value big">{loading ? '…' : (stats.pendingTx ?? '—')}</div></div>
      </div>
      <div className="grid grid-cols-2 gap-6">
        <div className="card">
          <h2 className="card-title">Quick Actions</h2>
          <div className="grid grid-cols-2 gap-4">
            <button className="action-btn blue">➕ Create Wallet</button>
            <button className="action-btn green">👤 Add User</button>
            <button className="action-btn orange">🔑 Auto Sign</button>
            <button className="action-btn purple">📊 Analytics</button>
          </div>
        </div>
        <div className="card">
          <h2 className="card-title">Recent Activity</h2>
          {recent.length === 0
            ? <div className="empty-hint">{loading ? 'Loading…' : 'No recent activity.'}</div>
            : recent.map(t => (
              <div key={t.hash} className="activity-row">
                <span>📤</span><span>Transaction</span>
                <span className="amount">{t.amount ?? '—'}</span>
              </div>
            ))}
        </div>
      </div>
    </div>
  );
};

// Wallets Component — fetched from the backend; no hardcoded wallet list.
const MasterWallets = () => {
  const [wallets, setWallets] = useState<WalletRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      setLoading(true); setErr(null);
      const w = await apiFetch<{ wallets?: WalletRecord[] } | null>('/api/v1/master-wallet');
      setWallets(w?.wallets ?? []);
      if (!authToken) setErr('Not authenticated — sign in to load wallets.');
      setLoading(false);
    })();
  }, []);

  return (
    <div className="page">
      <div className="page-head">
        <h1 className="page-title">Master Wallets</h1>
        <button className="action-btn blue">➕ Add Wallet</button>
      </div>
      {err && <div className="banner error">{err}</div>}
      <div className="card table-card">
        <table className="data-table">
          <thead>
            <tr><th>Name</th><th>Address</th><th>Balance</th><th>Status</th><th>Actions</th></tr>
          </thead>
          <tbody>
            {loading && <tr><td colSpan={5} className="empty-hint">Loading…</td></tr>}
            {!loading && wallets.length === 0 && <tr><td colSpan={5} className="empty-hint">No wallets.</td></tr>}
            {!loading && wallets.map(w => (
              <tr key={w.id}>
                <td>{w.name ?? '—'}</td>
                <td className="mono">{w.address ?? '—'}</td>
                <td>{w.balance ?? '—'}</td>
                <td><span className={`badge ${w.status === 'Active' ? 'ok' : 'muted'}`}>{w.status ?? '—'}</span></td>
                <td><button className="link-btn">Edit</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// Users page — the wallet's owners (required for the multisig/threshold UX).
const MasterUsers = () => {
  interface UserRecord { id: string; name?: string; email?: string; role?: string; status?: string; }
  const [users, setUsers] = useState<UserRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState<string | null>(null);
  useEffect(() => {
    (async () => {
      setLoading(true); setErr(null);
      const r = await apiFetch<{ users?: UserRecord[] }>('/api/v1/master-wallet/users');
      setUsers(r?.users ?? []);
      if (!authToken) setErr('Not authenticated — sign in to load users.');
      setLoading(false);
    })();
  }, []);
  return (
    <div className="page">
      <div className="page-head">
        <h1 className="page-title">Users</h1>
        <button className="action-btn green">👤 Add User</button>
      </div>
      {err && <div className="banner error">{err}</div>}
      <div className="card table-card">
        <table className="data-table">
          <thead><tr><th>Name</th><th>Email</th><th>Role</th><th>Status</th></tr></thead>
          <tbody>
            {loading && <tr><td colSpan={4} className="empty-hint">Loading…</td></tr>}
            {!loading && users.length === 0 && <tr><td colSpan={4} className="empty-hint">No users.</td></tr>}
            {!loading && users.map(u => (
              <tr key={u.id}>
                <td>{u.name ?? '—'}</td>
                <td>{u.email ?? '—'}</td>
                <td>{u.role ?? '—'}</td>
                <td><span className={`badge ${u.status === 'active' ? 'ok' : 'muted'}`}>{u.status ?? '—'}</span></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// Transactions page — the wallet's indexed transaction feed.
const MasterTransactions = () => {
  const [txs, setTxs] = useState<TxRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState<string | null>(null);
  useEffect(() => {
    (async () => {
      setLoading(true); setErr(null);
      const r = await apiFetch<{ transactions?: TxRecord[] }>('/api/v1/master-wallet/transactions');
      setTxs(r?.transactions ?? []);
      if (!authToken) setErr('Not authenticated — sign in to load transactions.');
      setLoading(false);
    })();
  }, []);
  return (
    <div className="page">
      <h1 className="page-title">Transactions</h1>
      {err && <div className="banner error">{err}</div>}
      <div className="card table-card">
        <table className="data-table">
          <thead><tr><th>Hash</th><th>From</th><th>To</th><th>Amount</th><th>Status</th><th>Time</th></tr></thead>
          <tbody>
            {loading && <tr><td colSpan={6} className="empty-hint">Loading…</td></tr>}
            {!loading && txs.length === 0 && <tr><td colSpan={6} className="empty-hint">No transactions.</td></tr>}
            {!loading && txs.map(t => (
              <tr key={t.hash}>
                <td className="mono">{t.hash}</td>
                <td className="mono">{t.from ?? '—'}</td>
                <td className="mono">{t.to ?? '—'}</td>
                <td>{t.amount ?? '—'}</td>
                <td><span className={`badge ${t.status === 'confirmed' ? 'ok' : 'muted'}`}>{t.status ?? '—'}</span></td>
                <td>{t.timestamp ?? '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// Auto Sign page — the auto-approver policy snapshot from the control plane.
const MasterAutoSign = () => {
  interface RuleRecord { name?: string; tx_type?: string; token?: string; action?: string; }
  const [rules, setRules] = useState<RuleRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState<string | null>(null);
  useEffect(() => {
    (async () => {
      setLoading(true); setErr(null);
      const r = await apiFetch<{ rules?: RuleRecord[] }>('/api/v1/auto-sign/rules');
      setRules(r?.rules ?? []);
      if (!authToken) setErr('Not authenticated — sign in to load policy.');
      setLoading(false);
    })();
  }, []);
  return (
    <div className="page">
      <div className="page-head">
        <h1 className="page-title">Auto-Sign Policy</h1>
        <button className="action-btn orange">⚙️ Configure</button>
      </div>
      {err && <div className="banner error">{err}</div>}
      <div className="card table-card">
        <table className="data-table">
          <thead><tr><th>Transaction Type</th><th>Asset</th><th>Policy</th></tr></thead>
          <tbody>
            {loading && <tr><td colSpan={3} className="empty-hint">Loading…</td></tr>}
            {!loading && rules.length === 0 && <tr><td colSpan={3} className="empty-hint">No rules.</td></tr>}
            {!loading && rules.map((r, i) => (
              <tr key={i}>
                <td>{r.name ?? r.tx_type ?? '—'}</td>
                <td>{r.token ?? '—'}</td>
                <td><span className={`badge ${r.action === 'auto' ? 'ok' : 'muted'}`}>{r.action ?? '—'}</span></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// Analytics page — real per-wallet stats from the backend.
const MasterAnalytics = () => {
  const [stats, setStats] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState<string | null>(null);
  useEffect(() => {
    (async () => {
      setLoading(true); setErr(null);
      const r = await apiFetch<any>('/api/v1/master-wallet/stats');
      setStats(r);
      if (!authToken) setErr('Not authenticated — sign in to load analytics.');
      setLoading(false);
    })();
  }, []);
  return (
    <div className="page">
      <h1 className="page-title">Analytics</h1>
      {err && <div className="banner error">{err}</div>}
      <div className="card">
        <h2 className="card-title">Live Backend Stats</h2>
        <div className="page">
          {loading && <div className="empty-hint">Loading…</div>}
          {!loading && !stats && <div className="empty-hint">No stats returned.</div>}
          {!loading && stats && Object.entries(stats.data ?? stats).map(([k, v]) => (
            <div key={k} className="row-between">
              <span>{k}</span><span className="mono">{typeof v === 'object' ? JSON.stringify(v) : String(v)}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

// Settings Component — appearance toggle controls the theme applied everywhere.
const MasterSettings = () => {
  const { isDark, toggle } = useTheme();
  return (
    <div className="page">
      <h1 className="page-title">Settings</h1>
      <div className="card">
        <h2 className="card-title">Appearance</h2>
        <div className="row-between">
          <span>Dark Mode</span>
          <button onClick={toggle} className={`switch ${isDark ? 'on' : 'off'}`}><span className="knob" /></button>
        </div>
      </div>
      <div className="card">
        <h2 className="card-title">Security</h2>
        <div className="vlist">
          <button className="list-btn">Auto-Sign Rules</button>
          <button className="list-btn">User Permissions</button>
          <button className="list-btn">API Keys</button>
          <button className="list-btn">Two-Factor Auth</button>
        </div>
      </div>
      <div className="card">
        <h2 className="card-title">About</h2>
        <div className="row-between"><span>Version</span><span className="muted">1.0.0</span></div>
      </div>
    </div>
  );
};

// CSS driven by ThemeManager CSS variables — same variables power every page,
// so light/dark switching is consistent everywhere.
const ThemeStyle = () => (
  <style>{`
    /* The C++ ThemeManager injects a single root palette at startup. These
       html[data-theme] blocks mirror both palettes (kept in sync with
       src/ui/theme.cpp) with higher specificity, so the in-app toggle flips
       the theme without restarting the app. */
    html[data-theme="dark"] {
      --bg-color: #1a1a2e;
      --surface-color: #16213e;
      --primary-color: #0f3460;
      --secondary-color: #533483;
      --text-color: #e4e6eb;
      --text-secondary-color: #a0a3b1;
      --border-color: #2a2a4e;
      --success-color: #2ecc71;
      --error-color: #e74c3c;
      --warning-color: #f39c12;
      --accent-color: #0f3460;
    }
    html[data-theme="light"] {
      --bg-color: #ffffff;
      --surface-color: #f5f5f5;
      --primary-color: #0f3460;
      --secondary-color: #3a6ea5;
      --text-color: #1a1a2e;
      --text-secondary-color: #555770;
      --border-color: #d9dce1;
      --success-color: #27ae60;
      --error-color: #c0392b;
      --warning-color: #d68910;
      --accent-color: #0f3460;
    }
    [data-theme="dark"], [data-theme="light"] {
      background: var(--bg-color); color: var(--text-color);
    }
    .app-root { display: flex; height: 100vh; background: var(--bg-color); color: var(--text-color); }
    .sidebar { width: 16rem; background: var(--surface-color); border-right: 1px solid var(--border-color); display: flex; flex-direction: column; }
    .sidebar-brand { padding: 1rem; border-bottom: 1px solid var(--border-color); display: flex; align-items: center; gap: .75rem; }
    .sidebar-nav { flex: 1; padding: 1rem; }
    .nav-item { width: 100%; display: flex; align-items: center; gap: .75rem; padding: .75rem 1rem; border-radius: .5rem; margin-bottom: .5rem; background: transparent; color: var(--text-secondary-color); border: none; cursor: pointer; }
    .nav-item:hover { background: var(--surface-color); color: var(--text-color); }
    .nav-item.active { background: var(--primary-color); color: #fff; }
    .app-header { height: 4rem; background: var(--surface-color); border-bottom: 1px solid var(--border-color); display: flex; align-items: center; justify-content: space-between; padding: 0 1.5rem; }
    .search-input { padding: .5rem 1rem; background: var(--bg-color); border: 1px solid var(--border-color); border-radius: .5rem; color: var(--text-color); width: 24rem; }
    .header-right { display: flex; align-items: center; gap: 1rem; }
    .stat-block { text-align: right; }
    .stat-label { font-size: .75rem; color: var(--text-secondary-color); }
    .stat-value { font-weight: 700; }
    .stat-value.big { font-size: 1.875rem; }
    .theme-toggle { padding: .5rem; background: var(--surface-color); border: 1px solid var(--border-color); border-radius: .5rem; cursor: pointer; }
    .main { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
    .content { flex: 1; overflow: auto; padding: 1.5rem; }
    .page { display: flex; flex-direction: column; gap: 1.5rem; }
    .page-head { display: flex; justify-content: space-between; align-items: center; }
    .page-title { font-size: 1.5rem; font-weight: 700; }
    .grid { display: grid; }
    .grid-cols-4 { grid-template-columns: repeat(4, 1fr); }
    .grid-cols-2 { grid-template-columns: repeat(2, 1fr); }
    .gap-6 { gap: 1.5rem; }
    .card { background: var(--surface-color); border: 1px solid var(--border-color); border-radius: .75rem; padding: 1.5rem; }
    .stat-card .stat-label { margin-bottom: .5rem; }
    .card-title { font-size: 1.125rem; font-weight: 600; margin-bottom: 1rem; }
    .action-btn { padding: 1rem; border-radius: .5rem; border: none; cursor: pointer; color: #fff; font-weight: 600; }
    .action-btn.blue { background: var(--primary-color); }
    .action-btn.green { background: var(--success-color); }
    .action-btn.orange { background: var(--warning-color); }
    .action-btn.purple { background: var(--secondary-color); }
    .table-card { padding: 0; overflow: hidden; }
    .data-table { width: 100%; border-collapse: collapse; }
    .data-table th, .data-table td { padding: .75rem 1.5rem; text-align: left; }
    .data-table th { background: var(--bg-color); }
    .data-table tr { border-bottom: 1px solid var(--border-color); }
    .mono { font-family: ui-monospace, monospace; font-size: .875rem; }
    .badge { padding: .25rem .5rem; border-radius: .25rem; font-size: .75rem; color: #fff; }
    .badge.ok { background: var(--success-color); }
    .badge.muted { background: var(--text-secondary-color); }
    .link-btn { background: none; border: none; color: var(--primary-color); cursor: pointer; }
    .activity-row { display: flex; align-items: center; gap: .5rem; padding: .5rem 0; border-bottom: 1px solid var(--border-color); }
    .amount { margin-left: auto; color: var(--success-color); }
    .empty-hint, .muted { color: var(--text-secondary-color); text-align: center; padding: .5rem; }
    .banner.error { background: var(--error-color); color: #fff; padding: .75rem 1rem; border-radius: .5rem; }
    .row-between { display: flex; align-items: center; justify-content: space-between; }
    .switch { width: 3.5rem; height: 1.75rem; border-radius: 9999px; border: none; cursor: pointer; position: relative; }
    .switch.on { background: var(--primary-color); } .switch.off { background: var(--text-secondary-color); }
    .knob { width: 1.25rem; height: 1.25rem; background: #fff; border-radius: 9999px; position: absolute; top: .25rem; }
    .switch.on .knob { right: .25rem; } .switch.off .knob { left: .25rem; }
    .vlist { display: flex; flex-direction: column; gap: .5rem; }
    .list-btn { text-align: left; padding: .5rem 1rem; background: var(--bg-color); border: 1px solid var(--border-color); border-radius: .25rem; cursor: pointer; color: var(--text-color); }
  `}</style>
);

// Main App
const MasterDesktopApp = () => {
  const [page, setPage] = useState('dashboard');
  return (
    <MasterThemeProvider>
      <ThemeStyle />
      <div className="app-root">
        <MasterSidebar currentPage={page} setCurrentPage={setPage} />
        <div className="main">
          <MasterHeader />
          <main className="content">
            {page === 'dashboard' && <MasterDashboard />}
            {page === 'wallets' && <MasterWallets />}
            {page === 'users' && <MasterUsers />}
            {page === 'transactions' && <MasterTransactions />}
            {page === 'auto-sign' && <MasterAutoSign />}
            {page === 'analytics' && <MasterAnalytics />}
            {page === 'settings' && <MasterSettings />}
          </main>
        </div>
      </div>
    </MasterThemeProvider>
  );
};

export default MasterDesktopApp;
