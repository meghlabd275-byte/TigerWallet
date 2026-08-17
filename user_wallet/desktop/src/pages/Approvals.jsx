import React, { useState, useEffect } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { api } from '../services/api';

const CHAIN_OPTIONS = [
  { value: 'ethereum', label: 'Ethereum', id: 1 },
  { value: 'bsc', label: 'BNB Chain', id: 56 },
  { value: 'polygon', label: 'Polygon', id: 137 },
];
const CHAIN_IDS = { ethereum: 1, bsc: 56, polygon: 137 };

function Approvals() {
  const { theme } = useTheme();
  const isDark = theme === 'dark';

  const [wallets, setWallets] = useState([]);
  const [walletId, setWalletId] = useState('');
  const [network, setNetwork] = useState('ethereum');
  const [approvals, setApprovals] = useState([]);
  const [loading, setLoading] = useState(true);
  const [fetching, setFetching] = useState(false);
  const [error, setError] = useState('');
  const [info, setInfo] = useState('');
  const [revokingId, setRevokingId] = useState(null);

  useEffect(() => {
    let alive = true;
    api.getWallets()
      .then((data) => {
        if (!alive) return;
        const list = data.wallets || [];
        setWallets(list);
        if (list.length > 0) setWalletId(list[0].id || list[0].wallet_id || '');
        setLoading(false);
      })
      .catch(() => { if (alive) setLoading(false); });
    return () => { alive = false; };
  }, []);

  const loadApprovals = async () => {
    setError('');
    setInfo('');
    const wallet = wallets.find((w) => (w.id || w.wallet_id) === walletId);
    if (!wallet) { setError('Select a wallet'); return; }
    setFetching(true);
    try {
      const data = await api.getApprovals(wallet.address, CHAIN_IDS[network] || 1);
      setApprovals(data.approvals || []);
    } catch (err) {
      setError(err.message || 'Failed to load approvals');
    } finally {
      setFetching(false);
    }
  };

  useEffect(() => {
    if (walletId) loadApprovals();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [walletId, network]);

  const handleRevoke = async (a) => {
    setError('');
    setInfo('');
    const id = a.id || a.approval_id || a.approvalId;
    setRevokingId(id);
    try {
      await api.revokeApproval({ approvalId: id });
      setInfo('Approval revoked.');
      loadApprovals();
    } catch (err) {
      setError(err.message || 'Failed to revoke approval');
    } finally {
      setRevokingId(null);
    }
  };

  return (
    <div className="wallets-page">
      <header className="page-header">
        <h1>Token Approvals</h1>
      </header>

      {loading ? (
        <p>Loading...</p>
      ) : wallets.length === 0 ? (
        <p>No wallets yet. Create one first to view approvals.</p>
      ) : (
        <>
          <div className="send-form" style={{ maxWidth: '600px' }}>
            {error && <div className="error">{error}</div>}
            {info && <div className="success-banner"><h3 style={{ color: isDark ? '#4CAF50' : 'var(--accent)' }}>✓ {info}</h3></div>}

            <label>Wallet</label>
            <select value={walletId} onChange={(e) => setWalletId(e.target.value)}>
              {wallets.map((w, idx) => (
                <option key={w.id || idx} value={w.id || w.wallet_id || ''}>
                  {w.label} — {w.address ? w.address.slice(0, 10) : ''}…
                </option>
              ))}
            </select>

            <label>Chain</label>
            <select value={network} onChange={(e) => setNetwork(e.target.value)}>
              {CHAIN_OPTIONS.map((c) => <option key={c.value} value={c.value}>{c.label}</option>)}
            </select>

            <div className="send-actions">
              <button type="button" onClick={loadApprovals} disabled={fetching}>
                {fetching ? 'Loading…' : 'Reload Approvals'}
              </button>
            </div>
          </div>

          {fetching ? (
            <p>Loading...</p>
          ) : approvals.length === 0 ? (
            <p style={{ marginTop: '16px' }}>No token approvals found for this wallet on this chain.</p>
          ) : (
            <table className="transactions-table" style={{ marginTop: '20px' }}>
              <thead>
                <tr>
                  <th>Token</th>
                  <th>Spender</th>
                  <th>Amount</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {approvals.map((a, idx) => {
                  const id = a.id || a.approval_id || a.approvalId || idx;
                  return (
                    <tr key={id}>
                      <td className="mono">{a.token || a.token_address || a.contract_address || '—'}</td>
                      <td className="mono">{(a.spender || a.spender_address || '').slice(0, 14)}…</td>
                      <td>{a.amount !== undefined ? a.amount : '∞'}</td>
                      <td>
                        <button
                          onClick={() => handleRevoke(a)}
                          disabled={revokingId === id}
                        >
                          {revokingId === id ? 'Revoking…' : 'Revoke'}
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          )}
        </>
      )}
    </div>
  );
}

export default Approvals;
