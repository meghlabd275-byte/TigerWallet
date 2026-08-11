import React, { useState, useEffect } from 'react';
import { api } from '../services/api';

const CHAIN_IDS = { ethereum: 1, bsc: 56, polygon: 137 };

function Wallets() {
  const [wallets, setWallets] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [formData, setFormData] = useState({ name: '', network: 'ethereum', password: '' });
  const [newMnemonic, setNewMnemonic] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    loadWallets();
  }, []);

  const loadWallets = () => {
    api.getWallets()
      .then((data) => {
        setWallets(data.wallets || []);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  };

  const handleCreate = async (e) => {
    e.preventDefault();
    setError('');
    if (formData.password.length < 8) {
      setError('Password must be at least 8 characters');
      return;
    }
    try {
      const w = await api.createWallet({
        label: formData.name,
        password: formData.password,
        chainId: CHAIN_IDS[formData.network] || 1,
      });
      if (w.mnemonic) setNewMnemonic(w.mnemonic);
      setShowCreate(false);
      setFormData({ name: '', network: 'ethereum', password: '' });
      loadWallets();
    } catch (err) {
      setError(err.message || 'Failed to create wallet');
    }
  };

  return (
    <div className="wallets-page">
      <header className="page-header">
        <h1>My Wallets</h1>
        <button onClick={() => setShowCreate(!showCreate)}>+ Add Wallet</button>
      </header>

      {newMnemonic && (
        <div className="mnemonic-warning">
          <h3>Save your recovery phrase</h3>
          <p>Shown only once. Store it securely — it controls your funds.</p>
          <code>{newMnemonic}</code>
          <button onClick={() => setNewMnemonic('')}>I&apos;ve saved it</button>
        </div>
      )}

      {showCreate && (
        <form className="create-form" onSubmit={handleCreate}>
          {error && <div className="error">{error}</div>}
          <input
            placeholder="Wallet Name"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            required
          />
          <select value={formData.network} onChange={(e) => setFormData({ ...formData, network: e.target.value })}>
            <option value="ethereum">Ethereum</option>
            <option value="bsc">BNB Chain</option>
            <option value="polygon">Polygon</option>
          </select>
          <input
            type="password"
            placeholder="Password (min 8 chars)"
            value={formData.password}
            onChange={(e) => setFormData({ ...formData, password: e.target.value })}
            required
            minLength={8}
          />
          <button type="submit">Create</button>
        </form>
      )}

      {loading ? (
        <p>Loading...</p>
      ) : wallets.length === 0 ? (
        <p>No wallets yet. Create one to get started!</p>
      ) : (
        <div className="wallets-grid">
          {wallets.map((wallet, idx) => (
            <div key={idx} className="wallet-card">
              <h3>{wallet.label}</h3>
              <p className="network">Chain #{wallet.chain_id}</p>
              <p className="address">{wallet.address}</p>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default Wallets;
