// TigerWallet Master — Web App Main Component
// Wires to the canonical backend (port 8450) for wallets/balances/transactions,
// theme via the ThemeProvider context (isDark ternaries), and the live WS feed.
import { useState, useEffect, useCallback, FormEvent } from 'react';
import { useTheme } from './index';
import {
  masterWalletAPI,
  getAuthToken,
  setAuthToken,
  clearAuthToken,
  ApiError,
  MasterWallet,
  SubWallet,
  Transaction,
  AutoSignRule,
  ChainConfig,
  BalanceResponse,
  RevenuePayoutResponse,
  WithdrawalRequestResponse,
} from './api';
import { webSocketService } from './services/webSocketService';
import {
  PoliciesPage,
  FeesPage,
  NotificationsPage,
  WebhooksPage,
  AuditPage,
  MultisigPage,
  ChainsPage,
  TokensPage,
  FeatureFlagsPage,
  PasskeysPage,
} from './pages';

type Page =
  | 'dashboard'
  | 'wallets'
  | 'transactions'
  | 'treasury'
  | 'auto-sign'
  | 'users'
  | 'analytics'
  | 'policies'
  | 'fees'
  | 'notifications'
  | 'webhooks'
  | 'audit'
  | 'multisig'
  | 'chains'
  | 'tokens'
  | 'feature-flags'
  | 'passkeys'
  | 'settings';

interface AuthForm {
  email: string;
  password: string;
  name: string;
}

const shortHash = (h: string | undefined, len = 10): string => {
  if (!h) return '—';
  return h.length > len + 3 ? `${h.slice(0, len)}…${h.slice(-4)}` : h;
};

const formatBalance = (b: string | undefined): string => {
  if (b === undefined || b === null || b === '') return '0';
  const n = Number(b);
  return Number.isFinite(n) ? n.toLocaleString(undefined, { maximumFractionDigits: 6 }) : b;
};

// ---------------- Auth gate ----------------

const AuthGate = ({ onAuthed }: { onAuthed: () => void }) => {
  const { isDark } = useTheme();
  const [mode, setMode] = useState<'login' | 'register'>('login');
  const [form, setForm] = useState<AuthForm>({ email: '', password: '', name: '' });
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    setBusy(true);
    try {
      if (mode === 'login') {
        await masterWalletAPI.login(form.email, form.password);
      } else {
        const res = await masterWalletAPI.register(form.email, form.password, form.name);
        setAuthToken(res.token);
      }
      onAuthed();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const card = isDark ? 'bg-gray-800 border-gray-700' : 'bg-white border-gray-200';
  const input = isDark
    ? 'bg-gray-700 border-gray-600 text-white placeholder-gray-400'
    : 'bg-gray-50 border-gray-300 text-gray-900 placeholder-gray-400';

  return (
    <div className={`min-h-screen flex items-center justify-center ${isDark ? 'bg-gray-900' : 'bg-gray-100'}`}>
      <div className={`w-full max-w-md p-8 rounded-2xl border shadow-lg ${card}`}>
        <div className="flex items-center space-x-3 mb-6">
          <span className="text-3xl">🏦</span>
          <div>
            <h1 className={`text-xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>MasterWallet</h1>
            <p className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Enterprise Treasury</p>
          </div>
        </div>

        <div className="flex mb-6 rounded-lg overflow-hidden border border-gray-300">
          <button
            type="button"
            onClick={() => setMode('login')}
            className={`flex-1 py-2 text-sm font-medium ${mode === 'login' ? 'bg-blue-600 text-white' : isDark ? 'bg-gray-700 text-gray-300' : 'bg-gray-100 text-gray-600'}`}
          >
            Login
          </button>
          <button
            type="button"
            onClick={() => setMode('register')}
            className={`flex-1 py-2 text-sm font-medium ${mode === 'register' ? 'bg-blue-600 text-white' : isDark ? 'bg-gray-700 text-gray-300' : 'bg-gray-100 text-gray-600'}`}
          >
            Register
          </button>
        </div>

        <form onSubmit={submit} className="space-y-4">
          {mode === 'register' && (
            <input
              type="text"
              required
              placeholder="Name"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              className={`w-full px-4 py-2 border rounded-lg focus:outline-none focus:border-blue-500 ${input}`}
            />
          )}
          <input
            type="email"
            required
            placeholder="Email"
            value={form.email}
            onChange={(e) => setForm({ ...form, email: e.target.value })}
            className={`w-full px-4 py-2 border rounded-lg focus:outline-none focus:border-blue-500 ${input}`}
          />
          <input
            type="password"
            required
            minLength={mode === 'register' ? 8 : 1}
            placeholder="Password"
            value={form.password}
            onChange={(e) => setForm({ ...form, password: e.target.value })}
            className={`w-full px-4 py-2 border rounded-lg focus:outline-none focus:border-blue-500 ${input}`}
          />
          {error && <div className="text-sm text-red-500">{error}</div>}
          <button
            type="submit"
            disabled={busy}
            className="w-full py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
          >
            {busy ? 'Working…' : mode === 'login' ? 'Login' : 'Create account'}
          </button>
        </form>
      </div>
    </div>
  );
};

// ---------------- Sidebar ----------------

interface SidebarProps {
  currentPage: Page;
  setCurrentPage: (p: Page) => void;
  isDark: boolean;
  masterAddress?: string;
  onLogout: () => void;
}

const Sidebar = ({ currentPage, setCurrentPage, isDark, masterAddress, onLogout }: SidebarProps) => {
  const menuItems: { id: Page; label: string; icon: string }[] = [
    { id: 'dashboard', label: 'Dashboard', icon: '📊' },
    { id: 'wallets', label: 'Sub-Wallets', icon: '💼' },
    { id: 'transactions', label: 'Transactions', icon: '📜' },
    { id: 'treasury', label: 'Treasury', icon: '🏛️' },
    { id: 'auto-sign', label: 'Auto Sign', icon: '🔑' },
    { id: 'users', label: 'Users', icon: '👥' },
    { id: 'analytics', label: 'Analytics', icon: '📈' },
    { id: 'policies', label: 'Policies', icon: '🛡️' },
    { id: 'fees', label: 'Fees', icon: '💸' },
    { id: 'notifications', label: 'Notifications', icon: '🔔' },
    { id: 'webhooks', label: 'Webhooks', icon: '🪝' },
    { id: 'audit', label: 'Audit', icon: '📋' },
    { id: 'multisig', label: 'Multisig', icon: '🔏' },
    { id: 'chains', label: 'Chains', icon: '⛓️' },
    { id: 'tokens', label: 'Tokens', icon: '🪙' },
    { id: 'feature-flags', label: 'Feature Flags', icon: '🏁' },
    { id: 'passkeys', label: 'Passkeys', icon: '🔐' },
    { id: 'settings', label: 'Settings', icon: '⚙️' },
  ];

  const surface = isDark ? 'bg-gray-900 border-gray-800' : 'bg-white border-gray-200';
  const active = 'bg-blue-600 text-white';
  const idle = isDark ? 'text-gray-400 hover:bg-gray-800 hover:text-white' : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900';

  return (
    <aside className={`w-64 ${surface} border-r flex flex-col min-h-screen`}>
      <div className={`p-6 border-b ${isDark ? 'border-gray-800' : 'border-gray-200'}`}>
        <div className="flex items-center space-x-3">
          <span className="text-3xl">🏦</span>
          <div>
            <h1 className={`text-xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>MasterWallet</h1>
            <p className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Enterprise</p>
          </div>
        </div>
      </div>

      <nav className="flex-1 p-4">
        {menuItems.map((item) => (
          <button
            key={item.id}
            onClick={() => setCurrentPage(item.id)}
            className={`w-full flex items-center space-x-3 px-4 py-3 rounded-lg mb-2 transition-colors ${
              currentPage === item.id ? active : idle
            }`}
          >
            <span>{item.icon}</span>
            <span>{item.label}</span>
          </button>
        ))}
      </nav>

      <div className={`p-4 border-t ${isDark ? 'border-gray-800' : 'border-gray-200'}`}>
        <div className={`rounded-lg p-3 ${isDark ? 'bg-gray-800' : 'bg-gray-100'}`}>
          <div className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Master Address</div>
          <div className={`font-mono text-sm ${isDark ? 'text-white' : 'text-gray-900'}`}>{shortHash(masterAddress, 10)}</div>
        </div>
        <button
          onClick={onLogout}
          className={`w-full mt-3 py-2 rounded-lg text-sm ${isDark ? 'bg-gray-800 text-gray-300 hover:bg-gray-700' : 'bg-gray-100 text-gray-700 hover:bg-gray-200'}`}
        >
          Logout
        </button>
      </div>
    </aside>
  );
};

// ---------------- Header ----------------

const Header = ({ isDark, onCreate }: { isDark: boolean; onCreate: () => void }) => {
  return (
    <header className={`h-16 ${isDark ? 'bg-gray-900 border-gray-800' : 'bg-white border-gray-200'} border-b flex items-center justify-between px-6`}>
      <div className="flex items-center space-x-4">
        <input
          type="text"
          placeholder="Search wallets, users, transactions..."
          className={`px-4 py-2 border rounded-lg text-sm w-96 focus:outline-none focus:border-blue-500 ${
            isDark ? 'bg-gray-800 border-gray-700 text-white placeholder-gray-500' : 'bg-gray-50 border-gray-300 text-gray-900 placeholder-gray-400'
          }`}
        />
      </div>

      <div className="flex items-center space-x-4">
        <button
          onClick={onCreate}
          className="px-4 py-2 bg-blue-600 rounded-lg hover:bg-blue-700 text-sm text-white"
        >
          + Create Wallet
        </button>
      </div>
    </header>
  );
};

// ---------------- Dashboard ----------------

interface DashboardProps {
  isDark: boolean;
  wallets: MasterWallet[];
  transactions: Transaction[];
  balances: Record<string, BalanceResponse>;
  loading: boolean;
  error: string | null;
}

const Dashboard = ({ isDark, wallets, transactions, balances, loading, error }: DashboardProps) => {
  const totalVolume = transactions
    .filter((t) => t.status === 'confirmed')
    .reduce((sum, t) => sum + (Number(t.amount) || 0), 0);

  const pendingTx = transactions.filter((t) => t.status === 'pending').length;
  const totalBalance = Object.values(balances).reduce((s, b) => s + (Number(b.balance) || 0), 0);

  const stats = [
    { label: '💼 Total Wallets', value: String(wallets.length) },
    { label: '💰 Total Balance', value: formatBalance(String(totalBalance)) },
    { label: '📜 Transactions', value: String(transactions.length) },
    { label: '⏳ Pending Tx', value: String(pendingTx), color: 'text-orange-500' },
    { label: '📈 Confirmed Volume', value: formatBalance(String(totalVolume)), color: 'text-green-500' },
  ];

  const card = isDark ? 'bg-gray-800' : 'bg-white border border-gray-200';
  const subtext = isDark ? 'text-gray-400' : 'text-gray-500';
  const heading = isDark ? 'text-white' : 'text-gray-900';
  const row = isDark ? 'bg-gray-700' : 'bg-gray-50';

  return (
    <div className="space-y-6">
      <h1 className={`text-2xl font-bold ${heading}`}>Dashboard</h1>

      {error && <div className="p-3 rounded-lg bg-red-500/10 text-red-500 text-sm">{error}</div>}

      <div className="grid grid-cols-5 gap-4">
        {stats.map((s) => (
          <div key={s.label} className={`${card} rounded-xl p-5`}>
            <div className={`${subtext} mb-2`}>{s.label}</div>
            <div className={`text-3xl font-bold ${s.color ?? heading}`}>{loading ? '…' : s.value}</div>
          </div>
        ))}
      </div>

      <div className="grid grid-cols-2 gap-6">
        <div className={`${card} rounded-xl p-6`}>
          <h2 className={`text-lg font-semibold mb-4 ${heading}`}>Wallets</h2>
          {loading ? (
            <div className={subtext}>Loading…</div>
          ) : wallets.length === 0 ? (
            <div className={subtext}>No wallets yet. Create one to get started.</div>
          ) : (
            <div className="space-y-3">
              {wallets.map((w) => (
                <div key={w.id} className={`flex items-center justify-between p-3 ${row} rounded-lg`}>
                  <div>
                    <div className={`font-medium ${heading}`}>{w.name}</div>
                    <div className={`text-xs font-mono ${subtext}`}>{shortHash(w.address, 14)}</div>
                  </div>
                  <div className={`text-right font-mono ${heading}`}>
                    {balances[w.id] ? `${formatBalance(balances[w.id].balance)} ${balances[w.id].symbol ?? ''}` : '—'}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className={`${card} rounded-xl p-6`}>
          <h2 className={`text-lg font-semibold mb-4 ${heading}`}>Recent Transactions</h2>
          {loading ? (
            <div className={subtext}>Loading…</div>
          ) : transactions.length === 0 ? (
            <div className={subtext}>No transactions yet.</div>
          ) : (
            <div className="space-y-3">
              {transactions.slice(0, 6).map((tx) => (
                <div key={tx.id} className={`flex items-center justify-between p-3 ${row} rounded-lg`}>
                  <div className="flex items-center space-x-3">
                    <span className="text-xl">📤</span>
                    <div>
                      <div className={`font-medium ${heading}`}>{tx.tx_type ?? 'transfer'}</div>
                      <div className={`text-xs font-mono ${subtext}`}>{shortHash(tx.tx_hash, 14)}</div>
                    </div>
                  </div>
                  <div className="text-right">
                    <div className={`font-bold ${heading}`}>{formatBalance(tx.amount)}</div>
                    <div className={`text-xs ${tx.status === 'confirmed' ? 'text-green-500' : tx.status === 'pending' ? 'text-orange-500' : 'text-red-500'}`}>{tx.status}</div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

// ---------------- Wallets page ----------------

interface WalletsProps {
  isDark: boolean;
  masterId: string | undefined;
  wallets: SubWallet[];
  loading: boolean;
  error: string | null;
  onRefresh: () => void;
}

const Wallets = ({ isDark, masterId, wallets, loading, error, onRefresh }: WalletsProps) => {
  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [chainId, setChainId] = useState(1);
  const [chains, setChains] = useState<ChainConfig[]>([]);
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  useEffect(() => {
    masterWalletAPI.getSupportedChains().then(setChains).catch(() => setChains([]));
  }, []);

  const create = async (e: FormEvent) => {
    e.preventDefault();
    if (!masterId) return;
    setBusy(true);
    setMsg(null);
    try {
      await masterWalletAPI.createSubWallet(masterId, name, password, chainId);
      setName('');
      setPassword('');
      onRefresh();
      setMsg('Sub-wallet created.');
    } catch (err) {
      setMsg(err instanceof ApiError ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const card = isDark ? 'bg-gray-800' : 'bg-white border border-gray-200';
  const thead = isDark ? 'bg-gray-700 text-white' : 'bg-gray-100 text-gray-700';
  const row = isDark ? 'border-gray-700 hover:bg-gray-750' : 'border-gray-200 hover:bg-gray-50';
  const heading = isDark ? 'text-white' : 'text-gray-900';
  const subtext = isDark ? 'text-gray-400' : 'text-gray-500';
  const input = isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-gray-50 border-gray-300 text-gray-900';

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className={`text-2xl font-bold ${heading}`}>Sub-Wallets</h1>
        <button onClick={onRefresh} className={`px-4 py-2 rounded-lg text-sm ${isDark ? 'bg-gray-700 text-white' : 'bg-gray-100 text-gray-700'}`}>Refresh</button>
      </div>

      {!masterId && <div className="p-3 rounded-lg bg-orange-500/10 text-orange-500 text-sm">Create a master wallet first to manage sub-wallets.</div>}
      {error && <div className="p-3 rounded-lg bg-red-500/10 text-red-500 text-sm">{error}</div>}

      {masterId && (
        <form onSubmit={create} className={`${card} rounded-xl p-5 grid grid-cols-4 gap-3`}>
          <input required placeholder="Name" value={name} onChange={(e) => setName(e.target.value)} className={`px-3 py-2 border rounded-lg ${input}`} />
          <input required type="password" placeholder="Password" value={password} onChange={(e) => setPassword(e.target.value)} className={`px-3 py-2 border rounded-lg ${input}`} />
          <select value={chainId} onChange={(e) => setChainId(Number(e.target.value))} className={`px-3 py-2 border rounded-lg ${input}`}>
            {chains.map((c) => (
              <option key={c.chain_id} value={c.chain_id}>{c.name}</option>
            ))}
          </select>
          <button type="submit" disabled={busy} className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">
            {busy ? 'Creating…' : '+ Create'}
          </button>
          {msg && <div className="col-span-4 text-sm text-gray-500">{msg}</div>}
        </form>
      )}

      <div className={`${card} rounded-xl overflow-hidden`}>
        <table className="w-full">
          <thead className={thead}>
            <tr>
              <th className="px-6 py-3 text-left">Name</th>
              <th className="px-6 py-3 text-left">Address</th>
              <th className="px-6 py-3 text-left">Balance</th>
              <th className="px-6 py-3 text-left">Status</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr><td colSpan={4} className={`px-6 py-4 ${subtext}`}>Loading…</td></tr>
            ) : wallets.length === 0 ? (
              <tr><td colSpan={4} className={`px-6 py-4 ${subtext}`}>No sub-wallets.</td></tr>
            ) : (
              wallets.map((w) => (
                <tr key={w.id} className={`border-b ${row}`}>
                  <td className={`px-6 py-4 font-medium ${heading}`}>{w.name}</td>
                  <td className={`px-6 py-4 font-mono text-sm ${subtext}`}>{shortHash(w.address, 16)}</td>
                  <td className={`px-6 py-4 ${heading}`}>{w.balance ? formatBalance(w.balance) : '—'}</td>
                  <td className={`px-6 py-4 ${subtext}`}>{w.status ?? 'active'}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// ---------------- Transactions page ----------------

interface TransactionsProps {
  isDark: boolean;
  masterId: string | undefined;
  transactions: Transaction[];
  loading: boolean;
  error: string | null;
  onRefresh: () => void;
}

const Transactions = ({ isDark, masterId, transactions, loading, error, onRefresh }: TransactionsProps) => {
  const [to, setTo] = useState('');
  const [amount, setAmount] = useState('');
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  const send = async (e: FormEvent) => {
    e.preventDefault();
    if (!masterId) return;
    setBusy(true);
    setMsg(null);
    try {
      const res = await masterWalletAPI.signTransaction(masterId, { to, amount, password });
      setMsg(`Broadcast: ${res.transaction_hash}`);
      setTo(''); setAmount(''); setPassword('');
      onRefresh();
    } catch (err) {
      setMsg(err instanceof ApiError ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const card = isDark ? 'bg-gray-800' : 'bg-white border border-gray-200';
  const thead = isDark ? 'bg-gray-700 text-white' : 'bg-gray-100 text-gray-700';
  const row = isDark ? 'border-gray-700 hover:bg-gray-750' : 'border-gray-200 hover:bg-gray-50';
  const heading = isDark ? 'text-white' : 'text-gray-900';
  const subtext = isDark ? 'text-gray-400' : 'text-gray-500';
  const input = isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-gray-50 border-gray-300 text-gray-900';

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className={`text-2xl font-bold ${heading}`}>Transactions</h1>
        <button onClick={onRefresh} className={`px-4 py-2 rounded-lg text-sm ${isDark ? 'bg-gray-700 text-white' : 'bg-gray-100 text-gray-700'}`}>Refresh</button>
      </div>

      {error && <div className="p-3 rounded-lg bg-red-500/10 text-red-500 text-sm">{error}</div>}

      {masterId && (
        <form onSubmit={send} className={`${card} rounded-xl p-5 grid grid-cols-4 gap-3`}>
          <input required placeholder="Recipient 0x…" value={to} onChange={(e) => setTo(e.target.value)} className={`px-3 py-2 border rounded-lg ${input}`} />
          <input required placeholder="Amount" value={amount} onChange={(e) => setAmount(e.target.value)} className={`px-3 py-2 border rounded-lg ${input}`} />
          <input required type="password" placeholder="Password" value={password} onChange={(e) => setPassword(e.target.value)} className={`px-3 py-2 border rounded-lg ${input}`} />
          <button type="submit" disabled={busy} className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">
            {busy ? 'Signing…' : 'Sign & Send'}
          </button>
          {msg && <div className="col-span-4 text-sm text-gray-500 break-all">{msg}</div>}
        </form>
      )}

      <div className={`${card} rounded-xl overflow-hidden`}>
        <table className="w-full">
          <thead className={thead}>
            <tr>
              <th className="px-6 py-3 text-left">Hash</th>
              <th className="px-6 py-3 text-left">Type</th>
              <th className="px-6 py-3 text-left">Amount</th>
              <th className="px-6 py-3 text-left">Status</th>
              <th className="px-6 py-3 text-left">Created</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr><td colSpan={5} className={`px-6 py-4 ${subtext}`}>Loading…</td></tr>
            ) : transactions.length === 0 ? (
              <tr><td colSpan={5} className={`px-6 py-4 ${subtext}`}>No transactions.</td></tr>
            ) : (
              transactions.map((tx) => (
                <tr key={tx.id} className={`border-b ${row}`}>
                  <td className={`px-6 py-4 font-mono text-sm ${subtext}`}>{shortHash(tx.tx_hash, 14)}</td>
                  <td className={`px-6 py-4 ${heading}`}>{tx.tx_type ?? 'transfer'}</td>
                  <td className={`px-6 py-4 ${heading}`}>{formatBalance(tx.amount)}</td>
                  <td className={`px-6 py-4 ${tx.status === 'confirmed' ? 'text-green-500' : tx.status === 'pending' ? 'text-orange-500' : 'text-red-500'}`}>{tx.status}</td>
                  <td className={`px-6 py-4 ${subtext}`}>{tx.created_at ? new Date(tx.created_at).toLocaleString() : '—'}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// ---------------- Treasury page (two-party revenue gate) ----------------

interface TreasuryPageProps {
  isDark: boolean;
  masterId: string | undefined;
}

const TreasuryPage = ({ isDark, masterId }: TreasuryPageProps) => {
  // Withdrawal Request form state (WL-side half of the two-party gate)
  const [wrTo, setWrTo] = useState('');
  const [wrAmount, setWrAmount] = useState('');
  const [wrCurrency, setWrCurrency] = useState('ETH');
  const [wrChainId, setWrChainId] = useState<number>(1);
  const [wrBusy, setWrBusy] = useState(false);
  const [wrError, setWrError] = useState<string | null>(null);
  const [wrResult, setWrResult] = useState<WithdrawalRequestResponse | null>(null);

  // Revenue Payout form state (requires SuperAdmin co-signed withdrawal_id)
  const [rpTo, setRpTo] = useState('');
  const [rpAmount, setRpAmount] = useState('');
  const [rpPassword, setRpPassword] = useState('');
  const [rpGasLimit, setRpGasLimit] = useState('');
  const [rpWithdrawalId, setRpWithdrawalId] = useState('');
  const [rpBusy, setRpBusy] = useState(false);
  const [rpError, setRpError] = useState<string | null>(null);
  const [rpResult, setRpResult] = useState<RevenuePayoutResponse | null>(null);

  const [chains, setChains] = useState<ChainConfig[]>([]);

  useEffect(() => {
    masterWalletAPI.getSupportedChains().then(setChains).catch(() => setChains([]));
  }, []);

  const submitWithdrawal = async (e: FormEvent) => {
    e.preventDefault();
    if (!masterId) return;
    setWrBusy(true);
    setWrError(null);
    setWrResult(null);
    try {
      const res = await masterWalletAPI.requestWithdrawal(masterId, {
        to_address: wrTo,
        amount_wei: wrAmount,
        currency: wrCurrency || undefined,
        chain_id: wrChainId || undefined,
      });
      setWrResult(res);
      setWrTo('');
      setWrAmount('');
    } catch (err) {
      setWrError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setWrBusy(false);
    }
  };

  const submitPayout = async (e: FormEvent) => {
    e.preventDefault();
    if (!masterId) return;
    setRpBusy(true);
    setRpError(null);
    setRpResult(null);
    try {
      const gasLimitNum = rpGasLimit ? Number(rpGasLimit) : undefined;
      const res = await masterWalletAPI.revenuePayout(masterId, {
        to: rpTo,
        amount: rpAmount,
        password: rpPassword,
        gas_limit: gasLimitNum && Number.isFinite(gasLimitNum) ? gasLimitNum : undefined,
        withdrawal_id: rpWithdrawalId,
      });
      setRpResult(res);
      setRpTo('');
      setRpAmount('');
      setRpPassword('');
      setRpGasLimit('');
      setRpWithdrawalId('');
    } catch (err) {
      setRpError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setRpBusy(false);
    }
  };

  const card = isDark ? 'bg-gray-800' : 'bg-white border border-gray-200';
  const heading = isDark ? 'text-white' : 'text-gray-900';
  const subtext = isDark ? 'text-gray-400' : 'text-gray-500';
  const input = isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-gray-50 border-gray-300 text-gray-900';
  const label = isDark ? 'text-gray-300' : 'text-gray-700';

  const useWithdrawalId = (id: string) => {
    setRpWithdrawalId(id);
    setRpError(null);
  };

  return (
    <div className="space-y-6">
      <h1 className={`text-2xl font-bold ${heading}`}>Treasury</h1>

      <div className={`${card} rounded-xl p-4`}>
        <div className={`flex items-start space-x-3 ${subtext} text-sm`}>
          <span className="text-lg">🔐</span>
          <p>
            Revenue payout requires <span className={`font-semibold ${heading}`}>SuperAdmin co-signature (two-party gate)</span>.
            Submit a withdrawal request first, then have SuperAdmin approve it, then enter the{' '}
            <span className={`font-mono ${heading}`}>withdrawal_id</span> in the Revenue Payout form below.
            Revenue never moves without both approvals.
          </p>
        </div>
      </div>

      {!masterId && (
        <div className="p-3 rounded-lg bg-orange-500/10 text-orange-500 text-sm">
          Create a master wallet first to use the treasury gate.
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Withdrawal Request form */}
        <div className={`${card} rounded-xl p-6 space-y-4`}>
          <div>
            <h2 className={`text-lg font-semibold ${heading}`}>1. Withdrawal Request</h2>
            <p className={`text-sm ${subtext}`}>Creates a two-party withdrawal request (WL-side half). SuperAdmin must co-approve before payout.</p>
          </div>
          <form onSubmit={submitWithdrawal} className="space-y-3">
            <div>
              <label className={`block text-xs mb-1 ${label}`}>Recipient address</label>
              <input required placeholder="0x…" value={wrTo} onChange={(e) => setWrTo(e.target.value)} className={`w-full px-3 py-2 border rounded-lg font-mono text-sm ${input}`} />
            </div>
            <div>
              <label className={`block text-xs mb-1 ${label}`}>Amount (wei)</label>
              <input required placeholder="e.g. 1000000000000000000" value={wrAmount} onChange={(e) => setWrAmount(e.target.value)} className={`w-full px-3 py-2 border rounded-lg font-mono text-sm ${input}`} />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className={`block text-xs mb-1 ${label}`}>Currency</label>
                <input placeholder="ETH" value={wrCurrency} onChange={(e) => setWrCurrency(e.target.value)} className={`w-full px-3 py-2 border rounded-lg ${input}`} />
              </div>
              <div>
                <label className={`block text-xs mb-1 ${label}`}>Chain ID</label>
                <select value={wrChainId} onChange={(e) => setWrChainId(Number(e.target.value))} className={`w-full px-3 py-2 border rounded-lg ${input}`}>
                  {chains.map((c) => (
                    <option key={c.chain_id} value={c.chain_id}>{c.name} ({c.chain_id})</option>
                  ))}
                </select>
              </div>
            </div>
            {wrError && <div className="text-sm text-red-500">{wrError}</div>}
            {wrResult && (
              <div className="p-3 rounded-lg bg-green-500/10 text-green-500 text-sm space-y-1">
                <div>✅ Withdrawal request created — pending SuperAdmin approval.</div>
                <div className="font-mono break-all">withdrawal_id: {wrResult.withdrawal_id}</div>
                <div>Status: {wrResult.status}</div>
                <button type="button" onClick={() => useWithdrawalId(wrResult.withdrawal_id)} className="mt-1 text-xs underline">
                  Use this withdrawal_id in the Revenue Payout form →
                </button>
              </div>
            )}
            <button type="submit" disabled={wrBusy || !masterId} className="w-full px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">
              {wrBusy ? 'Submitting…' : 'Submit Withdrawal Request'}
            </button>
          </form>
        </div>

        {/* Revenue Payout form */}
        <div className={`${card} rounded-xl p-6 space-y-4`}>
          <div>
            <h2 className={`text-lg font-semibold ${heading}`}>2. Revenue Payout</h2>
            <p className={`text-sm ${subtext}`}>Broadcasts revenue funds. Requires a SuperAdmin-approved withdrawal_id; the gate is checked fail-closed.</p>
          </div>
          <form onSubmit={submitPayout} className="space-y-3">
            <div>
              <label className={`block text-xs mb-1 ${label}`}>Recipient address</label>
              <input required placeholder="0x…" value={rpTo} onChange={(e) => setRpTo(e.target.value)} className={`w-full px-3 py-2 border rounded-lg font-mono text-sm ${input}`} />
            </div>
            <div>
              <label className={`block text-xs mb-1 ${label}`}>Amount</label>
              <input required placeholder="e.g. 1.0" value={rpAmount} onChange={(e) => setRpAmount(e.target.value)} className={`w-full px-3 py-2 border rounded-lg ${input}`} />
            </div>
            <div>
              <label className={`block text-xs mb-1 ${label}`}>Encryption password</label>
              <input required type="password" placeholder="Master wallet password" value={rpPassword} onChange={(e) => setRpPassword(e.target.value)} className={`w-full px-3 py-2 border rounded-lg ${input}`} />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className={`block text-xs mb-1 ${label}`}>Gas limit (optional)</label>
                <input placeholder="21000" value={rpGasLimit} onChange={(e) => setRpGasLimit(e.target.value)} className={`w-full px-3 py-2 border rounded-lg font-mono text-sm ${input}`} />
              </div>
              <div>
                <label className={`block text-xs mb-1 ${label}`}>Withdrawal ID *</label>
                <input required placeholder="from approved request" value={rpWithdrawalId} onChange={(e) => setRpWithdrawalId(e.target.value)} className={`w-full px-3 py-2 border rounded-lg font-mono text-sm ${input}`} />
              </div>
            </div>
            {rpError && <div className="text-sm text-red-500">{rpError}</div>}
            {rpResult && (
              <div className="p-3 rounded-lg bg-green-500/10 text-green-500 text-sm space-y-1">
                <div>✅ Revenue payout broadcast.</div>
                <div className="font-mono break-all">tx_hash: {rpResult.transaction_hash}</div>
                <div>Status: {rpResult.status}{rpResult.from ? ` • from: ${rpResult.from}` : ''}{rpResult.chain_id ? ` • chain: ${rpResult.chain_id}` : ''}</div>
              </div>
            )}
            <button type="submit" disabled={rpBusy || !masterId} className="w-full px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">
              {rpBusy ? 'Broadcasting…' : 'Submit Revenue Payout'}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
};

// ---------------- AutoSign page ----------------

interface AutoSignProps {
  isDark: boolean;
  masterId: string | undefined;
  rules: AutoSignRule[];
  loading: boolean;
  error: string | null;
  onRefresh: () => void;
}

const AutoSignPage = ({ isDark, masterId, rules, loading, error, onRefresh }: AutoSignProps) => {
  const [name, setName] = useState('');
  const [ruleType, setRuleType] = useState('max_amount');
  const [maxAmount, setMaxAmount] = useState('');
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  const create = async (e: FormEvent) => {
    e.preventDefault();
    if (!masterId) return;
    setBusy(true);
    setMsg(null);
    try {
      await masterWalletAPI.createAutoSignRule(masterId, {
        name,
        rule_type: ruleType,
        max_amount: maxAmount,
        is_active: true,
      });
      setName(''); setMaxAmount('');
      onRefresh();
      setMsg('Rule created.');
    } catch (err) {
      setMsg(err instanceof ApiError ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const remove = async (rid: string) => {
    if (!masterId) return;
    try {
      await masterWalletAPI.deleteAutoSignRule(masterId, rid);
      onRefresh();
    } catch (err) {
      setMsg(err instanceof ApiError ? err.message : String(err));
    }
  };

  const card = isDark ? 'bg-gray-800' : 'bg-white border border-gray-200';
  const heading = isDark ? 'text-white' : 'text-gray-900';
  const subtext = isDark ? 'text-gray-400' : 'text-gray-500';
  const input = isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-gray-50 border-gray-300 text-gray-900';

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className={`text-2xl font-bold ${heading}`}>Auto-Sign Rules</h1>
        <button onClick={onRefresh} className={`px-4 py-2 rounded-lg text-sm ${isDark ? 'bg-gray-700 text-white' : 'bg-gray-100 text-gray-700'}`}>Refresh</button>
      </div>

      {error && <div className="p-3 rounded-lg bg-red-500/10 text-red-500 text-sm">{error}</div>}

      {masterId && (
        <form onSubmit={create} className={`${card} rounded-xl p-5 grid grid-cols-4 gap-3`}>
          <input required placeholder="Name" value={name} onChange={(e) => setName(e.target.value)} className={`px-3 py-2 border rounded-lg ${input}`} />
          <select value={ruleType} onChange={(e) => setRuleType(e.target.value)} className={`px-3 py-2 border rounded-lg ${input}`}>
            <option value="max_amount">max_amount</option>
            <option value="whitelist">whitelist</option>
            <option value="schedule">schedule</option>
          </select>
          <input placeholder="Max amount" value={maxAmount} onChange={(e) => setMaxAmount(e.target.value)} className={`px-3 py-2 border rounded-lg ${input}`} />
          <button type="submit" disabled={busy} className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">
            {busy ? 'Creating…' : '+ Create Rule'}
          </button>
          {msg && <div className="col-span-4 text-sm text-gray-500">{msg}</div>}
        </form>
      )}

      <div className="grid grid-cols-1 gap-4">
        {loading ? (
          <div className={subtext}>Loading…</div>
        ) : rules.length === 0 ? (
          <div className={subtext}>No auto-sign rules.</div>
        ) : (
          rules.map((rule) => (
            <div key={rule.id} className={`${card} rounded-xl p-5 flex items-center justify-between`}>
              <div>
                <div className={`font-semibold text-lg ${heading}`}>{rule.name}</div>
                <div className={subtext}>Type: {rule.rule_type}{rule.max_amount ? ` • Max: ${rule.max_amount}` : ''}</div>
              </div>
              <div className="flex items-center space-x-4">
                <span className={`px-3 py-1 rounded ${rule.is_active ? 'bg-green-500 text-white' : isDark ? 'bg-gray-600 text-gray-300' : 'bg-gray-300 text-gray-700'}`}>
                  {rule.is_active ? 'Enabled' : 'Disabled'}
                </span>
                <button onClick={() => remove(rule.id)} className="text-red-500 hover:underline">Delete</button>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
};

// ---------------- Users page ----------------

interface UsersProps {
  isDark: boolean;
  masterId: string | undefined;
  users: { id: string; email: string; name: string; role: string; is_active?: boolean }[];
  loading: boolean;
  error: string | null;
  onRefresh: () => void;
}

const UsersPage = ({ isDark, masterId, users, loading, error, onRefresh }: UsersProps) => {
  const [email, setEmail] = useState('');
  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [role, setRole] = useState('user');
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  const create = async (e: FormEvent) => {
    e.preventDefault();
    if (!masterId) return;
    setBusy(true); setMsg(null);
    try {
      await masterWalletAPI.createUser(masterId, { email, password, name, role });
      setEmail(''); setName(''); setPassword(''); setRole('user');
      onRefresh();
      setMsg('User created.');
    } catch (err) {
      setMsg(err instanceof ApiError ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const remove = async (uid: string) => {
    if (!masterId) return;
    try {
      await masterWalletAPI.deleteUser(masterId, uid);
      onRefresh();
    } catch (err) {
      setMsg(err instanceof ApiError ? err.message : String(err));
    }
  };

  const card = isDark ? 'bg-gray-800' : 'bg-white border border-gray-200';
  const thead = isDark ? 'bg-gray-700 text-white' : 'bg-gray-100 text-gray-700';
  const row = isDark ? 'border-gray-700 hover:bg-gray-750' : 'border-gray-200 hover:bg-gray-50';
  const heading = isDark ? 'text-white' : 'text-gray-900';
  const subtext = isDark ? 'text-gray-400' : 'text-gray-500';
  const input = isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-gray-50 border-gray-300 text-gray-900';

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className={`text-2xl font-bold ${heading}`}>Users</h1>
        <button onClick={onRefresh} className={`px-4 py-2 rounded-lg text-sm ${isDark ? 'bg-gray-700 text-white' : 'bg-gray-100 text-gray-700'}`}>Refresh</button>
      </div>

      {error && <div className="p-3 rounded-lg bg-red-500/10 text-red-500 text-sm">{error}</div>}

      {masterId && (
        <form onSubmit={create} className={`${card} rounded-xl p-5 grid grid-cols-5 gap-3`}>
          <input required placeholder="Email" value={email} onChange={(e) => setEmail(e.target.value)} className={`px-3 py-2 border rounded-lg ${input}`} />
          <input placeholder="Name" value={name} onChange={(e) => setName(e.target.value)} className={`px-3 py-2 border rounded-lg ${input}`} />
          <input required type="password" placeholder="Password" value={password} onChange={(e) => setPassword(e.target.value)} className={`px-3 py-2 border rounded-lg ${input}`} />
          <select value={role} onChange={(e) => setRole(e.target.value)} className={`px-3 py-2 border rounded-lg ${input}`}>
            <option value="user">user</option>
            <option value="admin">admin</option>
            <option value="operator">operator</option>
          </select>
          <button type="submit" disabled={busy} className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">
            {busy ? 'Creating…' : '+ Add User'}
          </button>
          {msg && <div className="col-span-5 text-sm text-gray-500">{msg}</div>}
        </form>
      )}

      <div className={`${card} rounded-xl overflow-hidden`}>
        <table className="w-full">
          <thead className={thead}>
            <tr>
              <th className="px-6 py-3 text-left">Email</th>
              <th className="px-6 py-3 text-left">Name</th>
              <th className="px-6 py-3 text-left">Role</th>
              <th className="px-6 py-3 text-left">Status</th>
              <th className="px-6 py-3 text-left">Actions</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr><td colSpan={5} className={`px-6 py-4 ${subtext}`}>Loading…</td></tr>
            ) : users.length === 0 ? (
              <tr><td colSpan={5} className={`px-6 py-4 ${subtext}`}>No users.</td></tr>
            ) : (
              users.map((u) => (
                <tr key={u.id} className={`border-b ${row}`}>
                  <td className={`px-6 py-4 ${heading}`}>{u.email}</td>
                  <td className={`px-6 py-4 ${heading}`}>{u.name || '—'}</td>
                  <td className={`px-6 py-4 ${subtext}`}>{u.role}</td>
                  <td className={`px-6 py-4 ${u.is_active ? 'text-green-500' : 'text-gray-500'}`}>{u.is_active ? 'active' : 'inactive'}</td>
                  <td className="px-6 py-4"><button onClick={() => remove(u.id)} className="text-red-500 hover:underline">Delete</button></td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// ---------------- Analytics page ----------------

interface AnalyticsProps {
  isDark: boolean;
  masterId: string | undefined;
}

const AnalyticsPage = ({ isDark, masterId }: AnalyticsProps) => {
  const [volume, setVolume] = useState<{ total_volume: string; transaction_count: number } | null>(null);
  const [byStatus, setByStatus] = useState<Record<string, number>>({});
  const [wallets, setWallets] = useState<{ master_wallets: number; sub_wallets: number; users: number } | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      setError(null);
      try {
        const [v, t, w] = await Promise.all([
          masterId ? masterWalletAPI.getVolumeAnalytics(masterId) : Promise.resolve(null),
          masterId ? masterWalletAPI.getTransactionAnalytics(masterId) : Promise.resolve(null),
          masterId ? masterWalletAPI.getWalletAnalytics(masterId) : Promise.resolve(null),
        ]);
        if (cancelled) return;
        if (v) setVolume({ total_volume: v.total_volume, transaction_count: v.transaction_count });
        if (t) setByStatus(t.by_status);
        setWallets(w);
      } catch (err) {
        if (!cancelled) setError(err instanceof ApiError ? err.message : String(err));
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [masterId]);

  const card = isDark ? 'bg-gray-800' : 'bg-white border border-gray-200';
  const heading = isDark ? 'text-white' : 'text-gray-900';
  const subtext = isDark ? 'text-gray-400' : 'text-gray-500';

  return (
    <div className="space-y-6">
      <h1 className={`text-2xl font-bold ${heading}`}>Analytics</h1>
      {error && <div className="p-3 rounded-lg bg-red-500/10 text-red-500 text-sm">{error}</div>}
      {!masterId && <div className={subtext}>Create a master wallet to view its analytics.</div>}
      <div className="grid grid-cols-3 gap-4">
        <div className={`${card} rounded-xl p-5`}>
          <div className={`${subtext} mb-2`}>💰 Total Volume</div>
          <div className={`text-3xl font-bold ${heading}`}>{loading ? '…' : volume ? formatBalance(volume.total_volume) : '—'}</div>
        </div>
        <div className={`${card} rounded-xl p-5`}>
          <div className={`${subtext} mb-2`}>📜 Transaction Count</div>
          <div className={`text-3xl font-bold ${heading}`}>{loading ? '…' : volume ? String(volume.transaction_count) : '—'}</div>
        </div>
        <div className={`${card} rounded-xl p-5`}>
          <div className={`${subtext} mb-2`}>💼 Wallets</div>
          <div className={`text-3xl font-bold ${heading}`}>{loading ? '…' : wallets ? String(wallets.master_wallets) : '—'}</div>
        </div>
      </div>
      <div className={`${card} rounded-xl p-6`}>
        <h2 className={`text-lg font-semibold mb-4 ${heading}`}>Transactions by Status</h2>
        {Object.keys(byStatus).length === 0 ? (
          <div className={subtext}>{loading ? 'Loading…' : 'No data.'}</div>
        ) : (
          <div className="space-y-2">
            {Object.entries(byStatus).map(([status, count]) => (
              <div key={status} className="flex justify-between">
                <span className={subtext}>{status}</span>
                <span className={heading}>{count}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

// ---------------- Settings page ----------------

const SettingsPage = ({ isDark }: { isDark: boolean }) => {
  const { toggleTheme } = useTheme();
  const [apiUrl] = useState(masterWalletAPI.baseUrl);
  const [wsState, setWsState] = useState(webSocketService.connectionState);

  useEffect(() => {
    const handler = (s: typeof wsState) => setWsState(s);
    webSocketService.onStateChange(handler);
    return () => { webSocketService.offStateChange(handler); };
  }, []);

  const card = isDark ? 'bg-gray-800' : 'bg-white border border-gray-200';
  const row = isDark ? 'bg-gray-700' : 'bg-gray-50';
  const heading = isDark ? 'text-white' : 'text-gray-900';
  const subtext = isDark ? 'text-gray-400' : 'text-gray-500';
  const input = isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-gray-50 border-gray-300 text-gray-900';

  return (
    <div className="space-y-6">
      <h1 className={`text-2xl font-bold ${heading}`}>Settings</h1>

      <div className={`${card} rounded-xl p-6 space-y-4`}>
        <h2 className={`text-lg font-semibold ${heading}`}>Appearance</h2>
        <div className={`flex items-center justify-between p-4 ${row} rounded-lg`}>
          <span className={heading}>Dark Mode</span>
          <button
            onClick={toggleTheme}
            className={`w-14 h-7 rounded-full transition-colors ${isDark ? 'bg-blue-500' : 'bg-gray-400'}`}
          >
            <div className={`w-5 h-5 bg-white rounded-full transform transition-transform ${isDark ? 'translate-x-7' : 'translate-x-1'}`} />
          </button>
        </div>
      </div>

      <div className={`${card} rounded-xl p-6 space-y-4`}>
        <h2 className={`text-lg font-semibold ${heading}`}>Network</h2>
        <div className={`p-4 ${row} rounded-lg`}>
          <div className={subtext}>API Base URL</div>
          <div className={`font-mono text-sm ${heading}`}>{apiUrl}</div>
        </div>
        <input
          readOnly
          value={`${apiUrl.replace(/^http/, 'ws')}/ws`}
          className={`w-full px-4 py-3 border rounded-lg font-mono text-sm ${input}`}
        />
        <div className={`p-4 ${row} rounded-lg`}>
          <div className={subtext}>WebSocket State</div>
          <div className={heading}>{wsState}</div>
        </div>
      </div>

      <div className={`${card} rounded-xl p-6`}>
        <h2 className={`text-lg font-semibold mb-4 ${heading}`}>About</h2>
        <div className={`flex justify-between py-2`}>
          <span className={heading}>Version</span>
          <span className={subtext}>1.0.0</span>
        </div>
        <div className={`flex justify-between py-2`}>
          <span className={heading}>Backend</span>
          <span className={subtext}>{apiUrl}</span>
        </div>
      </div>
    </div>
  );
};

// ---------------- Create master wallet modal ----------------

interface CreateModalProps {
  isDark: boolean;
  onClose: () => void;
  onCreated: (walletId: string, mnemonic: string) => void;
}

const CreateWalletModal = ({ isDark, onClose, onCreated }: CreateModalProps) => {
  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [chainId, setChainId] = useState(1);
  const [chains, setChains] = useState<ChainConfig[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    masterWalletAPI.getSupportedChains().then(setChains).catch(() => setChains([]));
  }, []);

  const create = async (e: FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const res = (await masterWalletAPI.createMasterWallet(name, password, chainId)) as unknown as {
        wallet_id?: string;
        id?: string;
        mnemonic?: string;
      };
      const id = res.wallet_id ?? res.id ?? '';
      onCreated(id, res.mnemonic ?? '');
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const overlay = isDark ? 'bg-black/60' : 'bg-black/40';
  const card = isDark ? 'bg-gray-800 border-gray-700' : 'bg-white border-gray-200';
  const input = isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-gray-50 border-gray-300 text-gray-900';
  const heading = isDark ? 'text-white' : 'text-gray-900';

  return (
    <div className={`fixed inset-0 ${overlay} flex items-center justify-center z-50`}>
      <div className={`w-full max-w-md p-6 rounded-2xl border shadow-xl ${card}`}>
        <h2 className={`text-xl font-bold mb-4 ${heading}`}>Create Master Wallet</h2>
        <form onSubmit={create} className="space-y-4">
          <input required placeholder="Wallet name" value={name} onChange={(e) => setName(e.target.value)} className={`w-full px-4 py-2 border rounded-lg ${input}`} />
          <input required type="password" placeholder="Encryption password" value={password} onChange={(e) => setPassword(e.target.value)} className={`w-full px-4 py-2 border rounded-lg ${input}`} />
          <select value={chainId} onChange={(e) => setChainId(Number(e.target.value))} className={`w-full px-4 py-2 border rounded-lg ${input}`}>
            {chains.map((c) => (
              <option key={c.chain_id} value={c.chain_id}>{c.name}</option>
            ))}
          </select>
          {error && <div className="text-sm text-red-500">{error}</div>}
          <div className="flex justify-end space-x-3">
            <button type="button" onClick={onClose} className={`px-4 py-2 rounded-lg ${isDark ? 'bg-gray-700 text-gray-300' : 'bg-gray-100 text-gray-700'}`}>Cancel</button>
            <button type="submit" disabled={busy} className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">
              {busy ? 'Creating…' : 'Create'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

// ---------------- Main App ----------------

const App = () => {
  const { isDark } = useTheme();
  const [authed, setAuthed] = useState<boolean>(() => !!getAuthToken());
  const [currentPage, setCurrentPage] = useState<Page>('dashboard');
  const [wallets, setWallets] = useState<MasterWallet[]>([]);
  const [subWallets, setSubWallets] = useState<SubWallet[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [autoSignRules, setAutoSignRules] = useState<AutoSignRule[]>([]);
  const [users, setUsers] = useState<{ id: string; email: string; name: string; role: string; is_active?: boolean }[]>([]);
  const [balances, setBalances] = useState<Record<string, BalanceResponse>>({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [mnemonicNote, setMnemonicNote] = useState<string | null>(null);

  const activeWallet = wallets[0];
  const masterId = activeWallet?.id;

  const loadAll = useCallback(async () => {
    if (!getAuthToken()) {
      setAuthed(false);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const ws = await masterWalletAPI.getMasterWallets();
      setWallets(ws);
      if (ws.length > 0) {
        const id = ws[0].id;
        const [subs, txs, rules, usrs, ...balRes] = await Promise.all([
          masterWalletAPI.getSubWallets(id).catch(() => []),
          masterWalletAPI.getTransactions(id).catch(() => []),
          masterWalletAPI.getAutoSignRules(id).catch(() => []),
          masterWalletAPI.getUsers(id).catch(() => []),
          ...ws.map((w) => masterWalletAPI.getMasterWalletBalance(w.id).catch(() => null)),
        ]);
        setSubWallets(subs as SubWallet[]);
        setTransactions(txs as Transaction[]);
        setAutoSignRules(rules as AutoSignRule[]);
        setUsers(usrs as { id: string; email: string; name: string; role: string; is_active?: boolean }[]);
        const balMap: Record<string, BalanceResponse> = {};
        ws.forEach((w, i) => {
          const b = balRes[i];
          if (b) balMap[w.id] = b as BalanceResponse;
        });
        setBalances(balMap);
      } else {
        setSubWallets([]); setTransactions([]); setAutoSignRules([]); setUsers([]); setBalances({});
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, []);

  // Fetch wallets/balances/transactions on auth.
  useEffect(() => {
    if (authed) loadAll();
  }, [authed, loadAll]);

  // Connect the live WebSocket once we have a master wallet.
  useEffect(() => {
    if (!authed || !masterId) return;
    webSocketService.connect(masterId);
    const onBalance = () => { loadAll(); };
    webSocketService.onBalanceUpdate(onBalance);
    return () => {
      webSocketService.offStateChange(() => {});
    };
  }, [authed, masterId, loadAll]);

  const handleLogout = () => {
    webSocketService.disconnect();
    clearAuthToken();
    setAuthed(false);
    setWallets([]); setSubWallets([]); setTransactions([]); setAutoSignRules([]); setUsers([]); setBalances({});
  };

  const handleCreated = (_id: string, mnemonic: string) => {
    setShowCreate(false);
    setMnemonicNote(mnemonic || null);
    void loadAll();
  };

  if (!authed) {
    return <AuthGate onAuthed={() => setAuthed(true)} />;
  }

  return (
    <div className={`flex min-h-screen ${isDark ? 'bg-gray-900 text-white' : 'bg-gray-100 text-gray-900'}`}>
      <Sidebar
        currentPage={currentPage}
        setCurrentPage={setCurrentPage}
        isDark={isDark}
        masterAddress={activeWallet?.address}
        onLogout={handleLogout}
      />
      <div className="flex-1 flex flex-col">
        <Header isDark={isDark} onCreate={() => setShowCreate(true)} />
        <main className="flex-1 p-6 overflow-auto">
          {currentPage === 'dashboard' && (
            <Dashboard isDark={isDark} wallets={wallets} transactions={transactions} balances={balances} loading={loading} error={error} />
          )}
          {currentPage === 'wallets' && (
            <Wallets isDark={isDark} masterId={masterId} wallets={subWallets} loading={loading} error={error} onRefresh={loadAll} />
          )}
          {currentPage === 'transactions' && (
            <Transactions isDark={isDark} masterId={masterId} transactions={transactions} loading={loading} error={error} onRefresh={loadAll} />
          )}
          {currentPage === 'treasury' && (
            <TreasuryPage isDark={isDark} masterId={masterId} />
          )}
          {currentPage === 'auto-sign' && (
            <AutoSignPage isDark={isDark} masterId={masterId} rules={autoSignRules} loading={loading} error={error} onRefresh={loadAll} />
          )}
          {currentPage === 'users' && (
            <UsersPage isDark={isDark} masterId={masterId} users={users} loading={loading} error={error} onRefresh={loadAll} />
          )}
          {currentPage === 'analytics' && (
            <AnalyticsPage isDark={isDark} masterId={masterId} />
          )}
          {currentPage === 'policies' && (
            <PoliciesPage isDark={isDark} masterId={masterId} />
          )}
          {currentPage === 'fees' && (
            <FeesPage isDark={isDark} masterId={masterId} />
          )}
          {currentPage === 'notifications' && (
            <NotificationsPage isDark={isDark} masterId={masterId} />
          )}
          {currentPage === 'webhooks' && (
            <WebhooksPage isDark={isDark} masterId={masterId} />
          )}
          {currentPage === 'audit' && (
            <AuditPage isDark={isDark} masterId={masterId} />
          )}
          {currentPage === 'multisig' && (
            <MultisigPage isDark={isDark} masterId={masterId} />
          )}
          {currentPage === 'chains' && (
            <ChainsPage isDark={isDark} masterId={masterId} />
          )}
          {currentPage === 'tokens' && (
            <TokensPage isDark={isDark} masterId={masterId} />
          )}
          {currentPage === 'feature-flags' && (
            <FeatureFlagsPage isDark={isDark} masterId={masterId} />
          )}
          {currentPage === 'passkeys' && (
            <PasskeysPage isDark={isDark} masterId={masterId} />
          )}
          {currentPage === 'settings' && <SettingsPage isDark={isDark} />}
        </main>
      </div>

      {showCreate && (
        <CreateWalletModal isDark={isDark} onClose={() => setShowCreate(false)} onCreated={handleCreated} />
      )}

      {mnemonicNote && (
        <div className="fixed bottom-6 right-6 max-w-md p-4 bg-amber-500 text-black rounded-lg shadow-lg z-50">
          <div className="font-semibold mb-1">Save your mnemonic (shown once):</div>
          <div className="font-mono text-sm break-words">{mnemonicNote}</div>
          <button onClick={() => setMnemonicNote(null)} className="mt-2 text-xs underline">Dismiss</button>
        </div>
      )}
    </div>
  );
};

export default App;
