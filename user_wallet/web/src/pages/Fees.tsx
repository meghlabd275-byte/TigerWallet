import React, { useEffect, useState } from 'react';
import api from '../services/api';

interface FeeTier {
  tier_name: string;
  fee_type: string;
  rate_basis_points: string;
  min_amount: string;
  max_amount: string;
  chain_id: number | null;
}

interface FeeTx {
  fee_type: string;
  currency: string;
  amount: string;
  chain_id: number | null;
  created_at: string;
}

function bpsToPercent(bps: string): string {
  const n = parseFloat(bps);
  if (isNaN(n)) return bps;
  return (n / 100).toFixed(2) + '%';
}

export default function Fees() {
  const [fees, setFees] = useState<FeeTier[]>([]);
  const [txs, setTxs] = useState<FeeTx[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let mounted = true;
    (async () => {
      try {
        const [feeRes, txRes] = await Promise.all([
          api.getPublicFees(),
          api.getPublicFeeTransactions(),
        ]);
        if (!mounted) return;
        setFees(feeRes.fees || []);
        setTxs(txRes.transactions || []);
      } catch (e: unknown) {
        if (!mounted) return;
        setError(e instanceof Error ? e.message : 'Failed to load fee data');
      } finally {
        if (mounted) setLoading(false);
      }
    })();
    return () => { mounted = false; };
  }, []);

  if (loading) return <div className="page-container"><p>Loading fee schedule…</p></div>;
  if (error) return <div className="page-container"><p className="error-text">{error}</p></div>;

  return (
    <div className="page-container">
      <h2>Fee Transparency</h2>
      <p className="muted">All fees are published on-chain and visible here before you transact.</p>

      <div className="card" style={{ marginBottom: 24 }}>
        <h3>Active Fee Tiers</h3>
        {fees.length === 0 ? (
          <p className="muted">No fee tiers configured. Trading is currently fee-free.</p>
        ) : (
          <div style={{ overflowX: 'auto' }}>
            <table className="data-table">
              <thead>
                <tr>
                  <th>Tier</th>
                  <th>Type</th>
                  <th>Rate</th>
                  <th>Min</th>
                  <th>Max</th>
                  <th>Chain</th>
                </tr>
              </thead>
              <tbody>
                {fees.map((f, i) => (
                  <tr key={i}>
                    <td>{f.tier_name}</td>
                    <td><span className="badge">{f.fee_type}</span></td>
                    <td>{bpsToPercent(f.rate_basis_points)}</td>
                    <td>{f.min_amount}</td>
                    <td>{f.max_amount || '—'}</td>
                    <td>{f.chain_id ?? 'All'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <div className="card">
        <h3>Recent Settled Fee Transactions</h3>
        {txs.length === 0 ? (
          <p className="muted">No settled fee transactions yet.</p>
        ) : (
          <div style={{ overflowX: 'auto' }}>
            <table className="data-table">
              <thead>
                <tr>
                  <th>Type</th>
                  <th>Amount</th>
                  <th>Currency</th>
                  <th>Chain</th>
                  <th>Time</th>
                </tr>
              </thead>
              <tbody>
                {txs.map((t, i) => (
                  <tr key={i}>
                    <td><span className="badge">{t.fee_type}</span></td>
                    <td>{t.amount}</td>
                    <td>{t.currency}</td>
                    <td>{t.chain_id ?? '—'}</td>
                    <td>{new Date(t.created_at).toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
