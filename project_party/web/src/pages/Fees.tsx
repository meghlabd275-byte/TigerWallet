// Fees Page - ProjectParty
import React, { useState, useEffect } from 'react';
import { api } from '../services/api';

const FEATURES = [
  { key: 'featured', label: 'Featured placement' },
  { key: 'audit', label: 'Audit required' },
  { key: 'kyc', label: 'KYC verification' }
];

const PAYMENT_METHODS = ['USDT', 'USDC', 'ETH', 'BNB', 'CARD'];

export default function Fees() {
  const [fees, setFees] = useState<Record<string, number> | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [listingType, setListingType] = useState('basic');
  const [selected, setSelected] = useState<string[]>([]);
  const [calc, setCalc] = useState<any>(null);
  const [calcLoading, setCalcLoading] = useState(false);

  const [payAmount, setPayAmount] = useState('');
  const [payMethod, setPayMethod] = useState('USDT');
  const [paySubmitting, setPaySubmitting] = useState(false);
  const [payResult, setPayResult] = useState<any>(null);
  const [msg, setMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  useEffect(() => {
    (async () => {
      setLoading(true);
      setError('');
      try {
        const data = await api.getListingFees();
        setFees(data);
      } catch (e: any) {
        setError(e.message || 'Failed to load fees');
      }
      setLoading(false);
    })();
  }, []);

  const toggleFeature = (key: string) => {
    setSelected(prev => prev.includes(key) ? prev.filter(f => f !== key) : [...prev, key]);
  };

  const calculate = async (e: React.FormEvent) => {
    e.preventDefault();
    setCalcLoading(true);
    setMsg(null);
    try {
      const data = await api.calculateFees(listingType, selected);
      setCalc(data);
      setPayAmount(String(data.total_fee));
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Failed to calculate fees' });
    }
    setCalcLoading(false);
  };

  const pay = async (e: React.FormEvent) => {
    e.preventDefault();
    setPaySubmitting(true);
    setMsg(null);
    try {
      const data = await api.payFees(payAmount, payMethod);
      setPayResult(data);
      setMsg({ type: 'success', text: 'Payment processed.' });
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Payment failed' });
    }
    setPaySubmitting(false);
  };

  return (
    <div className="page">
      <div className="page-header"><h1>Fees</h1></div>
      <p className="subtitle">View the fee schedule, calculate listing fees, and process payments.</p>

      {msg && <div className={`alert ${msg.type}`}>{msg.text}</div>}
      {error && <div className="alert error">{error}</div>}

      <section>
        <div className="section-title"><h2>Fee Schedule</h2></div>
        {loading ? <div className="state">Loading fees...</div> : !fees || Object.keys(fees).length === 0 ? (
          <div className="state">No data available</div>
        ) : (
          <table>
            <thead><tr><th>Fee Type</th><th>Amount (USD)</th></tr></thead>
            <tbody>
              {Object.entries(fees).map(([k, v]) => (
                <tr key={k}><td>{k.replace(/_/g, ' ')}</td><td>${v}</td></tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      <div className="two-col">
        <section>
          <div className="section-title"><h2>Calculate Fees</h2></div>
          <form onSubmit={calculate}>
            <div className="form-grid">
              <div className="form-field">
                <label>Listing Type</label>
                <select value={listingType} onChange={e => setListingType(e.target.value)}>
                  <option value="basic">Basic listing</option>
                  <option value="launchpad">Launchpad</option>
                </select>
              </div>
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem', marginBottom: '0.75rem' }}>
              {FEATURES.map(f => (
                <label key={f.key} style={{ display: 'flex', gap: '0.5rem', alignItems: 'center', color: 'var(--text-primary)' }}>
                  <input type="checkbox" checked={selected.includes(f.key)} onChange={() => toggleFeature(f.key)} />
                  {f.label}
                </label>
              ))}
            </div>
            <button type="submit" disabled={calcLoading}>{calcLoading ? 'Calculating...' : 'Calculate'}</button>
          </form>
          {calc && (
            <div className="stat-card" style={{ marginTop: '0.85rem' }}>
              <h3>Estimated Total</h3>
              <p>${calc.total_fee} {calc.currency}</p>
            </div>
          )}
        </section>

        <section>
          <div className="section-title"><h2>Pay Fees</h2></div>
          <form onSubmit={pay}>
            <div className="form-grid">
              <div className="form-field">
                <label>Amount</label>
                <input value={payAmount} onChange={e => setPayAmount(e.target.value)} placeholder="e.g. 1500" required />
              </div>
              <div className="form-field">
                <label>Payment Method</label>
                <select value={payMethod} onChange={e => setPayMethod(e.target.value)}>
                  {PAYMENT_METHODS.map(m => <option key={m} value={m}>{m}</option>)}
                </select>
              </div>
            </div>
            <button type="submit" disabled={paySubmitting}>{paySubmitting ? 'Processing...' : 'Pay'}</button>
          </form>
          {payResult && (
            <div className="card-row" style={{ marginTop: '0.85rem' }}>
              <span>Transaction</span><span title={payResult.transaction_id}>{String(payResult.transaction_id).slice(0, 12)}...</span>
            </div>
          )}
        </section>
      </div>
    </div>
  );
}
