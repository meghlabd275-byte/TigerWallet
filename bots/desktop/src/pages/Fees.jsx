import React, { useEffect, useState, useCallback } from 'react';
import api from '../services/bot-api';

export default function Fees() {
  const [fees, setFees] = useState([]);
  const [error, setError] = useState(null);
  const [notice, setNotice] = useState(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(() => {
    setLoading(true); setError(null);
    api.getFeeConfigs()
      .then(d => setFees((d && (d.fees || d.configs)) || []))
      .catch(e => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  useEffect(load, [load]);

  const toggle = async (fee) => {
    setError(null); setNotice(null);
    try {
      await api.updateFeeConfig({
        id: fee.id,
        name: fee.name,
        percentage: fee.percentage,
        enabled: !fee.enabled,
      });
      setNotice(`${fee.name || fee.id} ${fee.enabled ? 'disabled' : 'enabled'}`);
      load();
    } catch (e) {
      setError(e.message);
    }
  };

  if (loading) return <div>Loading…</div>;

  return (
    <div className="fees-page">
      <h1>Fee configuration</h1>
      {error && <p className="error-banner">{error}</p>}
      {notice && <p className="success-banner">{notice}</p>}
      {fees.length === 0 ? <p className="empty-state">No fee tiers configured.</p> : (
        <table className="data-table">
          <thead><tr><th>Name</th><th>Percentage</th><th>Status</th><th>Actions</th></tr></thead>
          <tbody>
            {fees.map(f => (
              <tr key={f.id}>
                <td>{f.name || f.id}</td>
                <td>{f.percentage != null ? `${f.percentage}%` : '—'}</td>
                <td>{f.enabled ? 'enabled' : 'disabled'}</td>
                <td className="action-row">
                  <button onClick={() => toggle(f)}>{f.enabled ? 'Disable' : 'Enable'}</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
