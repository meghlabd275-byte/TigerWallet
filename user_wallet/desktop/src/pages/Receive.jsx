import React, { useState, useEffect } from 'react';
import { api } from '../services/api';

function Receive() {
  const [wallets, setWallets] = useState([]);
  const [selected, setSelected] = useState('');
  const [loading, setLoading] = useState(true);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    api.getWallets()
      .then((data) => {
        const list = data.wallets || [];
        setWallets(list);
        if (list.length > 0) setSelected(list[0].id || list[0].wallet_id || '');
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  const activeWallet = wallets.find((w) => (w.id || w.wallet_id) === selected) || wallets[0];
  const address = activeWallet ? activeWallet.address : '';

  const copyAddress = async () => {
    if (!address) return;
    try {
      await navigator.clipboard.writeText(address);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setError('Copy failed — select and copy manually');
    }
  };

  return (
    <div className="receive-page">
      <h1>Receive</h1>

      {loading ? (
        <p>Loading...</p>
      ) : wallets.length === 0 ? (
        <p>No wallets yet. Create one first to receive funds.</p>
      ) : (
        <>
          <label>Select wallet</label>
          <select value={selected} onChange={(e) => setSelected(e.target.value)}>
            {wallets.map((w, idx) => (
              <option key={w.id || idx} value={w.id || w.wallet_id || ''}>
                {w.label} (Chain #{w.chain_id})
              </option>
            ))}
          </select>

          {activeWallet && (
            <div className="receive-card">
              <h3>{activeWallet.label}</h3>
              <p className="network">Chain #{activeWallet.chain_id}</p>

              <div className="qr-placeholder">
                <span>QR</span>
              </div>
              <p className="address mono">{address}</p>

              {error && <div className="error">{error}</div>}
              <button onClick={copyAddress} disabled={!address}>
                {copied ? '✓ Copied' : '📋 Copy Address'}
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}

export default Receive;
