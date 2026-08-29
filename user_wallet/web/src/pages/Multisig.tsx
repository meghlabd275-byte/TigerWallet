// Multisig Page — create multisig wallets, list them, create/sign/execute
// multisig transactions. All calls go through the wallet_api multisig proxy
// (/wallet/multisig/* -> MasterWallet). Real backend state, fail-closed
// error display, no fabricated data.
import React from 'react';
import { api } from '../services/api';
export default function Multisig() {
  const [wallets, setWallets] = React.useState<any[]>([]);
  const [txs, setTxs] = React.useState<any[]>([]);
  const [name, setName] = React.useState('');
  const [owners, setOwners] = React.useState('');
  const [threshold, setThreshold] = React.useState('2');
  const [txWalletId, setTxWalletId] = React.useState('');
  const [txTo, setTxTo] = React.useState('');
  const [txValue, setTxValue] = React.useState('');
  const [txData, setTxData] = React.useState('');
  const [error, setError] = React.useState('');
  const [notice, setNotice] = React.useState('');
  const loadWallets = () => {
    api.listMultisigWallets()
      .then((d) => setWallets(d.multisig_wallets || d.wallets || []))
      .catch((e) => setError(`Multisig unavailable: ${e.message}`));
  };
  const loadTxs = (walletId: string) => {
    if (!walletId) return;
    api.listMultisigTransactions(walletId)
      .then((d) => setTxs(d.transactions || d.multisig_transactions || []))
      .catch((e) => setError(`Failed to load transactions: ${e.message}`));
  };
  React.useEffect(() => { loadWallets(); }, []);
  const createWallet = async () => {
    setError(''); setNotice('');
    const ownerList = owners.split(',').map((s) => s.trim()).filter(Boolean);
    const thr = parseInt(threshold, 10) || 0;
    if (!name || !ownerList.length || thr < 1) {
      setError('Enter name, owners and threshold');
      return;
    }
    try {
      await api.createMultisigWallet({ name, owners: ownerList, threshold: thr, chain_id: 1 });
      setNotice('Multisig wallet created');
      loadWallets();
    } catch (e: any) {
      setError(`Create failed: ${e.message}`);
    }
  };
  const createTx = async () => {
    setError(''); setNotice('');
    if (!txWalletId || !txTo || !txValue) {
      setError('Enter wallet id, to address and value');
      return;
    }
    try {
      await api.createMultisigTransaction(txWalletId, { to_address: txTo, value: txValue, data: txData });
      setNotice('Multisig transaction created — pending signatures');
      loadTxs(txWalletId);
    } catch (e: any) {
      setError(`Create tx failed: ${e.message}`);
    }
  };
  const txAction = async (txId: string, action: 'sign' | 'execute') => {
    setError(''); setNotice('');
    try {
      const r = action === 'sign'
        ? await api.signMultisigTransaction(txId)
        : await api.executeMultisigTransaction(txId);
      setNotice(action === 'execute'
        ? `Transaction submitted to the blockchain network: ${r.tx_hash || r.status || 'broadcast'}`
        : 'Multisig transaction signed');
      loadTxs(txWalletId);
    } catch (e: any) {
      setError(`${action} failed: ${e.message}`);
    }
  };
  return (
    <div>
      <header className="page-header"><h1>Multisig</h1></header>
      {error && <div className="error">{error}</div>}
      {notice && <div className="success-banner">{notice}</div>}
      <div style={{ marginBottom: 24 }}>
        <h2>Create Multisig</h2>
        <div className="form-group">
          <label>Name</label>
          <input value={name} onChange={(e) => setName(e.target.value)} placeholder="Treasury" />
        </div>
        <div className="form-group">
          <label>Owners (comma-separated)</label>
          <input value={owners} onChange={(e) => setOwners(e.target.value)} placeholder="0xabc…, 0xdef…" />
        </div>
        <div className="form-group">
          <label>Threshold</label>
          <input value={threshold} onChange={(e) => setThreshold(e.target.value)} placeholder="2" />
        </div>
        <button className="primary-btn" onClick={createWallet}>Create Multisig</button>
      </div>
      <div style={{ marginBottom: 24 }}>
        <h2>Wallets</h2>
        {wallets.length === 0 ? (
          <p className="empty-state">No multisig wallets</p>
        ) : (
          <div className="record-list">
            {wallets.map((w, i) => (
              <div key={i} className="record-item">
                {w.name || w.id} · {w.id} · chain {w.chain_id} · {w.threshold}-of-{(w.owners || []).length}
              </div>
            ))}
          </div>
        )}
      </div>
      <div style={{ marginBottom: 24 }}>
        <h2>Transactions</h2>
        <div className="form-group">
          <label>Multisig wallet ID</label>
          <input value={txWalletId} onChange={(e) => setTxWalletId(e.target.value)} placeholder="wallet id" />
        </div>
        <div className="form-group">
          <label>To address</label>
          <input value={txTo} onChange={(e) => setTxTo(e.target.value)} placeholder="0x…" />
        </div>
        <div className="form-group">
          <label>Value (wei)</label>
          <input value={txValue} onChange={(e) => setTxValue(e.target.value)} placeholder="0" />
        </div>
        <div className="form-group">
          <label>Data (hex, optional)</label>
          <input value={txData} onChange={(e) => setTxData(e.target.value)} placeholder="0x" />
        </div>
        <div className="action-row">
          <button className="primary-btn" onClick={createTx}>Create Transaction</button>
          <button className="secondary-btn" onClick={() => loadTxs(txWalletId)}>Load Transactions</button>
        </div>
        {txs.length === 0 ? (
          <p className="empty-state">No multisig transactions</p>
        ) : (
          <div className="record-list">
            {txs.map((t, i) => (
              <div key={i} className="record-item">
                <span>{t.id} → {t.to_address} · {t.status}</span>
                <span className="action-row">
                  <button className="secondary-btn" onClick={() => txAction(t.id, 'sign')}>Sign</button>
                  <button className="secondary-btn" onClick={() => txAction(t.id, 'execute')}>Execute</button>
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
