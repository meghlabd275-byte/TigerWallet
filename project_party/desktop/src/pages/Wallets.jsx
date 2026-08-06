import React, { useState, useEffect } from 'react';

const API_URL = 'http://localhost:8105/api/v1';

function Wallets() {
  const [wallets, setWallets] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [formData, setFormData] = useState({ name: '', network: 'ethereum', tokens: ['ethereum'] });

  useEffect(() => {
    loadWallets();
  }, []);

  const loadWallets = () => {
    fetch(`${API_URL}/wallet/wallets`)
      .then(res => res.json())
      .then(data => {
        setWallets(data.wallets || []);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  };

  const handleCreate = async (e) => {
    e.preventDefault();
    try {
      await fetch(`${API_URL}/wallet/wallets`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(formData)
      });
      setShowCreate(false);
      loadWallets();
    } catch (err) {
      alert('Failed to create wallet');
    }
  };

  return (
    <div className="wallets-page">
      <header className="page-header">
        <h1>My Wallets</h1>
        <button onClick={() => setShowCreate(!showCreate)}>+ Add Wallet</button>
      </header>

      {showCreate && (
        <form className="create-form" onSubmit={handleCreate}>
          <input 
            placeholder="Wallet Name" 
            value={formData.name} 
            onChange={e => setFormData({...formData, name: e.target.value})} 
            required 
          />
          <select value={formData.network} onChange={e => setFormData({...formData, network: e.target.value})}>
            <option value="ethereum">Ethereum</option>
            <option value="bsc">BNB Chain</option>
            <option value="polygon">Polygon</option>
          </select>
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
              <h3>{wallet.name}</h3>
              <p className="network">{wallet.network}</p>
              <p className="address">{wallet.address}</p>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default Wallets;
