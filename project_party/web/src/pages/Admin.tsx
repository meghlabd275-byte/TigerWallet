// Admin — WL-ProjectParty listing-admin console: token review (approve /
// reject / verify-contract / feature) and fee-payment on-chain verification.
// All actions hit the real admin-gated WL backend routes; every button shows
// the backend's real error (403 etc.) when the caller lacks the admin role.
import React, { useCallback, useEffect, useState } from 'react';
import { api } from '../services/api';
import { useAuth } from '../contexts/AuthContext';

type Tab = 'tokens' | 'fees';

interface Token {
  id: string;
  name: string;
  symbol: string;
  status: string;
  chain_id?: number;
  contract_address?: string;
  contract_verified?: boolean;
  is_featured?: boolean;
}

interface FeePayment {
  id: string;
  token_id?: string;
  user_id?: string;
  amount: string;
  currency: string;
  payment_method?: string;
  tx_hash?: string;
  status: string;
  created_at: string;
}

const REVIEWABLE = new Set(['draft', 'submitted', 'pending', 'in_review']);

export default function Admin() {
  const { isAdmin } = useAuth();
  const [tab, setTab] = useState<Tab>('tokens');
  const [tokens, setTokens] = useState<Token[]>([]);
  const [payments, setPayments] = useState<FeePayment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [busy, setBusy] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [t, p] = await Promise.all([
        api.listTokens(),
        api.listFeePayments()
      ]);
      setTokens(t.tokens || []);
      setPayments(p.fee_payments || []);
    } catch (e: any) {
      setError(e?.message || 'Failed to load admin data');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (isAdmin) load(); else setLoading(false);
  }, [isAdmin, load]);

  if (!isAdmin) {
    return (
      <div>
        <h1>Admin</h1>
        <div className="alert error">This console requires an admin or listing_admin account.</div>
      </div>
    );
  }

  const act = async (key: string, fn: () => Promise<any>, okMsg: string) => {
    setBusy(key);
    setError('');
    setNotice('');
    try {
      const res = await fn();
      setNotice(res?.message ? `${okMsg}: ${res.message}` : okMsg);
      await load();
    } catch (e: any) {
      setError(e?.message || `${okMsg} failed`);
    } finally {
      setBusy(null);
    }
  };

  const reviewable = tokens.filter(t => REVIEWABLE.has(t.status));
  const pendingFees = payments.filter(p => p.status === 'pending');

  return (
    <div>
      <h1>Admin Console</h1>
      <div className="tabs" style={{ display: 'flex', gap: '0.5rem', marginBottom: '1rem' }}>
        <button className={tab === 'tokens' ? '' : 'secondary'} onClick={() => setTab('tokens')}>
          Token Review ({reviewable.length})
        </button>
        <button className={tab === 'fees' ? '' : 'secondary'} onClick={() => setTab('fees')}>
          Fee Verification ({pendingFees.length} pending)
        </button>
      </div>
      {error && <div className="alert error">{error}</div>}
      {notice && <div className="alert success">{notice}</div>}
      {loading ? (
        <p className="muted">Loading…</p>
      ) : tab === 'tokens' ? (
        <>
          <h2>Tokens awaiting review</h2>
          {reviewable.length === 0 ? (
            <p className="muted">No tokens awaiting review.</p>
          ) : (
            <table>
              <thead>
                <tr>
                  <th>Symbol</th><th>Name</th><th>Status</th><th>Chain</th>
                  <th>Contract</th><th>Verified</th><th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {reviewable.map(t => (
                  <tr key={t.id}>
                    <td><span className="badge accent">{t.symbol}</span></td>
                    <td>{t.name}</td>
                    <td><span className="badge">{t.status}</span></td>
                    <td>{t.chain_id ?? '—'}</td>
                    <td style={{ maxWidth: 160, overflow: 'hidden', textOverflow: 'ellipsis' }}>
                      {t.contract_address || '—'}
                    </td>
                    <td>{t.contract_verified ? '✅' : '—'}</td>
                    <td style={{ display: 'flex', gap: '0.4rem', flexWrap: 'wrap' }}>
                      <button disabled={busy === `ap-${t.id}`}
                        onClick={() => act(`ap-${t.id}`, () => api.approveToken(t.id), 'Token approved')}>
                        Approve
                      </button>
                      <button className="secondary" disabled={busy === `rj-${t.id}`}
                        onClick={() => {
                          const reason = window.prompt('Rejection reason (optional):') || undefined;
                          act(`rj-${t.id}`, () => api.rejectToken(t.id, reason), 'Token rejected');
                        }}>
                        Reject
                      </button>
                      {t.contract_address && (
                        <button className="secondary" disabled={busy === `vc-${t.id}`}
                          onClick={() => act(`vc-${t.id}`, () => api.verifyTokenContract(t.id), 'Contract verified on-chain')}>
                          Verify Contract
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
          <h2 style={{ marginTop: '1.5rem' }}>All listed tokens</h2>
          {tokens.filter(t => !REVIEWABLE.has(t.status)).length === 0 ? (
            <p className="muted">No listed tokens yet.</p>
          ) : (
            <table>
              <thead>
                <tr><th>Symbol</th><th>Name</th><th>Status</th><th>Featured</th><th>Actions</th></tr>
              </thead>
              <tbody>
                {tokens.filter(t => !REVIEWABLE.has(t.status)).map(t => (
                  <tr key={t.id}>
                    <td><span className="badge accent">{t.symbol}</span></td>
                    <td>{t.name}</td>
                    <td><span className={`badge ${t.status === 'listed' || t.status === 'active' ? 'active' : 'error'}`}>{t.status}</span></td>
                    <td>{t.is_featured ? '⭐' : '—'}</td>
                    <td>
                      <button className="secondary" disabled={busy === `ft-${t.id}`}
                        onClick={() => act(`ft-${t.id}`, () => api.toggleFeatured(t.id), 'Featured toggled')}>
                        Toggle Featured
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </>
      ) : (
        <>
          <h2>Fee payments</h2>
          {payments.length === 0 ? (
            <p className="muted">No fee payments recorded.</p>
          ) : (
            <table>
              <thead>
                <tr>
                  <th>Created</th><th>Amount</th><th>Method</th><th>Tx Hash</th><th>Status</th><th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {payments.map(p => (
                  <tr key={p.id}>
                    <td>{new Date(p.created_at).toLocaleString()}</td>
                    <td>{p.amount} {p.currency}</td>
                    <td>{p.payment_method || '—'}</td>
                    <td style={{ maxWidth: 160, overflow: 'hidden', textOverflow: 'ellipsis' }}>
                      {p.tx_hash || '—'}
                    </td>
                    <td>
                      <span className={`badge ${p.status === 'completed' ? 'active' : 'error'}`}>{p.status}</span>
                    </td>
                    <td>
                      {p.status === 'pending' && p.tx_hash && (
                        <button disabled={busy === `vf-${p.id}`}
                          onClick={() => act(`vf-${p.id}`, () => api.verifyFeePayment(p.id), 'Payment verified on-chain')}>
                          Verify On-Chain
                        </button>
                      )}
                      {p.status === 'pending' && !p.tx_hash && <span className="muted">no tx hash</span>}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </>
      )}
    </div>
  );
}
