// MasterWallet Web — additional pages that wire the remaining api.ts methods
// to real screens (Policies, Fees, Notifications, Webhooks, Audit, Multisig,
// Chains, Tokens, Feature Flags, Passkeys). Every page fetches live data from
// the canonical MasterWallet backend (:8450) with loading/error/empty states
// and full light/dark theme support via useTheme(). No mock data.

import React, { useCallback, useEffect, useState, type FormEvent } from 'react';
import {
  masterWalletAPI,
  ApiError,
  tradingControlAPI,
  type FeeConfig,
  type AuditLog,
  type NotificationItem,
  type Webhook,
  type MultisigWallet,
} from './api';

// ---------------- shared helpers ----------------

const card = (isDark: boolean) =>
  isDark ? 'bg-gray-800 border-gray-700 text-white' : 'bg-white border-gray-200 text-gray-900';
const muted = (isDark: boolean) => (isDark ? 'text-gray-400' : 'text-gray-500');
const inputCls = (isDark: boolean) =>
  `w-full px-3 py-2 rounded-lg border ${isDark ? 'bg-gray-900 border-gray-700 text-white' : 'bg-white border-gray-300 text-gray-900'} focus:outline-none focus:ring-2 focus:ring-blue-500`;
const btn = (isDark: boolean, variant: 'primary' | 'danger' | 'ghost' = 'primary') => {
  if (variant === 'danger') return 'px-3 py-1.5 rounded-lg text-sm bg-red-600 text-white hover:bg-red-700';
  if (variant === 'ghost')
    return `px-3 py-1.5 rounded-lg text-sm ${isDark ? 'bg-gray-700 text-gray-200 hover:bg-gray-600' : 'bg-gray-100 text-gray-700 hover:bg-gray-200'}`;
  return 'px-3 py-1.5 rounded-lg text-sm bg-blue-600 text-white hover:bg-blue-700';
};

function Banner({ isDark, loading, error }: { isDark: boolean; loading: boolean; error: string | null }) {
  if (loading)
    return <div className={`mb-4 p-3 rounded-lg ${isDark ? 'bg-gray-800 text-gray-300' : 'bg-blue-50 text-blue-700'}`}>Loading…</div>;
  if (error)
    return <div className="mb-4 p-3 rounded-lg bg-red-100 text-red-700">{error}</div>;
  return null;
}

const PageShell = ({ isDark, title, children }: { isDark: boolean; title: string; children: React.ReactNode }) => (
  <div className="space-y-4">
    <h2 className={`text-2xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>{title}</h2>
    {children}
  </div>
);

function useFetch<T>(loader: () => Promise<T>, deps: unknown[] = []): {
  data: T | null;
  loading: boolean;
  error: string | null;
  reload: () => Promise<void>;
  setData: React.Dispatch<React.SetStateAction<T | null>>;
} {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const run = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      setData(await loader());
    } catch (e) {
      setError(e instanceof ApiError ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);
  useEffect(() => { void run(); }, [run]);
  return { data, loading, error, reload: run, setData };
}

// ---------------- Policies ----------------

interface Policy {
  id: string;
  name: string;
  policy_type: string;
  is_active: boolean;
  priority: number;
  conditions?: Record<string, unknown>;
  actions?: Record<string, unknown>;
  created_at?: string;
}

export const PoliciesPage = ({ isDark, masterId }: { isDark: boolean; masterId?: string }) => {
  const { data, loading, error, reload } = useFetch(async () => {
    if (!masterId) return [] as Policy[];
    return masterWalletAPI.getPolicies(masterId);
  }, [masterId]);
  const [name, setName] = useState('');
  const [ptype, setPtype] = useState('spending_limit');
  const [priority, setPriority] = useState('1');
  const [submitting, setSubmitting] = useState(false);

  const create = async (e: FormEvent) => {
    e.preventDefault();
    if (!masterId || !name.trim()) return;
    setSubmitting(true);
    try {
      await masterWalletAPI.createPolicy(masterId, {
        name: name.trim(),
        policy_type: ptype,
        priority: Number(priority) || 0,
        conditions: {},
        actions: {},
      });
      setName('');
      void reload();
    } catch (err) {
      setError2(err instanceof ApiError ? err.message : String(err));
    } finally {
      setSubmitting(false);
    }
  };
  const [error2, setError2] = useState<string | null>(null);
  const remove = async (pid: string) => {
    if (!masterId) return;
    try {
      await masterWalletAPI.deletePolicy(masterId, pid);
      void reload();
    } catch (err) {
      setError2(err instanceof ApiError ? err.message : String(err));
    }
  };

  return (
    <PageShell isDark={isDark} title="Policies">
      <Banner isDark={isDark} loading={loading} error={error || error2} />
      <form onSubmit={create} className={`p-4 rounded-lg border space-y-3 ${card(isDark)}`}>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          <input className={inputCls(isDark)} placeholder="Policy name" value={name} onChange={(e) => setName(e.target.value)} required />
          <select className={inputCls(isDark)} value={ptype} onChange={(e) => setPtype(e.target.value)}>
            <option value="spending_limit">spending_limit</option>
            <option value="whitelist">whitelist</option>
            <option value="rate_limit">rate_limit</option>
            <option value="approval_threshold">approval_threshold</option>
          </select>
          <input className={inputCls(isDark)} type="number" placeholder="Priority" value={priority} onChange={(e) => setPriority(e.target.value)} />
        </div>
        <button className={btn(isDark)} disabled={submitting || !masterId}>{submitting ? 'Creating…' : 'Create Policy'}</button>
      </form>
      <div className="space-y-2">
        {(data ?? []).length === 0 ? (
          <div className={`p-4 rounded-lg border ${card(isDark)} ${muted(isDark)}`}>No policies configured.</div>
        ) : (
          (data ?? []).map((p) => (
            <div key={p.id} className={`p-4 rounded-lg border flex items-center justify-between ${card(isDark)}`}>
              <div>
                <div className="font-semibold">{p.name}</div>
                <div className={`text-sm ${muted(isDark)}`}>{p.policy_type} · priority {p.priority} · {p.is_active ? 'active' : 'inactive'}</div>
              </div>
              <button className={btn(isDark, 'danger')} onClick={() => remove(p.id)}>Delete</button>
            </div>
          ))
        )}
      </div>
    </PageShell>
  );
};

// ---------------- Fees ----------------

export const FeesPage = ({ isDark, masterId }: { isDark: boolean; masterId?: string }) => {
  const { data, loading, error, reload } = useFetch<FeeConfig[]>(async () => {
    if (!masterId) return [] as FeeConfig[];
    return masterWalletAPI.getFeeConfigs(masterId);
  }, [masterId]);
  const [name, setName] = useState('');
  const [pct, setPct] = useState('0.5');
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const create = async (e: FormEvent) => {
    e.preventDefault();
    if (!masterId || !name.trim()) return;
    setSubmitting(true);
    try {
      await masterWalletAPI.createFeeConfig(masterId, {
        name: name.trim(),
        fee_type: 'percentage',
        fee_percentage: Number(pct) || 0,
      } as any);
      setName('');
      void reload();
    } catch (e2) {
      setErr(e2 instanceof ApiError ? e2.message : String(e2));
    } finally {
      setSubmitting(false);
    }
  };
  const remove = async (fid: string) => {
    if (!masterId) return;
    try {
      await masterWalletAPI.deleteFeeConfig(masterId, fid);
      void reload();
    } catch (e2) {
      setErr(e2 instanceof ApiError ? e2.message : String(e2));
    }
  };

  return (
    <PageShell isDark={isDark} title="Fees">
      <Banner isDark={isDark} loading={loading} error={error || err} />
      <form onSubmit={create} className={`p-4 rounded-lg border space-y-3 ${card(isDark)}`}>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          <input className={inputCls(isDark)} placeholder="Fee name" value={name} onChange={(e) => setName(e.target.value)} required />
          <input className={inputCls(isDark)} type="number" step="0.01" placeholder="Percentage %" value={pct} onChange={(e) => setPct(e.target.value)} />
        </div>
        <button className={btn(isDark)} disabled={submitting || !masterId}>{submitting ? 'Creating…' : 'Create Fee'}</button>
      </form>
      <div className="space-y-2">
        {(data ?? []).length === 0 ? (
          <div className={`p-4 rounded-lg border ${card(isDark)} ${muted(isDark)}`}>No fee configs.</div>
        ) : (
          ((data ?? []) as any[]).map((f) => (
            <div key={f.id} className={`p-4 rounded-lg border flex items-center justify-between ${card(isDark)}`}>
              <div>
                <div className="font-semibold">{f.fee_type ?? 'Fee'}</div>
                <div className={`text-sm ${muted(isDark)}`}>{f.fee_percentage != null ? `${f.fee_percentage}%` : (f.fee_fixed ?? '—')}</div>
              </div>
              <button className={btn(isDark, 'danger')} onClick={() => remove(f.id)}>Delete</button>
            </div>
          ))
        )}
      </div>
    </PageShell>
  );
};

// ---------------- Notifications ----------------

export const NotificationsPage = ({ isDark, masterId }: { isDark: boolean; masterId?: string }) => {
  const { data, loading, error, reload } = useFetch<NotificationItem[]>(async () => {
    if (!masterId) return [] as NotificationItem[];
    return masterWalletAPI.getNotifications(masterId);
  }, [masterId]);
  const [title, setTitle] = useState('');
  const [message, setMessage] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const create = async (e: FormEvent) => {
    e.preventDefault();
    if (!masterId || !title.trim()) return;
    setSubmitting(true);
    try {
      await masterWalletAPI.createNotification(masterId, { title: title.trim(), message: message.trim(), type: 'info' } as any);
      setTitle(''); setMessage('');
      void reload();
    } catch (e2) {
      setErr(e2 instanceof ApiError ? e2.message : String(e2));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <PageShell isDark={isDark} title="Notifications">
      <Banner isDark={isDark} loading={loading} error={error || err} />
      <form onSubmit={create} className={`p-4 rounded-lg border space-y-3 ${card(isDark)}`}>
        <input className={inputCls(isDark)} placeholder="Title" value={title} onChange={(e) => setTitle(e.target.value)} required />
        <textarea className={inputCls(isDark)} placeholder="Message" value={message} onChange={(e) => setMessage(e.target.value)} rows={2} />
        <button className={btn(isDark)} disabled={submitting || !masterId}>{submitting ? 'Sending…' : 'Send Notification'}</button>
      </form>
      <div className="space-y-2">
        {(data ?? []).length === 0 ? (
          <div className={`p-4 rounded-lg border ${card(isDark)} ${muted(isDark)}`}>No notifications.</div>
        ) : (
          (data ?? []).map((n) => (
            <div key={n.id} className={`p-4 rounded-lg border ${card(isDark)}`}>
              <div className="font-semibold">{n.title ?? 'Notification'}</div>
              <div className={`text-sm ${muted(isDark)}`}>{n.message}</div>
            </div>
          ))
        )}
      </div>
    </PageShell>
  );
};

// ---------------- Webhooks ----------------

export const WebhooksPage = ({ isDark, masterId }: { isDark: boolean; masterId?: string }) => {
  const { data, loading, error, reload } = useFetch<Webhook[]>(async () => {
    if (!masterId) return [] as Webhook[];
    return masterWalletAPI.getWebhooks(masterId);
  }, [masterId]);
  const [url, setUrl] = useState('');
  const [events, setEvents] = useState('transaction.confirmed,balance.updated');
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const create = async (e: FormEvent) => {
    e.preventDefault();
    if (!masterId || !url.trim()) return;
    setSubmitting(true);
    try {
      await masterWalletAPI.createWebhook(masterId, {
        url: url.trim(),
        events: events.split(',').map((s) => s.trim()).filter(Boolean),
      } as any);
      setUrl('');
      void reload();
    } catch (e2) {
      setErr(e2 instanceof ApiError ? e2.message : String(e2));
    } finally {
      setSubmitting(false);
    }
  };
  const remove = async (wid: string) => {
    if (!masterId) return;
    try {
      await masterWalletAPI.deleteWebhook(masterId, wid);
      void reload();
    } catch (e2) {
      setErr(e2 instanceof ApiError ? e2.message : String(e2));
    }
  };

  return (
    <PageShell isDark={isDark} title="Webhooks">
      <Banner isDark={isDark} loading={loading} error={error || err} />
      <form onSubmit={create} className={`p-4 rounded-lg border space-y-3 ${card(isDark)}`}>
        <input className={inputCls(isDark)} placeholder="https://example.com/webhook" value={url} onChange={(e) => setUrl(e.target.value)} required />
        <input className={inputCls(isDark)} placeholder="event types (comma separated)" value={events} onChange={(e) => setEvents(e.target.value)} />
        <button className={btn(isDark)} disabled={submitting || !masterId}>{submitting ? 'Creating…' : 'Add Webhook'}</button>
      </form>
      <div className="space-y-2">
        {(data ?? []).length === 0 ? (
          <div className={`p-4 rounded-lg border ${card(isDark)} ${muted(isDark)}`}>No webhooks.</div>
        ) : (
          (data ?? []).map((w) => (
            <div key={w.id} className={`p-4 rounded-lg border flex items-center justify-between ${card(isDark)}`}>
              <div>
                <div className="font-mono text-sm">{w.url}</div>
                <div className={`text-xs ${muted(isDark)}`}>{(w.events ?? []).join(', ')}</div>
              </div>
              <button className={btn(isDark, 'danger')} onClick={() => remove(w.id)}>Delete</button>
            </div>
          ))
        )}
      </div>
    </PageShell>
  );
};

// ---------------- Audit ----------------

export const AuditPage = ({ isDark, masterId }: { isDark: boolean; masterId?: string }) => {
  const { data, loading, error } = useFetch<AuditLog[]>(async () => {
    if (!masterId) return [] as AuditLog[];
    return masterWalletAPI.getAuditLogs(masterId);
  }, [masterId]);
  const logs = data ?? [];
  return (
    <PageShell isDark={isDark} title="Audit Log">
      <Banner isDark={isDark} loading={loading} error={error} />
      {logs.length === 0 ? (
        <div className={`p-4 rounded-lg border ${card(isDark)} ${muted(isDark)}`}>No audit entries.</div>
      ) : (
        <div className={`rounded-lg border overflow-hidden ${card(isDark)}`}>
          <table className="w-full text-sm">
            <thead className={isDark ? 'bg-gray-900' : 'bg-gray-50'}>
              <tr>
                <th className="text-left p-3">Event</th>
                <th className="text-left p-3">Actor</th>
                <th className="text-left p-3">Target</th>
                <th className="text-left p-3">Severity</th>
                <th className="text-left p-3">Time</th>
              </tr>
            </thead>
            <tbody>
              {logs.map((l) => (
                <tr key={l.id ?? l.event_type + String(l.created_at)} className={isDark ? 'border-t border-gray-700' : 'border-t border-gray-200'}>
                  <td className="p-3">{l.event_type}</td>
                  <td className="p-3">{l.actor_id ?? '—'}</td>
                  <td className="p-3">{l.target_id ?? '—'}</td>
                  <td className="p-3">{l.severity ?? '—'}</td>
                  <td className="p-3">{l.created_at ?? '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </PageShell>
  );
};

// ---------------- Multisig ----------------

export const MultisigPage = ({ isDark, masterId }: { isDark: boolean; masterId?: string }) => {
  const [view, setView] = useState<'wallets' | 'txs'>('wallets');
  const [selectedWallet, setSelectedWallet] = useState<string | null>(null);
  const { data: wallets, loading: lw, error: ew, reload: rw } = useFetch<MultisigWallet[]>(async () => {
    if (!masterId) return [] as MultisigWallet[];
    return masterWalletAPI.getMultisigWallets(masterId);
  }, [masterId]);
  const { data: txs, loading: lt, error: et } = useFetch<unknown[]>(async () => {
    if (!masterId || !selectedWallet) return [];
    return masterWalletAPI.getMultisigTransactions(masterId, selectedWallet);
  }, [masterId, selectedWallet]);
  const [name, setName] = useState('');
  const [owners, setOwners] = useState('');
  const [threshold, setThreshold] = useState('1');
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const create = async (e: FormEvent) => {
    e.preventDefault();
    if (!masterId || !name.trim()) return;
    setSubmitting(true);
    try {
      await masterWalletAPI.createMultisigWallet(masterId, {
        name: name.trim(),
        owners: owners.split(',').map((s) => s.trim()).filter(Boolean),
        threshold: Number(threshold) || 1,
      });
      setName(''); setOwners(''); setThreshold('1');
      void rw();
    } catch (e2) {
      setErr(e2 instanceof ApiError ? e2.message : String(e2));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <PageShell isDark={isDark} title="Multisig Wallets">
      <Banner isDark={isDark} loading={lw || lt} error={ew || et || err} />
      <div className="flex gap-2 mb-4">
        <button className={btn(isDark, view === 'wallets' ? 'primary' : 'ghost')} onClick={() => setView('wallets')}>Wallets</button>
        <button className={btn(isDark, view === 'txs' ? 'primary' : 'ghost')} onClick={() => setView('txs')} disabled={!selectedWallet}>Transactions</button>
      </div>
      <form onSubmit={create} className={`p-4 rounded-lg border space-y-3 mb-4 ${card(isDark)}`}>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          <input className={inputCls(isDark)} placeholder="Wallet name" value={name} onChange={(e) => setName(e.target.value)} required />
          <input className={inputCls(isDark)} placeholder="owners (comma separated 0x…)" value={owners} onChange={(e) => setOwners(e.target.value)} />
          <input className={inputCls(isDark)} type="number" placeholder="threshold" value={threshold} onChange={(e) => setThreshold(e.target.value)} />
        </div>
        <button className={btn(isDark)} disabled={submitting || !masterId}>{submitting ? 'Creating…' : 'Create Multisig'}</button>
      </form>
      {view === 'wallets' ? (
        <div className="space-y-2">
          {(wallets ?? []).length === 0 ? (
            <div className={`p-4 rounded-lg border ${card(isDark)} ${muted(isDark)}`}>No multisig wallets.</div>
          ) : (
            (wallets ?? []).map((w: any) => (
              <div key={w.id} className={`p-4 rounded-lg border flex items-center justify-between ${card(isDark)}`}>
                <div>
                  <div className="font-semibold">{w.name}</div>
                  <div className={`text-sm ${muted(isDark)}`}>threshold {w.threshold} · {w.owners?.length ?? 0} owners · chain {w.chain_id ?? 1}</div>
                </div>
                <button className={btn(isDark, 'ghost')} onClick={() => { setSelectedWallet(w.id); setView('txs'); }}>View txs</button>
              </div>
            ))
          )}
        </div>
      ) : (
        <div className="space-y-2">
          {(txs ?? []).length === 0 ? (
            <div className={`p-4 rounded-lg border ${card(isDark)} ${muted(isDark)}`}>No transactions for this multisig wallet.</div>
          ) : (
            ((txs ?? []) as any[]).map((t: any) => (
              <div key={t.id} className={`p-4 rounded-lg border ${card(isDark)}`}>
                <div className="font-mono text-sm">→ {t.to_address} · {t.value}</div>
                <div className={`text-xs ${muted(isDark)}`}>status {t.status} · nonce {t.nonce}</div>
              </div>
            ))
          )}
        </div>
      )}
    </PageShell>
  );
};

// ---------------- Chains (UserWallet governance) ----------------


export const ChainsPage = ({ isDark, masterId }: { isDark: boolean; masterId?: string }) => {
  const { data, loading, error, reload } = useFetch<unknown[]>(async () => {
    if (!masterId) return [];
    return masterWalletAPI.listUserEVMChains(masterId);
  }, [masterId]);
  const [chainId, setChainId] = useState('');
  const [name, setName] = useState('');
  const [symbol, setSymbol] = useState('');
  const [rpc, setRpc] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const add = async (e: FormEvent) => {
    e.preventDefault();
    if (!masterId || !chainId.trim() || !name.trim()) return;
    setSubmitting(true);
    try {
      await masterWalletAPI.addUserEVMChain(masterId, {
        chain_id: Number(chainId),
        name: name.trim(),
        symbol: symbol.trim(),
        rpc_endpoint: rpc.trim(),
      } as any);
      setChainId(''); setName(''); setSymbol(''); setRpc('');
      void reload();
    } catch (e2) {
      setErr(e2 instanceof ApiError ? e2.message : String(e2));
    } finally {
      setSubmitting(false);
    }
  };
  const remove = async (cid: number) => {
    if (!masterId) return;
    try {
      await masterWalletAPI.removeUserEVMChain(masterId, cid);
      void reload();
    } catch (e2) {
      setErr(e2 instanceof ApiError ? e2.message : String(e2));
    }
  };

  return (
    <PageShell isDark={isDark} title="UserWallet EVM Chains">
      <Banner isDark={isDark} loading={loading} error={error || err} />
      <form onSubmit={add} className={`p-4 rounded-lg border space-y-3 ${card(isDark)}`}>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
          <input className={inputCls(isDark)} type="number" placeholder="Chain ID" value={chainId} onChange={(e) => setChainId(e.target.value)} required />
          <input className={inputCls(isDark)} placeholder="Name" value={name} onChange={(e) => setName(e.target.value)} required />
          <input className={inputCls(isDark)} placeholder="Symbol" value={symbol} onChange={(e) => setSymbol(e.target.value)} />
          <input className={inputCls(isDark)} placeholder="RPC endpoint" value={rpc} onChange={(e) => setRpc(e.target.value)} />
        </div>
        <button className={btn(isDark)} disabled={submitting || !masterId}>{submitting ? 'Adding…' : 'Add Chain'}</button>
      </form>
      <div className="space-y-2">
        {(data ?? []).length === 0 ? (
          <div className={`p-4 rounded-lg border ${card(isDark)} ${muted(isDark)}`}>No user EVM chains configured.</div>
        ) : (
          ((data ?? []) as any[]).map((c) => (
            <div key={c.chain_id} className={`p-4 rounded-lg border flex items-center justify-between ${card(isDark)}`}>
              <div>
                <div className="font-semibold">{c.name} <span className={muted(isDark)}>(#{c.chain_id})</span></div>
                <div className={`text-sm ${muted(isDark)}`}>{c.symbol} · {c.rpc_endpoint ?? '—'}</div>
              </div>
              <button className={btn(isDark, 'danger')} onClick={() => remove(c.chain_id)}>Remove</button>
            </div>
          ))
        )}
      </div>
    </PageShell>
  );
};

// ---------------- Tokens (UserWallet governance) ----------------


export const TokensPage = ({ isDark, masterId }: { isDark: boolean; masterId?: string }) => {
  const { data, loading, error, reload } = useFetch<unknown[]>(async () => {
    if (!masterId) return [];
    return masterWalletAPI.listUserTokens(masterId);
  }, [masterId]);
  const [addr, setAddr] = useState('');
  const [symbol, setSymbol] = useState('');
  const [decimals, setDecimals] = useState('18');
  const [chainId, setChainId] = useState('1');
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const add = async (e: FormEvent) => {
    e.preventDefault();
    if (!masterId || !addr.trim()) return;
    setSubmitting(true);
    try {
      await masterWalletAPI.addUserToken(masterId, {
        contract_address: addr.trim(),
        symbol: symbol.trim(),
        decimals: Number(decimals) || 18,
        chain_id: Number(chainId) || 1,
      } as any);
      setAddr(''); setSymbol('');
      void reload();
    } catch (e2) {
      setErr(e2 instanceof ApiError ? e2.message : String(e2));
    } finally {
      setSubmitting(false);
    }
  };
  const remove = async (tid: string) => {
    if (!masterId) return;
    try {
      await masterWalletAPI.removeUserToken(masterId, tid);
      void reload();
    } catch (e2) {
      setErr(e2 instanceof ApiError ? e2.message : String(e2));
    }
  };

  return (
    <PageShell isDark={isDark} title="UserWallet Tokens">
      <Banner isDark={isDark} loading={loading} error={error || err} />
      <form onSubmit={add} className={`p-4 rounded-lg border space-y-3 ${card(isDark)}`}>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
          <input className={inputCls(isDark)} placeholder="Contract address 0x…" value={addr} onChange={(e) => setAddr(e.target.value)} required />
          <input className={inputCls(isDark)} placeholder="Symbol" value={symbol} onChange={(e) => setSymbol(e.target.value)} />
          <input className={inputCls(isDark)} type="number" placeholder="Decimals" value={decimals} onChange={(e) => setDecimals(e.target.value)} />
          <input className={inputCls(isDark)} type="number" placeholder="Chain ID" value={chainId} onChange={(e) => setChainId(e.target.value)} />
        </div>
        <button className={btn(isDark)} disabled={submitting || !masterId}>{submitting ? 'Adding…' : 'Add Token'}</button>
      </form>
      <div className="space-y-2">
        {(data ?? []).length === 0 ? (
          <div className={`p-4 rounded-lg border ${card(isDark)} ${muted(isDark)}`}>No user tokens configured.</div>
        ) : (
          ((data ?? []) as any[]).map((t) => (
            <div key={t.id} className={`p-4 rounded-lg border flex items-center justify-between ${card(isDark)}`}>
              <div>
                <div className="font-semibold">{t.symbol ?? 'Token'}</div>
                <div className={`text-sm font-mono ${muted(isDark)}`}>{t.contract_address}</div>
              </div>
              <button className={btn(isDark, 'danger')} onClick={() => remove(t.id)}>Remove</button>
            </div>
          ))
        )}
      </div>
    </PageShell>
  );
};

// ---------------- Feature Flags ----------------


export const FeatureFlagsPage = ({ isDark, masterId }: { isDark: boolean; masterId?: string }) => {
  const { data, loading, error, reload } = useFetch<unknown[]>(async () => {
    if (!masterId) return [];
    return masterWalletAPI.listFeatureFlags(masterId);
  }, [masterId]);
  const [key, setKey] = useState('');
  const [desc, setDesc] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const add = async (e: FormEvent) => {
    e.preventDefault();
    if (!masterId || !key.trim()) return;
    setSubmitting(true);
    try {
      await masterWalletAPI.addFeatureFlag(masterId, { flag_key: key.trim(), enabled: true, description: desc.trim() } as any);
      setKey(''); setDesc('');
      void reload();
    } catch (e2) {
      setErr(e2 instanceof ApiError ? e2.message : String(e2));
    } finally {
      setSubmitting(false);
    }
  };
  const remove = async (fid: string | number) => {
    if (!masterId) return;
    try {
      await masterWalletAPI.removeFeatureFlag(masterId, fid);
      void reload();
    } catch (e2) {
      setErr(e2 instanceof ApiError ? e2.message : String(e2));
    }
  };

  return (
    <PageShell isDark={isDark} title="Feature Flags">
      <Banner isDark={isDark} loading={loading} error={error || err} />
      <form onSubmit={add} className={`p-4 rounded-lg border space-y-3 ${card(isDark)}`}>
        <input className={inputCls(isDark)} placeholder="flag_key (e.g. user_wallet.swap)" value={key} onChange={(e) => setKey(e.target.value)} required />
        <input className={inputCls(isDark)} placeholder="description" value={desc} onChange={(e) => setDesc(e.target.value)} />
        <button className={btn(isDark)} disabled={submitting || !masterId}>{submitting ? 'Adding…' : 'Add Flag'}</button>
      </form>
      <div className="space-y-2">
        {(data ?? []).length === 0 ? (
          <div className={`p-4 rounded-lg border ${card(isDark)} ${muted(isDark)}`}>No feature flags.</div>
        ) : (
          ((data ?? []) as any[]).map((f) => (
            <div key={f.id} className={`p-4 rounded-lg border flex items-center justify-between ${card(isDark)}`}>
              <div>
                <div className="font-semibold">{f.flag_key ?? f.id}</div>
                <div className={`text-sm ${muted(isDark)}`}>{f.description ?? ''} · {f.enabled ? 'enabled' : 'disabled'}</div>
              </div>
              <button className={btn(isDark, 'danger')} onClick={() => remove(f.id)}>Remove</button>
            </div>
          ))
        )}
      </div>
    </PageShell>
  );
};

// ---------------- Passkeys ----------------

import PasskeyService from './services/PasskeyService';

export const PasskeysPage = ({ isDark, masterId }: { isDark: boolean; masterId?: string }) => {
  const { data, loading, error, reload } = useFetch(async () => {
    if (!masterId) return [];
    return PasskeyService.listRegistered(masterId);
  }, [masterId]);
  const [label, setLabel] = useState('');
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<string | null>(null);
  const [err, setErr] = useState<string | null>(null);

  const register = async (e: FormEvent) => {
    e.preventDefault();
    if (!masterId) return;
    setBusy(true); setResult(null); setErr(null);
    try {
      const challenge = crypto.getRandomValues(new Uint8Array(32));
      const challengeB64 = btoa(String.fromCharCode(...challenge)).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
      const res = await PasskeyService.register(
        masterId,
        { name: 'TigerWallet MasterWallet' },
        { id: masterId, name: label || 'master-owner', displayName: label || 'Master Owner' },
        challengeB64,
        label.trim() || undefined,
      );
      if (res.success) setResult(`Registered passkey ${res.passkeyId ?? res.credential?.id}`);
      else setErr(res.error ?? 'Registration failed');
      void reload();
    } catch (e2) {
      setErr(e2 instanceof Error ? e2.message : String(e2));
    } finally {
      setBusy(false);
    }
  };

  const remove = async (credId: string) => {
    if (!masterId) return;
    try {
      await PasskeyService.remove(masterId, credId);
      void reload();
    } catch (e2) {
      setErr(e2 instanceof Error ? e2.message : String(e2));
    }
  };

  return (
    <PageShell isDark={isDark} title="Passkeys (WebAuthn)">
      <Banner isDark={isDark} loading={loading} error={error || err} />
      {result && <div className="mb-4 p-3 rounded-lg bg-green-100 text-green-800">{result}</div>}
      <form onSubmit={register} className={`p-4 rounded-lg border space-y-3 ${card(isDark)}`}>
        <input className={inputCls(isDark)} placeholder="Label (e.g. iPhone, YubiKey)" value={label} onChange={(e) => setLabel(e.target.value)} />
        <button className={btn(isDark)} disabled={busy || !masterId || !PasskeyService.isSupported()}>
          {busy ? 'Ceremony in progress…' : !PasskeyService.isSupported() ? 'WebAuthn not supported' : 'Register Passkey'}
        </button>
      </form>
      <div className="space-y-2">
        {(data ?? []).length === 0 ? (
          <div className={`p-4 rounded-lg border ${card(isDark)} ${muted(isDark)}`}>No passkeys registered.</div>
        ) : (
          (data ?? []).map((p: any) => (
            <div key={p.id} className={`p-4 rounded-lg border flex items-center justify-between ${card(isDark)}`}>
              <div>
                <div className="font-semibold">{p.label ?? 'Passkey'}</div>
                <div className={`text-xs font-mono ${muted(isDark)}`}>{p.credential_id}</div>
                <div className={`text-xs ${muted(isDark)}`}>sign count {p.sign_count}</div>
              </div>
              <button className={btn(isDark, 'danger')} onClick={() => remove(p.credential_id)}>Revoke</button>
            </div>
          ))
        )}
      </div>
    </PageShell>
  );
};


// ---------------- Trading Control-Plane ----------------

const TC_VERTICALS = ['spot', 'perpetual', 'futures', 'margin', 'options', 'copy', 'liquidity'];

type TCTab = 'contracts' | 'pools' | 'pairs' | 'margin' | 'options' | 'copy' | 'verticals' | 'audit';

type TCLifecycleAPI = {
  list: (mid: string) => Promise<any>;
  create: (mid: string, data: any) => Promise<any>;
  stop: (mid: string, id: string) => Promise<any>;
  resume: (mid: string, id: string) => Promise<any>;
  remove: (mid: string, id: string) => Promise<any>;
};

export const TradingControlPage = ({ isDark, masterId }: { isDark: boolean; masterId?: string }) => {
  const [tab, setTab] = useState<TCTab>('contracts');
  const [overview, setOverview] = useState<any>(null);
  const [rows, setRows] = useState<any[]>([]);
  const [auditRows, setAuditRows] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [contractForm, setContractForm] = useState({ kind: 'perpetual', symbol: '', base_asset: '', quote_asset: 'USDT', max_leverage: '10' });
  const [poolForm, setPoolForm] = useState({ chain_id: '1', dex: '', token0: '', token1: '', fee_bps: '30' });
  const [pairForm, setPairForm] = useState({ symbol: '', base_asset: '', quote_asset: 'USDT', market: 'spot' });
  const [marginForm, setMarginForm] = useState({ symbol: '', base_asset: '', quote_asset: 'USDT', max_leverage: '3' });
  const [optionsForm, setOptionsForm] = useState({ underlying: '', quote_asset: 'USDT', strike: '', expiry: '', style: 'call', iv_bps: '8000', contract_size: '1' });
  const [copyForm, setCopyForm] = useState({ trader: '', display_name: '', fee_bps: '100', max_copiers: '0' });

  const load = useCallback(async () => {
    if (!masterId) { setLoading(false); setRows([]); setAuditRows([]); setOverview(null); return; }
    setLoading(true);
    setError(null);
    try {
      if (tab === 'contracts') { const d: any = await tradingControlAPI.contracts.list(masterId); setRows(d.contracts || []); }
      else if (tab === 'pools') { const d: any = await tradingControlAPI.pools.list(masterId); setRows(d.pools || []); }
      else if (tab === 'pairs') { const d: any = await tradingControlAPI.pairs.list(masterId); setRows(d.pairs || []); }
      else if (tab === 'margin') { const d: any = await tradingControlAPI.marginMarkets.list(masterId); setRows(d.margin_markets || []); }
      else if (tab === 'options') { const d: any = await tradingControlAPI.optionsSeries.list(masterId); setRows(d.series || []); }
      else if (tab === 'copy') { const d: any = await tradingControlAPI.copyTraders.list(masterId); setRows(d.traders || []); }
      else if (tab === 'audit') { const d: any = await tradingControlAPI.audit(masterId); setAuditRows(d.audit || []); }
      try { setOverview(await tradingControlAPI.overview(masterId)); } catch { /* best-effort */ }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, [tab, masterId]);

  useEffect(() => { void load(); }, [load]);

  const run = async (fn: () => Promise<any>) => {
    if (!masterId) return;
    setBusy(true);
    try { await fn(); await load(); }
    catch (err) { setError(err instanceof ApiError ? err.message : String(err)); }
    finally { setBusy(false); }
  };

  const badge = (st: string) => (
    <span className={`px-2 py-1 rounded text-xs ${st === 'active' ? 'bg-green-100 text-green-700' : st === 'removed' ? 'bg-red-100 text-red-700' : 'bg-yellow-100 text-yellow-700'}`}>{st}</span>
  );

  const lifecycle = (r: any, api: TCLifecycleAPI) => (
    <div className="flex gap-2 flex-wrap">
      <button className={btn(isDark, 'ghost')} disabled={busy || !masterId || r.status === 'stopped'} onClick={() => run(() => api.stop(masterId!, r.id))}>Stop</button>
      <button className={btn(isDark)} disabled={busy || !masterId || r.status === 'active'} onClick={() => run(() => api.resume(masterId!, r.id))}>Resume</button>
      <button className={btn(isDark, 'danger')} disabled={busy || !masterId} onClick={() => { if (confirm('Remove permanently?')) run(() => api.remove(masterId!, r.id)); }}>Remove</button>
    </div>
  );

  const halts = (overview && overview.vertical_halts) || {};

  const th = `px-4 py-2 text-left text-xs font-semibold ${muted(isDark)}`;
  const td = 'px-4 py-3';

  return (
    <PageShell isDark={isDark} title="Trading Control-Plane">
      <p className={muted(isDark)}>
        Builtin TigerWallet trading governance — contracts, liquidity pools, pairs, margin markets, options, copy trading.
        Decisions publish to the shared control plane enforced by every wallet engine.
      </p>

      {overview && (
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3">
          {[
            ['Contracts', overview.contracts_active],
            ['Pools', overview.pools_active],
            ['Pairs', overview.pairs_active],
            ['Margin Mkts', overview.margin_markets_active],
            ['Options', overview.options_active],
            ['Copy Configs', overview.copy_configs_active],
          ].map(([label, value]) => (
            <div key={label as string} className={`p-3 rounded-lg border ${card(isDark)}`}>
              <div className="text-xl font-bold">{(value as number) ?? 0}</div>
              <div className={`text-xs ${muted(isDark)}`}>{label}</div>
            </div>
          ))}
        </div>
      )}

      {!masterId && <div className="p-3 rounded-lg bg-orange-500/10 text-orange-500 text-sm">Create a master wallet first to manage the trading control-plane.</div>}

      <div className="flex gap-2 flex-wrap">
        {(['contracts', 'pools', 'pairs', 'margin', 'options', 'copy', 'verticals', 'audit'] as TCTab[]).map((t) => (
          <button key={t} className={tab === t ? btn(isDark) : btn(isDark, 'ghost')} onClick={() => setTab(t)}>
            {t === 'margin' ? 'Margin Markets' : t === 'options' ? 'Options Series' : t === 'copy' ? 'Copy Traders' : t.charAt(0).toUpperCase() + t.slice(1)}
          </button>
        ))}
      </div>

      <Banner isDark={isDark} loading={loading} error={error} />

      {!loading && tab === 'verticals' && (
        <div className={`rounded-lg border overflow-hidden ${card(isDark)}`}>
          <table className="w-full">
            <thead><tr>{['Vertical', 'State', 'Actions'].map((h) => <th key={h} className={th}>{h}</th>)}</tr></thead>
            <tbody>
              {TC_VERTICALS.map((v) => (
                <tr key={v} className="border-t border-gray-700/40">
                  <td className={td}>{v}</td>
                  <td className={td}>{halts[v] ? <span className="px-2 py-1 rounded text-xs bg-red-100 text-red-700">halted</span> : <span className="px-2 py-1 rounded text-xs bg-green-100 text-green-700">running</span>}</td>
                  <td className={td}><div className="flex gap-2">
                    <button className={btn(isDark, 'danger')} disabled={busy || !masterId || !!halts[v]} onClick={() => run(() => tradingControlAPI.haltVertical(masterId!, v))}>Halt</button>
                    <button className={btn(isDark)} disabled={busy || !masterId || !halts[v]} onClick={() => run(() => tradingControlAPI.resumeVertical(masterId!, v))}>Resume</button>
                  </div></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {!loading && tab === 'audit' && (
        <div className={`rounded-lg border overflow-x-auto ${card(isDark)}`}>
          <table className="w-full">
            <thead><tr>{['Actor', 'Role', 'Action', 'Kind', 'Entity', 'When'].map((h) => <th key={h} className={th}>{h}</th>)}</tr></thead>
            <tbody>
              {auditRows.length === 0 && <tr><td colSpan={6} className={`${td} text-center ${muted(isDark)}`}>No control-plane actions recorded yet.</td></tr>}
              {auditRows.map((a, i) => (
                <tr key={a.id || i} className="border-t border-gray-700/40">
                  <td className={`${td} ${muted(isDark)}`}>{a.actor || '—'}</td>
                  <td className={`${td} ${muted(isDark)}`}>{a.actor_role || '—'}</td>
                  <td className={td}>{a.action}</td>
                  <td className={`${td} ${muted(isDark)}`}>{a.kind}</td>
                  <td className={td}>{a.entity}</td>
                  <td className={`${td} ${muted(isDark)}`}>{a.created_at ? new Date(a.created_at).toLocaleString() : ''}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {!loading && tab === 'contracts' && (
        <>
          <form onSubmit={(e) => { e.preventDefault(); void run(() => tradingControlAPI.contracts.create(masterId!, { ...contractForm, max_leverage: Number(contractForm.max_leverage) || 1 })); }} className={`p-4 rounded-lg border space-y-3 ${card(isDark)}`}>
            <div className="font-semibold">New Contract</div>
            <div className="flex gap-2 flex-wrap">
              <select className={inputCls(isDark)} value={contractForm.kind} onChange={(e) => setContractForm({ ...contractForm, kind: e.target.value })}>
                <option value="perpetual">perpetual</option><option value="futures">futures</option><option value="options">options</option>
              </select>
              <input className={inputCls(isDark)} placeholder="Symbol (BTC-PERP)" required value={contractForm.symbol} onChange={(e) => setContractForm({ ...contractForm, symbol: e.target.value })} />
              <input className={inputCls(isDark)} placeholder="Base (BTC)" required value={contractForm.base_asset} onChange={(e) => setContractForm({ ...contractForm, base_asset: e.target.value })} />
              <input className={inputCls(isDark)} placeholder="Quote" required value={contractForm.quote_asset} onChange={(e) => setContractForm({ ...contractForm, quote_asset: e.target.value })} />
              <input className={inputCls(isDark)} type="number" placeholder="Max lev" value={contractForm.max_leverage} onChange={(e) => setContractForm({ ...contractForm, max_leverage: e.target.value })} />
              <button className={btn(isDark)} disabled={busy}>Create</button>
            </div>
          </form>
          <div className={`rounded-lg border overflow-x-auto ${card(isDark)}`}>
            <table className="w-full">
              <thead><tr>{['Kind', 'Symbol', 'Assets', 'Max Lev', 'Status', 'Actions'].map((h) => <th key={h} className={th}>{h}</th>)}</tr></thead>
              <tbody>
                {rows.length === 0 && <tr><td colSpan={6} className={`${td} text-center ${muted(isDark)}`}>No contracts yet.</td></tr>}
                {rows.map((r) => (
                  <tr key={r.id} className="border-t border-gray-700/40">
                    <td className={`${td} ${muted(isDark)}`}>{r.kind}</td>
                    <td className={td}>{r.symbol}</td>
                    <td className={`${td} ${muted(isDark)}`}>{r.base_asset}/{r.quote_asset}</td>
                    <td className={`${td} ${muted(isDark)}`}>{r.max_leverage}x</td>
                    <td className={td}>{badge(r.status)}</td>
                    <td className={td}>{lifecycle(r, tradingControlAPI.contracts)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {!loading && tab === 'pools' && (
        <>
          <form onSubmit={(e) => { e.preventDefault(); void run(() => tradingControlAPI.pools.create(masterId!, { chain_id: Number(poolForm.chain_id), dex: poolForm.dex, token0: poolForm.token0, token1: poolForm.token1, fee_bps: Number(poolForm.fee_bps) || 30 })); }} className={`p-4 rounded-lg border space-y-3 ${card(isDark)}`}>
            <div className="font-semibold">New Liquidity Pool</div>
            <div className="flex gap-2 flex-wrap">
              <input className={inputCls(isDark)} type="number" placeholder="Chain ID" required value={poolForm.chain_id} onChange={(e) => setPoolForm({ ...poolForm, chain_id: e.target.value })} />
              <input className={inputCls(isDark)} placeholder="DEX" required value={poolForm.dex} onChange={(e) => setPoolForm({ ...poolForm, dex: e.target.value })} />
              <input className={inputCls(isDark)} placeholder="Token0" required value={poolForm.token0} onChange={(e) => setPoolForm({ ...poolForm, token0: e.target.value })} />
              <input className={inputCls(isDark)} placeholder="Token1" required value={poolForm.token1} onChange={(e) => setPoolForm({ ...poolForm, token1: e.target.value })} />
              <input className={inputCls(isDark)} type="number" placeholder="Fee bps" value={poolForm.fee_bps} onChange={(e) => setPoolForm({ ...poolForm, fee_bps: e.target.value })} />
              <button className={btn(isDark)} disabled={busy}>Create</button>
            </div>
          </form>
          <div className={`rounded-lg border overflow-x-auto ${card(isDark)}`}>
            <table className="w-full">
              <thead><tr>{['Chain', 'DEX', 'Tokens', 'Fee', 'Status', 'Actions'].map((h) => <th key={h} className={th}>{h}</th>)}</tr></thead>
              <tbody>
                {rows.length === 0 && <tr><td colSpan={6} className={`${td} text-center ${muted(isDark)}`}>No pools yet.</td></tr>}
                {rows.map((r) => (
                  <tr key={r.id} className="border-t border-gray-700/40">
                    <td className={`${td} ${muted(isDark)}`}>{r.chain_id}</td>
                    <td className={td}>{r.dex}</td>
                    <td className={`${td} ${muted(isDark)}`}>{r.token0}/{r.token1}</td>
                    <td className={`${td} ${muted(isDark)}`}>{r.fee_bps} bps</td>
                    <td className={td}>{badge(r.status)}</td>
                    <td className={td}>{lifecycle(r, tradingControlAPI.pools)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {!loading && tab === 'pairs' && (
        <>
          <form onSubmit={(e) => { e.preventDefault(); void run(() => tradingControlAPI.pairs.create(masterId!, pairForm)); }} className={`p-4 rounded-lg border space-y-3 ${card(isDark)}`}>
            <div className="font-semibold">New Trading Pair</div>
            <div className="flex gap-2 flex-wrap">
              <input className={inputCls(isDark)} placeholder="Symbol (BTC/USDT)" required value={pairForm.symbol} onChange={(e) => setPairForm({ ...pairForm, symbol: e.target.value })} />
              <input className={inputCls(isDark)} placeholder="Base" required value={pairForm.base_asset} onChange={(e) => setPairForm({ ...pairForm, base_asset: e.target.value })} />
              <input className={inputCls(isDark)} placeholder="Quote" required value={pairForm.quote_asset} onChange={(e) => setPairForm({ ...pairForm, quote_asset: e.target.value })} />
              <select className={inputCls(isDark)} value={pairForm.market} onChange={(e) => setPairForm({ ...pairForm, market: e.target.value })}>
                <option value="spot">spot</option><option value="perpetual">perpetual</option><option value="margin">margin</option>
              </select>
              <button className={btn(isDark)} disabled={busy}>Create</button>
            </div>
          </form>
          <div className={`rounded-lg border overflow-x-auto ${card(isDark)}`}>
            <table className="w-full">
              <thead><tr>{['Symbol', 'Assets', 'Market', 'Status', 'Actions'].map((h) => <th key={h} className={th}>{h}</th>)}</tr></thead>
              <tbody>
                {rows.length === 0 && <tr><td colSpan={5} className={`${td} text-center ${muted(isDark)}`}>No pairs yet.</td></tr>}
                {rows.map((r) => (
                  <tr key={r.id} className="border-t border-gray-700/40">
                    <td className={td}>{r.symbol}</td>
                    <td className={`${td} ${muted(isDark)}`}>{r.base_asset}/{r.quote_asset}</td>
                    <td className={`${td} ${muted(isDark)}`}>{r.market}</td>
                    <td className={td}>{badge(r.status)}</td>
                    <td className={td}>{lifecycle(r, tradingControlAPI.pairs)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {!loading && tab === 'margin' && (
        <>
          <form onSubmit={(e) => { e.preventDefault(); void run(() => tradingControlAPI.marginMarkets.create(masterId!, { ...marginForm, max_leverage: Number(marginForm.max_leverage) || 3 })); }} className={`p-4 rounded-lg border space-y-3 ${card(isDark)}`}>
            <div className="font-semibold">New Margin Market</div>
            <div className="flex gap-2 flex-wrap">
              <input className={inputCls(isDark)} placeholder="Symbol (BTC/USDT)" required value={marginForm.symbol} onChange={(e) => setMarginForm({ ...marginForm, symbol: e.target.value })} />
              <input className={inputCls(isDark)} placeholder="Base" required value={marginForm.base_asset} onChange={(e) => setMarginForm({ ...marginForm, base_asset: e.target.value })} />
              <input className={inputCls(isDark)} placeholder="Quote" required value={marginForm.quote_asset} onChange={(e) => setMarginForm({ ...marginForm, quote_asset: e.target.value })} />
              <input className={inputCls(isDark)} type="number" placeholder="Max lev" value={marginForm.max_leverage} onChange={(e) => setMarginForm({ ...marginForm, max_leverage: e.target.value })} />
              <button className={btn(isDark)} disabled={busy}>Create</button>
            </div>
          </form>
          <div className={`rounded-lg border overflow-x-auto ${card(isDark)}`}>
            <table className="w-full">
              <thead><tr>{['Symbol', 'Assets', 'Max Lev', 'Status', 'Actions'].map((h) => <th key={h} className={th}>{h}</th>)}</tr></thead>
              <tbody>
                {rows.length === 0 && <tr><td colSpan={5} className={`${td} text-center ${muted(isDark)}`}>No margin markets yet.</td></tr>}
                {rows.map((r) => (
                  <tr key={r.id} className="border-t border-gray-700/40">
                    <td className={td}>{r.symbol}</td>
                    <td className={`${td} ${muted(isDark)}`}>{r.base_asset}/{r.quote_asset}</td>
                    <td className={`${td} ${muted(isDark)}`}>{r.max_leverage}x</td>
                    <td className={td}>{badge(r.status)}</td>
                    <td className={td}>{lifecycle(r, tradingControlAPI.marginMarkets)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {!loading && tab === 'options' && (
        <>
          <form onSubmit={(e) => { e.preventDefault(); void run(() => tradingControlAPI.optionsSeries.create(masterId!, { underlying: optionsForm.underlying, quote_asset: optionsForm.quote_asset, strike: optionsForm.strike, expiry_unix: Math.floor(new Date(optionsForm.expiry).getTime() / 1000), style: optionsForm.style, iv_bps: Number(optionsForm.iv_bps) || 8000, contract_size: optionsForm.contract_size })); }} className={`p-4 rounded-lg border space-y-3 ${card(isDark)}`}>
            <div className="font-semibold">New Options Series</div>
            <div className="flex gap-2 flex-wrap">
              <input className={inputCls(isDark)} placeholder="Underlying (BTC)" required value={optionsForm.underlying} onChange={(e) => setOptionsForm({ ...optionsForm, underlying: e.target.value })} />
              <input className={inputCls(isDark)} placeholder="Quote" required value={optionsForm.quote_asset} onChange={(e) => setOptionsForm({ ...optionsForm, quote_asset: e.target.value })} />
              <input className={inputCls(isDark)} type="number" step="any" placeholder="Strike" required value={optionsForm.strike} onChange={(e) => setOptionsForm({ ...optionsForm, strike: e.target.value })} />
              <input className={inputCls(isDark)} type="datetime-local" required value={optionsForm.expiry} onChange={(e) => setOptionsForm({ ...optionsForm, expiry: e.target.value })} />
              <select className={inputCls(isDark)} value={optionsForm.style} onChange={(e) => setOptionsForm({ ...optionsForm, style: e.target.value })}>
                <option value="call">call</option><option value="put">put</option>
              </select>
              <input className={inputCls(isDark)} type="number" placeholder="IV bps" value={optionsForm.iv_bps} onChange={(e) => setOptionsForm({ ...optionsForm, iv_bps: e.target.value })} />
              <input className={inputCls(isDark)} placeholder="Contract size" value={optionsForm.contract_size} onChange={(e) => setOptionsForm({ ...optionsForm, contract_size: e.target.value })} />
              <button className={btn(isDark)} disabled={busy || !masterId}>Create</button>
            </div>
          </form>
          <div className={`rounded-lg border overflow-x-auto ${card(isDark)}`}>
            <table className="w-full">
              <thead><tr>{['Underlying', 'Strike', 'Style', 'Expiry', 'IV', 'Status', 'Actions'].map((h) => <th key={h} className={th}>{h}</th>)}</tr></thead>
              <tbody>
                {rows.length === 0 && <tr><td colSpan={7} className={`${td} text-center ${muted(isDark)}`}>No options series yet.</td></tr>}
                {rows.map((r) => (
                  <tr key={r.id} className="border-t border-gray-700/40">
                    <td className={td}>{r.underlying}<span className={muted(isDark)}>/{r.quote_asset}</span></td>
                    <td className={`${td} ${muted(isDark)}`}>{r.strike}</td>
                    <td className={`${td} ${muted(isDark)}`}>{r.style}</td>
                    <td className={`${td} ${muted(isDark)}`}>{r.expiry_unix ? new Date(Number(r.expiry_unix) * 1000).toLocaleString() : ''}</td>
                    <td className={`${td} ${muted(isDark)}`}>{r.iv_bps} bps</td>
                    <td className={td}>{badge(r.status)}</td>
                    <td className={td}>{lifecycle(r, tradingControlAPI.optionsSeries)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {!loading && tab === 'copy' && (
        <>
          <form onSubmit={(e) => { e.preventDefault(); void run(() => tradingControlAPI.copyTraders.create(masterId!, { trader: copyForm.trader, display_name: copyForm.display_name, fee_bps: Number(copyForm.fee_bps) || 0, max_copiers: Number(copyForm.max_copiers) || 0 })); }} className={`p-4 rounded-lg border space-y-3 ${card(isDark)}`}>
            <div className="font-semibold">New Copy Trader</div>
            <div className="flex gap-2 flex-wrap">
              <input className={inputCls(isDark)} placeholder="Trader address (0x…)" required value={copyForm.trader} onChange={(e) => setCopyForm({ ...copyForm, trader: e.target.value })} />
              <input className={inputCls(isDark)} placeholder="Display name" value={copyForm.display_name} onChange={(e) => setCopyForm({ ...copyForm, display_name: e.target.value })} />
              <input className={inputCls(isDark)} type="number" placeholder="Fee bps" value={copyForm.fee_bps} onChange={(e) => setCopyForm({ ...copyForm, fee_bps: e.target.value })} />
              <input className={inputCls(isDark)} type="number" placeholder="Max copiers (0 = unlimited)" value={copyForm.max_copiers} onChange={(e) => setCopyForm({ ...copyForm, max_copiers: e.target.value })} />
              <button className={btn(isDark)} disabled={busy || !masterId}>Create</button>
            </div>
          </form>
          <div className={`rounded-lg border overflow-x-auto ${card(isDark)}`}>
            <table className="w-full">
              <thead><tr>{['Trader', 'Name', 'Fee', 'Max Copiers', 'Status', 'Actions'].map((h) => <th key={h} className={th}>{h}</th>)}</tr></thead>
              <tbody>
                {rows.length === 0 && <tr><td colSpan={6} className={`${td} text-center ${muted(isDark)}`}>No copy traders yet.</td></tr>}
                {rows.map((r) => (
                  <tr key={r.id} className="border-t border-gray-700/40">
                    <td className={`${td} font-mono text-xs`}>{r.trader}</td>
                    <td className={td}>{r.display_name || '—'}</td>
                    <td className={`${td} ${muted(isDark)}`}>{r.fee_bps} bps</td>
                    <td className={`${td} ${muted(isDark)}`}>{r.max_copiers === 0 ? 'unlimited' : r.max_copiers}</td>
                    <td className={td}>{badge(r.status)}</td>
                    <td className={td}>{lifecycle(r, tradingControlAPI.copyTraders)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </PageShell>
  );
};
