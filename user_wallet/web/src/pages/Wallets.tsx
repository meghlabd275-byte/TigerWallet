// Wallets Page
import React, { useState, useEffect } from 'react';
import { api, WalletRecord, BalanceResult } from '../services/api';

export default function Wallets() {
  const [wallets, setWallets] = useState<WalletRecord[]>([]);
  const [balances, setBalances] = useState<Record<string, BalanceResult | null>>({});
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [name, setName] = useState('');
  const [walletType, setWalletType] = useState('ethereum');
  const [password, setPassword] = useState('');
  const [newMnemonic, setNewMnemonic] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    loadWallets();
  }, []);

  const loadWallets = () => {
    api.getWallets().then((data) => {
      setWallets(data.wallets || []);
      setLoading(false);
      // Fan out a real balance fetch per wallet (GET /wallets/:id/balance).
      (data.wallets || []).forEach((w) => {
        api.getBalance(w.id).then((b) => setBalances((prev) => ({ ...prev, [w.id]: b }))).catch(() => {
          setBalances((prev) => ({ ...prev, [w.id]: null }));
        });
      });
    }).catch(() => setLoading(false));
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    if (password.length < 8) {
      setError('Password must be at least 8 characters');
      return;
    }
    try {
      const w = await api.createWalletTyped({
        label: name,
        password,
        chainId: walletType === 'ethereum' ? 1 : walletType === 'bsc' ? 56 : walletType === 'polygon' ? 137 : 1,
      });
      if (w.mnemonic) setNewMnemonic(w.mnemonic);
      setShowCreate(false);
      setName('');
      setPassword('');
      loadWallets();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to create wallet');
    }
  };

  return (
    <div className="wallets-page">
      <header className="page-header">
        <h1>My Wallets</h1>
        <button onClick={() => setShowCreate(!showCreate)}>+ Create Wallet</button>
      </header>

      {newMnemonic && (
        <div className="mnemonic-warning">
          <h3>Save your recovery phrase</h3>
          <p>This is shown only once. Store it securely — it controls your funds.</p>
          <code>{newMnemonic}</code>
          <button onClick={() => setNewMnemonic('')}>I've saved it</button>
        </div>
      )}

      {showCreate && (
        <div className="create-form">
          <h3>Create New Wallet</h3>
          {error && <div className="error">{error}</div>}
          <form onSubmit={handleCreate}>
            <input placeholder="Wallet Name" value={name} onChange={(e) => setName(e.target.value)} required />
            <select value={walletType} onChange={(e) => setWalletType(e.target.value)}>
              <option value="ethereum">Ethereum</option>
              <option value="bsc">BNB Chain</option>
              <option value="polygon">Polygon</option>
            </select>
            <input
              type="password"
              placeholder="Password (min 8 chars)"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              minLength={8}
            />
            <button type="submit">Create</button>
          </form>
        </div>
      )}

      {loading ? <p>Loading...</p> : wallets.length === 0 ? (
        <p>No wallets yet. Create one to get started!</p>
      ) : (
        <div className="wallets-grid">
          {wallets.map((wallet) => {
            const bal = balances[wallet.id];
            return (
              <div key={wallet.id} className="wallet-card">
                <h3>{wallet.label || 'Untitled'}</h3>
                <p className="wallet-type">Chain #{wallet.chain_id}{bal ? ` · ${bal.symbol}` : ''}</p>
                <p className="wallet-address">{wallet.address}</p>
                <p className="wallet-type">
                  Balance: {bal ? `${bal.balance_f.toFixed(6)} ${bal.symbol}` : '…'}
                </p>
                {wallet.created_at && (
                  <p className="wallet-type">Created {new Date(wallet.created_at).toLocaleString()}</p>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
