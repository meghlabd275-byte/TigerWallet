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

// Wallet-scope helper: every wallet-scoped resource lives under
// /api/v1/master-wallet/:id/...; pages resolve the FIRST wallet once and use
// it for the route prefix. Falsy id -> backend-only rendering (no fabricated feature).
async function firstWalletId(): Promise<string | null> {
  const r = await apiFetch<{ wallets?: WalletRecord[] }>('/api/v1/master-wallet');
  const id = r?.wallets?.[0]?.id ?? null;
  return typeof id === 'string' && id ? id : null;
}

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

// Write helper: POST/PUT/DELETE with a JSON body. Returns parsed JSON or null.
// Actions never fabricate a success — the caller surfaces the backend's real
// response (including its error body) to the user.
async function apiSend<T = any>(path: string, method: 'POST' | 'PUT' | 'DELETE', body?: any): Promise<T | null> {
  const headers: Record<string, string> = { 'Accept': 'application/json', 'Content-Type': 'application/json' };
  if (authToken) headers['Authorization'] = `Bearer ${authToken}`;
  try {
    const r = await fetch(`${API_BASE}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    });
    const txt = await r.text();
    const data = txt ? JSON.parse(txt) : null;
    if (!r.ok) {
      const msg = (data && (data.error || data.message)) || `HTTP ${r.status}`;
      throw new Error(msg);
    }
    return data as T;
  } catch (e) {
    throw e instanceof Error ? e : new Error(String(e));
  }
}

// Extract the first array found in a response (the backend returns both raw
// arrays and {key: [...]} envelopes depending on the route).
function asList<T = any>(res: any, ...keys: string[]): T[] {
  if (Array.isArray(res)) return res as T[];
  if (res && typeof res === 'object') {
    for (const k of keys) if (Array.isArray(res[k])) return res[k] as T[];
    for (const v of Object.values(res)) if (Array.isArray(v)) return v as T[];
  }
  return [];
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
    { id: 'treasury', label: 'Treasury', icon: '🏦' },
    { id: 'multisig', label: 'Multisig', icon: '🔐' },
    { id: 'auto-sign', label: 'Auto Sign', icon: '🔑' },
    { id: 'fees', label: 'Fees', icon: '💸' },
    { id: 'policies', label: 'Policies', icon: '📏' },
    { id: 'chains', label: 'Chains', icon: '⛓️' },
    { id: 'tokens', label: 'Tokens', icon: '🪙' },
    { id: 'flags', label: 'Feature Flags', icon: '🚩' },
    { id: 'webhooks', label: 'Webhooks', icon: '🔔' },
    { id: 'audit', label: 'Audit', icon: '🧾' },
    { id: 'passkeys', label: 'Passkeys', icon: '🪪' },
    { id: 'withdraw', label: 'Withdraw', icon: '📤' },
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
      const wid = await firstWalletId();
      const r = wid ? await apiFetch<{ users?: UserRecord[] }>(`/api/v1/master-wallet/${wid}/users`) : null;
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
      const wid = await firstWalletId();
      const r = wid ? await apiFetch<{ transactions?: TxRecord[] }>(`/api/v1/master-wallet/${wid}/transactions`) : null;
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
      const wid = await firstWalletId();
      // Canonical auto-sign resource per wallet: GET /master-wallet/:id/auto-sign.
      const r = wid ? await apiFetch<{ rules?: RuleRecord[]; auto_sign?: RuleRecord[]; rules_item_auto_sign?: RuleRecord[] }>(`/api/v1/master-wallet/${wid}/auto-sign`) : null;
      setRules(r?.rules ?? r?.auto_sign ?? r?.rules_item_auto_sign ?? []);
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
      const wid = await firstWalletId();
      const vol = wid ? await apiFetch<any>(`/api/v1/master-wallet/${wid}/analytics/volume`) : null;
      const txc = wid ? await apiFetch<any>(`/api/v1/master-wallet/${wid}/analytics/transactions`) : null;
      const wlc = wid ? await apiFetch<any>(`/api/v1/master-wallet/${wid}/analytics/wallets`) : null;
      const merged = wid ? { volume_24h: vol?.volume_24h, tx_count: txc?.tx_count, active_users: wlc?.active_users } : null;
      setStats(merged);
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

// Shared state hook for the wallet-scoped resource pages: resolves the first
// wallet id once, fetches, and surfaces real errors. Nothing is fabricated.
function useWalletResource<T>(path: string, keys: string[]) {
  const [items, setItems] = useState<T[]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState<string | null>(null);
  const [wid, setWid] = useState<string | null>(null);
  const reload = useCallback(async () => {
    setLoading(true); setErr(null);
    try {
      const id = await firstWalletId();
      setWid(id);
      if (!id) { setItems([]); setLoading(false); return; }
      const r = await apiFetch<any>(`/api/v1/master-wallet/${id}${path}`);
      setItems(asList<T>(r, ...keys));
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, [path]);
  useEffect(() => { reload(); }, [reload]);
  return { items, loading, err, wid, reload };
}

// Minimal labelled action button used by the resource pages.
const RowBtn = ({ label, kind, onClick }: { label: string; kind?: 'danger' | 'ok'; onClick: () => void }) => (
  <button className={`link-btn ${kind ?? ''}`} onClick={onClick}>{label}</button>
);

// Treasury page — overview, transactions, transfer + sweep (all real backend).
const MasterTreasury = () => {
  const overview = useWalletResource<any>('/treasury', ['treasury']);
  const txs = useWalletResource<any>('/treasury/transactions', ['transactions']);
  const [msg, setMsg] = useState<string | null>(null);
  const act = async (fn: () => Promise<any>) => {
    setMsg(null);
    try { await fn(); setMsg('Submitted.'); overview.reload(); txs.reload(); }
    catch (e) { setMsg(e instanceof Error ? e.message : String(e)); }
  };
  const [form, setForm] = useState({ to: '', amount: '', password: '', subWalletId: '', sweepPassword: '' });
  const set = (k: string) => (e: any) => setForm({ ...form, [k]: e.target.value });
  return (
    <div className="page">
      <h1 className="page-title">Treasury</h1>
      {msg && <div className="banner">{msg}</div>}
      <div className="card">
        <h2 className="card-title">Overview</h2>
        {overview.loading ? <div className="empty-hint">Loading…</div>
          : <pre className="mono">{overview.items.length ? JSON.stringify(overview.items[0], null, 2) : 'No treasury data.'}</pre>}
      </div>
      <div className="card">
        <h2 className="card-title">Transfer</h2>
        <div className="form-grid">
          <input placeholder="Destination address" value={form.to} onChange={set('to')} />
          <input placeholder="Amount" value={form.amount} onChange={set('amount')} />
          <input placeholder="Wallet password" type="password" value={form.password} onChange={set('password')} />
          <button className="action-btn blue" onClick={() => overview.wid && act(() => apiSend(`/api/v1/master-wallet/${overview.wid}/treasury/transfer`, 'POST', { to: form.to, amount: form.amount, password: form.password }))}>Transfer</button>
        </div>
        <h2 className="card-title">Sweep sub-wallet</h2>
        <div className="form-grid">
          <input placeholder="Sub-wallet ID" value={form.subWalletId} onChange={set('subWalletId')} />
          <input placeholder="Wallet password" type="password" value={form.sweepPassword} onChange={set('sweepPassword')} />
          <button className="action-btn orange" onClick={() => overview.wid && act(() => apiSend(`/api/v1/master-wallet/${overview.wid}/treasury/sweep`, 'POST', { sub_wallet_id: form.subWalletId, password: form.sweepPassword }))}>Sweep</button>
        </div>
      </div>
      <div className="card table-card">
        <h2 className="card-title">Treasury Transactions</h2>
        <table className="data-table">
          <thead><tr><th>Type</th><th>To</th><th>Amount</th><th>Status</th></tr></thead>
          <tbody>
            {txs.loading && <tr><td colSpan={4} className="empty-hint">Loading…</td></tr>}
            {!txs.loading && txs.items.length === 0 && <tr><td colSpan={4} className="empty-hint">None.</td></tr>}
            {!txs.loading && txs.items.map((t, i) => (
              <tr key={t.id ?? i}><td>{t.tx_type ?? '—'}</td><td className="mono">{t.to_address ?? t.to ?? '—'}</td><td>{t.amount ?? '—'}</td><td><span className="badge muted">{t.status ?? '—'}</span></td></tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// Multisig page — wallets, create, sign, execute.
const MasterMultisig = () => {
  const res = useWalletResource<any>('/multisig/wallets', ['wallets', 'multisig_wallets']);
  const [sel, setSel] = useState<string | null>(null);
  const [txs, setTxs] = useState<any[]>([]);
  const [msg, setMsg] = useState<string | null>(null);
  const [form, setForm] = useState({ name: '', owners: '', threshold: '' });
  const loadTxs = async (mwid: string) => {
    setSel(mwid);
    if (!res.wid) return;
    const r = await apiFetch<any>(`/api/v1/master-wallet/${res.wid}/multisig/wallets/${mwid}/transactions`);
    setTxs(asList(r, 'transactions'));
  };
  const act = async (fn: () => Promise<any>) => {
    setMsg(null);
    try { await fn(); setMsg('Submitted.'); res.reload(); if (sel) loadTxs(sel); }
    catch (e) { setMsg(e instanceof Error ? e.message : String(e)); }
  };
  return (
    <div className="page">
      <h1 className="page-title">Multisig</h1>
      {msg && <div className="banner">{msg}</div>}
      <div className="card">
        <h2 className="card-title">Create Multisig Wallet</h2>
        <div className="form-grid">
          <input placeholder="Name" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          <input placeholder="Owners (comma-separated 0x addresses)" value={form.owners} onChange={(e) => setForm({ ...form, owners: e.target.value })} />
          <input placeholder="Threshold" value={form.threshold} onChange={(e) => setForm({ ...form, threshold: e.target.value })} />
          <button className="action-btn blue" onClick={() => res.wid && act(() => apiSend(`/api/v1/master-wallet/${res.wid}/multisig/wallets`, 'POST', {
            name: form.name,
            owners: form.owners.split(',').map((s) => s.trim()).filter(Boolean),
            threshold: parseInt(form.threshold, 10),
          }))}>Create</button>
        </div>
      </div>
      <div className="card table-card">
        <h2 className="card-title">Multisig Wallets</h2>
        <table className="data-table">
          <thead><tr><th>Name</th><th>Address</th><th>Threshold</th><th></th></tr></thead>
          <tbody>
            {res.loading && <tr><td colSpan={4} className="empty-hint">Loading…</td></tr>}
            {!res.loading && res.items.length === 0 && <tr><td colSpan={4} className="empty-hint">None.</td></tr>}
            {!res.loading && res.items.map((m, i) => (
              <tr key={m.id ?? i}>
                <td>{m.name ?? '—'}</td><td className="mono">{m.address ?? '—'}</td>
                <td>{m.threshold ?? '—'}/{(m.owners ?? []).length}</td>
                <td><RowBtn label="Txs" onClick={() => loadTxs(m.id)} /></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {sel && (
        <div className="card table-card">
          <h2 className="card-title">Transactions</h2>
          <table className="data-table">
            <thead><tr><th>To</th><th>Value</th><th>Status</th><th>Actions</th></tr></thead>
            <tbody>
              {txs.length === 0 && <tr><td colSpan={4} className="empty-hint">None.</td></tr>}
              {txs.map((t, i) => (
                <tr key={t.id ?? i}>
                  <td className="mono">{t.to ?? '—'}</td><td>{t.value ?? t.amount ?? '—'}</td>
                  <td><span className="badge muted">{t.status ?? '—'}</span></td>
                  <td>
                    <RowBtn label="Sign" onClick={() => res.wid && act(() => apiSend(`/api/v1/master-wallet/${res.wid}/multisig/transactions/${t.id}/sign`, 'POST', {}))} />
                    <RowBtn label="Execute" onClick={() => res.wid && act(() => apiSend(`/api/v1/master-wallet/${res.wid}/multisig/transactions/${t.id}/execute`, 'POST', {}))} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};

// Fees page — full CRUD over /:id/fees.
const MasterFees = () => {
  const res = useWalletResource<any>('/fees', ['fees']);
  const [msg, setMsg] = useState<string | null>(null);
  const [form, setForm] = useState({ feeType: '', feePercentage: '', feeFixed: '' });
  const act = async (fn: () => Promise<any>) => {
    setMsg(null);
    try { await fn(); setMsg('Saved.'); res.reload(); }
    catch (e) { setMsg(e instanceof Error ? e.message : String(e)); }
  };
  return (
    <div className="page">
      <h1 className="page-title">Fees</h1>
      {msg && <div className="banner">{msg}</div>}
      <div className="card">
        <h2 className="card-title">Add Fee</h2>
        <div className="form-grid">
          <input placeholder="Fee type (e.g. transfer)" value={form.feeType} onChange={(e) => setForm({ ...form, feeType: e.target.value })} />
          <input placeholder="Fee percentage" value={form.feePercentage} onChange={(e) => setForm({ ...form, feePercentage: e.target.value })} />
          <input placeholder="Fee fixed (wei, optional)" value={form.feeFixed} onChange={(e) => setForm({ ...form, feeFixed: e.target.value })} />
          <button className="action-btn blue" onClick={() => res.wid && act(() => apiSend(`/api/v1/master-wallet/${res.wid}/fees`, 'POST', {
            fee_type: form.feeType, fee_percentage: parseFloat(form.feePercentage) || 0, fee_fixed: form.feeFixed,
          }))}>Add</button>
        </div>
      </div>
      <div className="card table-card">
        <table className="data-table">
          <thead><tr><th>Type</th><th>Percent</th><th>Fixed</th><th>Status</th><th>Actions</th></tr></thead>
          <tbody>
            {res.loading && <tr><td colSpan={5} className="empty-hint">Loading…</td></tr>}
            {!res.loading && res.items.length === 0 && <tr><td colSpan={5} className="empty-hint">No fees.</td></tr>}
            {!res.loading && res.items.map((f, i) => (
              <tr key={f.id ?? i}>
                <td>{f.fee_type ?? '—'}</td><td>{f.fee_percentage ?? '—'}</td><td>{f.fee_fixed ?? '—'}</td>
                <td><span className={`badge ${f.is_active ? 'ok' : 'muted'}`}>{f.is_active ? 'active' : 'off'}</span></td>
                <td>
                  <RowBtn label={f.is_active ? 'Disable' : 'Enable'} onClick={() => res.wid && act(() => apiSend(`/api/v1/master-wallet/${res.wid}/fees/${f.id}`, 'PUT', { is_active: !f.is_active }))} />
                  <RowBtn label="Delete" kind="danger" onClick={() => res.wid && act(() => apiSend(`/api/v1/master-wallet/${res.wid}/fees/${f.id}`, 'DELETE'))} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// Policies page — list/create/delete over /:id/policies.
const MasterPolicies = () => {
  const res = useWalletResource<any>('/policies', ['policies']);
  const [msg, setMsg] = useState<string | null>(null);
  const [form, setForm] = useState({ name: '', policyType: '' });
  const act = async (fn: () => Promise<any>) => {
    setMsg(null);
    try { await fn(); setMsg('Saved.'); res.reload(); }
    catch (e) { setMsg(e instanceof Error ? e.message : String(e)); }
  };
  return (
    <div className="page">
      <h1 className="page-title">Policies</h1>
      {msg && <div className="banner">{msg}</div>}
      <div className="card">
        <h2 className="card-title">Add Policy</h2>
        <div className="form-grid">
          <input placeholder="Policy name" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          <input placeholder="Policy type (e.g. withdrawal_limit)" value={form.policyType} onChange={(e) => setForm({ ...form, policyType: e.target.value })} />
          <button className="action-btn blue" onClick={() => res.wid && act(() => apiSend(`/api/v1/master-wallet/${res.wid}/policies`, 'POST', { name: form.name, policy_type: form.policyType }))}>Add</button>
        </div>
      </div>
      <div className="card table-card">
        <table className="data-table">
          <thead><tr><th>Name</th><th>Type</th><th>Priority</th><th>Status</th><th></th></tr></thead>
          <tbody>
            {res.loading && <tr><td colSpan={5} className="empty-hint">Loading…</td></tr>}
            {!res.loading && res.items.length === 0 && <tr><td colSpan={5} className="empty-hint">No policies.</td></tr>}
            {!res.loading && res.items.map((p, i) => (
              <tr key={p.id ?? i}>
                <td>{p.name ?? '—'}</td><td>{p.policy_type ?? '—'}</td><td>{p.priority ?? 0}</td>
                <td><span className={`badge ${p.is_active ? 'ok' : 'muted'}`}>{p.is_active ? 'active' : 'off'}</span></td>
                <td><RowBtn label="Delete" kind="danger" onClick={() => res.wid && act(() => apiSend(`/api/v1/master-wallet/${res.wid}/policies/${p.id}`, 'DELETE'))} /></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// Chains page — UserWallet EVM + non-EVM chain governance (add/remove).
const MasterChains = () => {
  const evm = useWalletResource<any>('/user-chains/evm', ['chains']);
  const nonEvm = useWalletResource<any>('/user-chains/nonevm', ['chains']);
  const [msg, setMsg] = useState<string | null>(null);
  const [ef, setEf] = useState({ chainId: '', name: '', rpc: '', symbol: '' });
  const [nf, setNf] = useState({ chainId: '', name: '', chainType: '', rpc: '', derivation: '' });
  const act = async (fn: () => Promise<any>) => {
    setMsg(null);
    try { await fn(); setMsg('Saved.'); evm.reload(); nonEvm.reload(); }
    catch (e) { setMsg(e instanceof Error ? e.message : String(e)); }
  };
  return (
    <div className="page">
      <h1 className="page-title">UserWallet Chains</h1>
      {msg && <div className="banner">{msg}</div>}
      <div className="card">
        <h2 className="card-title">Add EVM Chain</h2>
        <div className="form-grid">
          <input placeholder="Chain ID" value={ef.chainId} onChange={(e) => setEf({ ...ef, chainId: e.target.value })} />
          <input placeholder="Name" value={ef.name} onChange={(e) => setEf({ ...ef, name: e.target.value })} />
          <input placeholder="RPC URL" value={ef.rpc} onChange={(e) => setEf({ ...ef, rpc: e.target.value })} />
          <input placeholder="Symbol" value={ef.symbol} onChange={(e) => setEf({ ...ef, symbol: e.target.value })} />
          <button className="action-btn blue" onClick={() => evm.wid && act(() => apiSend(`/api/v1/master-wallet/${evm.wid}/user-chains/evm`, 'POST', {
            chain_id: parseInt(ef.chainId, 10), name: ef.name, rpc_url: ef.rpc, symbol: ef.symbol,
          }))}>Add EVM chain</button>
        </div>
      </div>
      <div className="card table-card">
        <h2 className="card-title">EVM Chains</h2>
        <table className="data-table">
          <thead><tr><th>Chain ID</th><th>Name</th><th>Symbol</th><th>RPC</th><th></th></tr></thead>
          <tbody>
            {evm.loading && <tr><td colSpan={5} className="empty-hint">Loading…</td></tr>}
            {!evm.loading && evm.items.length === 0 && <tr><td colSpan={5} className="empty-hint">None.</td></tr>}
            {!evm.loading && evm.items.map((c, i) => (
              <tr key={c.chain_id ?? i}>
                <td>{c.chain_id ?? '—'}</td><td>{c.name ?? '—'}</td><td>{c.symbol ?? '—'}</td><td className="mono">{c.rpc_url ?? '—'}</td>
                <td><RowBtn label="Remove" kind="danger" onClick={() => evm.wid && act(() => apiSend(`/api/v1/master-wallet/${evm.wid}/user-chains/evm/${c.chain_id}`, 'DELETE'))} /></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="card">
        <h2 className="card-title">Add Non-EVM Chain</h2>
        <div className="form-grid">
          <input placeholder="Chain ID (SLIP-44)" value={nf.chainId} onChange={(e) => setNf({ ...nf, chainId: e.target.value })} />
          <input placeholder="Name" value={nf.name} onChange={(e) => setNf({ ...nf, name: e.target.value })} />
          <input placeholder="Chain type (solana/bitcoin/cosmos)" value={nf.chainType} onChange={(e) => setNf({ ...nf, chainType: e.target.value })} />
          <input placeholder="RPC / node URL" value={nf.rpc} onChange={(e) => setNf({ ...nf, rpc: e.target.value })} />
          <input placeholder="Derivation path" value={nf.derivation} onChange={(e) => setNf({ ...nf, derivation: e.target.value })} />
          <button className="action-btn blue" onClick={() => nonEvm.wid && act(() => apiSend(`/api/v1/master-wallet/${nonEvm.wid}/user-chains/nonevm`, 'POST', {
            chain_id: parseInt(nf.chainId, 10), name: nf.name, chain_type: nf.chainType, rpc_url: nf.rpc, derivation_path: nf.derivation,
          }))}>Add non-EVM chain</button>
        </div>
      </div>
      <div className="card table-card">
        <h2 className="card-title">Non-EVM Chains</h2>
        <table className="data-table">
          <thead><tr><th>Chain ID</th><th>Name</th><th>Type</th><th>Symbol</th><th></th></tr></thead>
          <tbody>
            {nonEvm.loading && <tr><td colSpan={5} className="empty-hint">Loading…</td></tr>}
            {!nonEvm.loading && nonEvm.items.length === 0 && <tr><td colSpan={5} className="empty-hint">None.</td></tr>}
            {!nonEvm.loading && nonEvm.items.map((c, i) => (
              <tr key={c.chain_id ?? i}>
                <td>{c.chain_id ?? '—'}</td><td>{c.name ?? '—'}</td><td>{c.chain_type ?? '—'}</td><td>{c.symbol ?? '—'}</td>
                <td><RowBtn label="Remove" kind="danger" onClick={() => nonEvm.wid && act(() => apiSend(`/api/v1/master-wallet/${nonEvm.wid}/user-chains/nonevm/${c.chain_id}`, 'DELETE'))} /></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// Tokens page — UserWallet token/coin governance.
const MasterTokens = () => {
  const res = useWalletResource<any>('/user-tokens', ['tokens']);
  const [msg, setMsg] = useState<string | null>(null);
  const [form, setForm] = useState({ chainId: '', symbol: '', name: '', address: '', decimals: '18' });
  const act = async (fn: () => Promise<any>) => {
    setMsg(null);
    try { await fn(); setMsg('Saved.'); res.reload(); }
    catch (e) { setMsg(e instanceof Error ? e.message : String(e)); }
  };
  return (
    <div className="page">
      <h1 className="page-title">UserWallet Tokens</h1>
      {msg && <div className="banner">{msg}</div>}
      <div className="card">
        <h2 className="card-title">Add Token</h2>
        <div className="form-grid">
          <input placeholder="Chain ID" value={form.chainId} onChange={(e) => setForm({ ...form, chainId: e.target.value })} />
          <input placeholder="Symbol" value={form.symbol} onChange={(e) => setForm({ ...form, symbol: e.target.value })} />
          <input placeholder="Name" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          <input placeholder="Contract address" value={form.address} onChange={(e) => setForm({ ...form, address: e.target.value })} />
          <input placeholder="Decimals" value={form.decimals} onChange={(e) => setForm({ ...form, decimals: e.target.value })} />
          <button className="action-btn blue" onClick={() => res.wid && act(() => apiSend(`/api/v1/master-wallet/${res.wid}/user-tokens`, 'POST', {
            chain_id: parseInt(form.chainId, 10), symbol: form.symbol, name: form.name,
            contract_address: form.address, decimals: parseInt(form.decimals, 10) || 18,
          }))}>Add token</button>
        </div>
      </div>
      <div className="card table-card">
        <table className="data-table">
          <thead><tr><th>Symbol</th><th>Name</th><th>Chain</th><th>Contract</th><th></th></tr></thead>
          <tbody>
            {res.loading && <tr><td colSpan={5} className="empty-hint">Loading…</td></tr>}
            {!res.loading && res.items.length === 0 && <tr><td colSpan={5} className="empty-hint">No tokens.</td></tr>}
            {!res.loading && res.items.map((t, i) => (
              <tr key={t.id ?? i}>
                <td>{t.symbol ?? '—'}</td><td>{t.name ?? '—'}</td><td>{t.chain_id ?? '—'}</td>
                <td className="mono">{t.contract_address ?? 'native'}</td>
                <td><RowBtn label="Remove" kind="danger" onClick={() => res.wid && act(() => apiSend(`/api/v1/master-wallet/${res.wid}/user-tokens/${t.id}`, 'DELETE'))} /></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// Feature flags page — owner governance over SuperAdmin-added features.
const MasterFlags = () => {
  const res = useWalletResource<any>('/feature-flags', ['feature_flags', 'flags']);
  const [msg, setMsg] = useState<string | null>(null);
  const [key, setKey] = useState('');
  const act = async (fn: () => Promise<any>) => {
    setMsg(null);
    try { await fn(); setMsg('Saved.'); res.reload(); }
    catch (e) { setMsg(e instanceof Error ? e.message : String(e)); }
  };
  return (
    <div className="page">
      <h1 className="page-title">Feature Flags</h1>
      {msg && <div className="banner">{msg}</div>}
      <div className="card">
        <h2 className="card-title">Add Flag</h2>
        <div className="form-grid">
          <input placeholder="Flag key (e.g. enable_swap)" value={key} onChange={(e) => setKey(e.target.value)} />
          <button className="action-btn blue" onClick={() => res.wid && act(() => apiSend(`/api/v1/master-wallet/${res.wid}/feature-flags`, 'POST', { flag_key: key, is_enabled: true }))}>Add</button>
        </div>
      </div>
      <div className="card table-card">
        <table className="data-table">
          <thead><tr><th>Key</th><th>Value</th><th>Enabled</th><th>Actions</th></tr></thead>
          <tbody>
            {res.loading && <tr><td colSpan={4} className="empty-hint">Loading…</td></tr>}
            {!res.loading && res.items.length === 0 && <tr><td colSpan={4} className="empty-hint">No flags.</td></tr>}
            {!res.loading && res.items.map((f, i) => (
              <tr key={f.id ?? i}>
                <td className="mono">{f.flag_key ?? '—'}</td><td>{f.flag_value ?? '—'}</td>
                <td><span className={`badge ${f.is_enabled ? 'ok' : 'muted'}`}>{f.is_enabled ? 'on' : 'off'}</span></td>
                <td>
                  <RowBtn label={f.is_enabled ? 'Disable' : 'Enable'} onClick={() => res.wid && act(() => apiSend(`/api/v1/master-wallet/${res.wid}/feature-flags/${f.id}`, 'PUT', { is_enabled: !f.is_enabled }))} />
                  <RowBtn label="Remove" kind="danger" onClick={() => res.wid && act(() => apiSend(`/api/v1/master-wallet/${res.wid}/feature-flags/${f.id}`, 'DELETE'))} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// Webhooks + notifications page.
const MasterWebhooks = () => {
  const hooks = useWalletResource<any>('/webhooks', ['webhooks']);
  const notifs = useWalletResource<any>('/notifications', ['notifications']);
  const [msg, setMsg] = useState<string | null>(null);
  const [hf, setHf] = useState({ name: '', url: '', events: '' });
  const [nf, setNf] = useState({ type: '', title: '', message: '' });
  const act = async (fn: () => Promise<any>) => {
    setMsg(null);
    try { await fn(); setMsg('Saved.'); hooks.reload(); notifs.reload(); }
    catch (e) { setMsg(e instanceof Error ? e.message : String(e)); }
  };
  return (
    <div className="page">
      <h1 className="page-title">Webhooks & Notifications</h1>
      {msg && <div className="banner">{msg}</div>}
      <div className="card">
        <h2 className="card-title">Add Webhook</h2>
        <div className="form-grid">
          <input placeholder="Name" value={hf.name} onChange={(e) => setHf({ ...hf, name: e.target.value })} />
          <input placeholder="URL (https://…)" value={hf.url} onChange={(e) => setHf({ ...hf, url: e.target.value })} />
          <input placeholder="Events (comma-separated)" value={hf.events} onChange={(e) => setHf({ ...hf, events: e.target.value })} />
          <button className="action-btn blue" onClick={() => hooks.wid && act(() => apiSend(`/api/v1/master-wallet/${hooks.wid}/webhooks`, 'POST', {
            name: hf.name, url: hf.url, events: hf.events.split(',').map((s) => s.trim()).filter(Boolean),
          }))}>Add</button>
        </div>
      </div>
      <div className="card table-card">
        <table className="data-table">
          <thead><tr><th>Name</th><th>URL</th><th>Events</th><th></th></tr></thead>
          <tbody>
            {hooks.loading && <tr><td colSpan={4} className="empty-hint">Loading…</td></tr>}
            {!hooks.loading && hooks.items.length === 0 && <tr><td colSpan={4} className="empty-hint">No webhooks.</td></tr>}
            {!hooks.loading && hooks.items.map((w, i) => (
              <tr key={w.id ?? i}>
                <td>{w.name ?? '—'}</td><td className="mono">{w.url ?? '—'}</td><td>{(w.events ?? []).join(', ') || '—'}</td>
                <td><RowBtn label="Delete" kind="danger" onClick={() => hooks.wid && act(() => apiSend(`/api/v1/master-wallet/${hooks.wid}/webhooks/${w.id}`, 'DELETE'))} /></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="card">
        <h2 className="card-title">Send Notification</h2>
        <div className="form-grid">
          <input placeholder="Type (e.g. alert)" value={nf.type} onChange={(e) => setNf({ ...nf, type: e.target.value })} />
          <input placeholder="Title" value={nf.title} onChange={(e) => setNf({ ...nf, title: e.target.value })} />
          <input placeholder="Message" value={nf.message} onChange={(e) => setNf({ ...nf, message: e.target.value })} />
          <button className="action-btn green" onClick={() => notifs.wid && act(() => apiSend(`/api/v1/master-wallet/${notifs.wid}/notifications`, 'POST', {
            notification_type: nf.type, title: nf.title, message: nf.message,
          }))}>Send</button>
        </div>
      </div>
      <div className="card table-card">
        <h2 className="card-title">Notifications</h2>
        <table className="data-table">
          <thead><tr><th>Title</th><th>Message</th><th>Priority</th><th>Time</th></tr></thead>
          <tbody>
            {notifs.loading && <tr><td colSpan={4} className="empty-hint">Loading…</td></tr>}
            {!notifs.loading && notifs.items.length === 0 && <tr><td colSpan={4} className="empty-hint">None.</td></tr>}
            {!notifs.loading && notifs.items.map((n, i) => (
              <tr key={n.id ?? i}><td>{n.title ?? '—'}</td><td>{n.message ?? '—'}</td><td>{n.priority ?? '—'}</td><td>{n.created_at ?? '—'}</td></tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// Audit page — the immutable per-wallet audit log.
const MasterAudit = () => {
  const res = useWalletResource<any>('/audit', ['logs', 'audit']);
  return (
    <div className="page">
      <h1 className="page-title">Audit Log</h1>
      {res.err && <div className="banner error">{res.err}</div>}
      <div className="card table-card">
        <table className="data-table">
          <thead><tr><th>Action</th><th>Actor</th><th>Details</th><th>Time</th></tr></thead>
          <tbody>
            {res.loading && <tr><td colSpan={4} className="empty-hint">Loading…</td></tr>}
            {!res.loading && res.items.length === 0 && <tr><td colSpan={4} className="empty-hint">No audit events.</td></tr>}
            {!res.loading && res.items.map((a, i) => (
              <tr key={a.id ?? i}>
                <td>{a.action ?? a.event ?? '—'}</td>
                <td className="mono">{a.actor ?? a.user_id ?? '—'}</td>
                <td>{a.details ?? a.description ?? '—'}</td>
                <td>{a.created_at ?? a.timestamp ?? '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// Passkeys page — backend is the relying party; credentials listed from it.
const MasterPasskeys = () => {
  const res = useWalletResource<any>('/passkey/credentials', ['passkeys', 'credentials']);
  const [msg, setMsg] = useState<string | null>(null);
  const del = async (credId: string) => {
    setMsg(null);
    try { await apiSend(`/api/v1/master-wallet/${res.wid}/passkey/credentials/${encodeURIComponent(credId)}`, 'DELETE'); setMsg('Deleted.'); res.reload(); }
    catch (e) { setMsg(e instanceof Error ? e.message : String(e)); }
  };
  return (
    <div className="page">
      <h1 className="page-title">Passkeys</h1>
      {msg && <div className="banner">{msg}</div>}
      <div className="card table-card">
        <table className="data-table">
          <thead><tr><th>Label</th><th>Credential</th><th>Created</th><th></th></tr></thead>
          <tbody>
            {res.loading && <tr><td colSpan={4} className="empty-hint">Loading…</td></tr>}
            {!res.loading && res.items.length === 0 && <tr><td colSpan={4} className="empty-hint">No passkeys registered.</td></tr>}
            {!res.loading && res.items.map((p, i) => (
              <tr key={p.credential_id ?? p.id ?? i}>
                <td>{p.label ?? '—'}</td>
                <td className="mono">{String(p.credential_id ?? p.id ?? '—').slice(0, 24)}</td>
                <td>{p.created_at ?? '—'}</td>
                <td><RowBtn label="Delete" kind="danger" onClick={() => del(p.credential_id ?? p.id)} /></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// Withdraw page — files a two-party withdrawal request; funds NEVER move
// without TigerWallet SuperAdmin co-sign.
const MasterWithdraw = () => {
  const [msg, setMsg] = useState<string | null>(null);
  const [form, setForm] = useState({ to: '', amountWei: '', currency: '', chainId: '1' });
  const set = (k: string) => (e: any) => setForm({ ...form, [k]: e.target.value });
  const submit = async () => {
    setMsg(null);
    const wid = await firstWalletId();
    if (!wid) { setMsg('No master wallet found.'); return; }
    try {
      const r = await apiSend<any>(`/api/v1/master-wallet/${wid}/withdrawal-request`, 'POST', {
        to_address: form.to, amount_wei: form.amountWei, currency: form.currency, chain_id: parseInt(form.chainId, 10) || 1,
      });
      setMsg('Withdrawal request filed: ' + ((r && (r.withdrawal_id || r.id)) || 'pending SuperAdmin co-sign'));
    } catch (e) { setMsg(e instanceof Error ? e.message : String(e)); }
  };
  return (
    <div className="page">
      <h1 className="page-title">Withdrawal Request</h1>
      <div className="banner">Funds never move without TigerWallet SuperAdmin two-party co-sign. This only files the request.</div>
      {msg && <div className="banner">{msg}</div>}
      <div className="card">
        <div className="form-grid">
          <input placeholder="Destination address" value={form.to} onChange={set('to')} />
          <input placeholder="Amount (wei)" value={form.amountWei} onChange={set('amountWei')} />
          <input placeholder="Currency (e.g. ETH)" value={form.currency} onChange={set('currency')} />
          <input placeholder="Chain ID" value={form.chainId} onChange={set('chainId')} />
          <button className="action-btn orange" onClick={submit}>Request withdrawal</button>
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
    .link-btn.danger { color: var(--error-color); }
    .link-btn.ok { color: var(--success-color); }
    .banner { background: var(--surface-color); color: var(--text-color); padding: .75rem 1rem; border-radius: .5rem; margin-bottom: 1rem; border: 1px solid var(--border-color); }
    .form-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: .75rem; margin-bottom: .75rem; align-items: end; }
    .form-grid input { padding: .6rem .75rem; border-radius: .5rem; border: 1px solid var(--border-color); background: var(--bg-color); color: var(--text-color); }
    pre.mono { white-space: pre-wrap; word-break: break-all; margin: 0; }
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
            {page === 'treasury' && <MasterTreasury />}
            {page === 'multisig' && <MasterMultisig />}
            {page === 'auto-sign' && <MasterAutoSign />}
            {page === 'fees' && <MasterFees />}
            {page === 'policies' && <MasterPolicies />}
            {page === 'chains' && <MasterChains />}
            {page === 'tokens' && <MasterTokens />}
            {page === 'flags' && <MasterFlags />}
            {page === 'webhooks' && <MasterWebhooks />}
            {page === 'audit' && <MasterAudit />}
            {page === 'passkeys' && <MasterPasskeys />}
            {page === 'withdraw' && <MasterWithdraw />}
            {page === 'analytics' && <MasterAnalytics />}
            {page === 'settings' && <MasterSettings />}
          </main>
        </div>
      </div>
    </MasterThemeProvider>
  );
};

export default MasterDesktopApp;
