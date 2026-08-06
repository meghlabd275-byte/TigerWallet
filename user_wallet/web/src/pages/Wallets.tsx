// Wallets Page
import React, { useState, useEffect } from 'react';
import { api } from '../../services/api';

export default function Wallets() {
  const [wallets, setWallets] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [name, setName] = useState('');
  const [walletType, setWalletType] = useState('ethereum');
  const [networks, setNetworks] = useState<string[]>(['ethereum']);

  useEffect(() => {
    loadWallets();
  }, []);

  const loadWallets = () => {
    api.getWallets().then(data => {
      setWallets(data.wallets || []);
      setLoading(false);
    }).catch(() => setLoading(false));
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await api.createWallet(name, walletType, networks);
      setShowCreate(false);
      setName('');
      loadWallets();
    } catch (err) {
      alert('Failed to create wallet');
    }
  };

  return (
    <div className="wallets-page">
      <header className="page-header">
        <h1>My Wallets</h1>
        <button onClick={() => setShowCreate(!showCreate)}>+ Create Wallet</button>
      </header>

      {showCreate && (
        <div className="create-form">
          <h3>Create New Wallet</h3>
          <form onSubmit={handleCreate}>
            <input placeholder="Wallet Name" value={name} onChange={e => setName(e.target.value)} required />
            <select value={walletType} onChange={e => setWalletType(e.target.value)}>
              <option value="ethereum">Ethereum</option>
              <option value="bsc">BNB Chain</option>
              <option value="polygon">Polygon</option>
              <option value="solana">Solana</option>
            </select>
            <button type="submit">Create</button>
          </form>
        </div>
      )}

      {loading ? <p>Loading...</p> : wallets.length === 0 ? (
        <p>No wallets yet. Create one to get started!</p>
      ) : (
        <div className="wallets-grid">
          {wallets.map((wallet: any) => (
            <div key={wallet.id} className="wallet-card">
              <h3>{wallet.name}</h3>
              <p className="wallet-type">{wallet.wallet_type}</p>
              <p className="wallet-address">{wallet.address}</p>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
