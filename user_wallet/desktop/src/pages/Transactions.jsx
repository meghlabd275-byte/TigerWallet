// Transactions Page — wired to the canonical wallet-api backend RESTful routes.
// Lists a wallet's transactions (GET /wallets/:id/transactions) and provides
// a real send form (POST /wallets/:id/send). No stubs: every value is a real
// backend fetch.
import React, { useState, useEffect } from 'react';
import { api } from '../services/api';
import TxSubmittedBanner from '../components/TxSubmittedBanner';

function Transactions() {
  const [wallets, setWallets] = useState([]);
  const [walletId, setWalletId] = useState('');
  const [transactions, setTransactions] = useState([]);
  const [loading, setLoading] = useState(false);
  const [filter, setFilter] = useState({ network: '', token: '' });
  const [send, setSend] = useState({ to: '', amount: '', password: '' });
  const [sendResult, setSendResult] = useState(null);
  const [sign, setSign] = useState({ message: '', password: '' });
  const [signResult, setSignResult] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    api.getWallets().then((data) => {
      const ws = data.wallets || [];
      setWallets(ws);
      if (ws.length > 0 && !walletId) setWalletId(ws[0].id);
    }).catch(() => {
      /* empty wallets list stays empty */
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!walletId) {
      setTransactions([]);
      return;
    }
    loadTransactions();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [walletId]);

  const loadTransactions = () => {
    if (!walletId) {
      setTransactions([]);
      return;
    }
    setLoading(true);
    api.getTransactions({ walletId, network: filter.network || undefined, token: filter.token || undefined })
      .then((data) => {
        setTransactions(data.transactions || []);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  };

  const handleSend = async (e) => {
    e.preventDefault();
    setError('');
    setSendResult(null);
    if (!walletId) {
      setError('Select a wallet first');
      return;
    }
    if (send.password.length < 8) {
      setError('Password must be at least 8 characters');
      return;
    }
    try {
      const res = await api.sendTransaction({
        walletId,
        password: send.password,
        to: send.to,
        value: send.amount,
      });
      // Show the "Transaction submitted to the blockchain network" banner with
      // the real tx hash. The chain id comes from the active wallet.
      const activeWallet = wallets.find((w) => w.id === walletId);
      const chainId = activeWallet ? (activeWallet.chain_id || 1) : 1;
      setSendResult({ hash: res.transaction_hash, chainId });
      setSend({ to: '', amount: '', password: '' });
      loadTransactions();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Send failed');
    }
  };

  const selectedWallet = wallets.find((w) => w.id === walletId);

  const handleSign = async (e) => {
    e.preventDefault();
    setError('');
    setSignResult('');
    if (!walletId) {
      setError('Select a wallet first');
      return;
    }
    if (sign.password.length < 8) {
      setError('Password must be at least 8 characters');
      return;
    }
    try {
      const res = await api.signMessage({
        walletId,
        password: sign.password,
        message: sign.message,
      });
      setSignResult(res.signature);
      setSign({ message: '', password: '' });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Sign failed');
    }
  };

  return (
    <div className="transactions-page">
      <header className="page-header">
        <h1>Transactions</h1>
        <div className="filters">
          <select value={walletId} onChange={(e) => setWalletId(e.target.value)}>
            <option value="">Select wallet…</option>
            {wallets.map((w) => (
              <option key={w.id} value={w.id}>
                {w.label || (w.address || '').slice(0, 8)} (#{w.chain_id})
              </option>
            ))}
          </select>
          <select value={filter.network} onChange={(e) => setFilter({ ...filter, network: e.target.value })}>
            <option value="">All Networks</option>
            <option value="ethereum">Ethereum</option>
            <option value="bsc">BNB Chain</option>
            <option value="polygon">Polygon</option>
          </select>
          <select value={filter.token} onChange={(e) => setFilter({ ...filter, token: e.target.value })}>
            <option value="">All Tokens</option>
            <option value="ETH">ETH</option>
            <option value="USDT">USDT</option>
            <option value="USDC">USDC</option>
          </select>
          <button onClick={loadTransactions}>Apply</button>
        </div>
      </header>

      <div className="create-form">
        <h3>Send {selectedWallet ? `from ${selectedWallet.label || (selectedWallet.address || '').slice(0, 8)}` : ''}</h3>
        {error && <div className="error">{error}</div>}
        {sendResult && (
          <TxSubmittedBanner
            txHash={sendResult.hash}
            chainId={sendResult.chainId}
            onDismiss={() => setSendResult(null)}
          />
        )}
        <form onSubmit={handleSend}>
          <input
            placeholder="Recipient address (0x…)"
            value={send.to}
            onChange={(e) => setSend({ ...send, to: e.target.value })}
            required
          />
          <input
            placeholder="Amount (native units, e.g. 0.01)"
            value={send.amount}
            onChange={(e) => setSend({ ...send, amount: e.target.value })}
            required
          />
          <input
            type="password"
            placeholder="Wallet password (decrypts seed)"
            value={send.password}
            onChange={(e) => setSend({ ...send, password: e.target.value })}
            required
            minLength={8}
          />
          <button type="submit">Send</button>
        </form>
      </div>

      <div className="create-form">
        <h3>Sign Message {selectedWallet ? `with ${selectedWallet.label || (selectedWallet.address || '').slice(0, 8)}` : ''}</h3>
        {error && <div className="error">{error}</div>}
        {signResult && (
          <div className="status confirmed" style={{ display: 'block', marginBottom: 12, wordBreak: 'break-all' }}>
            Signature: {signResult}
          </div>
        )}
        <form onSubmit={handleSign}>
          <input
            placeholder="Message to sign (EIP-191 personal_sign)"
            value={sign.message}
            onChange={(e) => setSign({ ...sign, message: e.target.value })}
            required
          />
          <input
            type="password"
            placeholder="Wallet password (decrypts seed)"
            value={sign.password}
            onChange={(e) => setSign({ ...sign, password: e.target.value })}
            required
            minLength={8}
          />
          <button type="submit">Sign</button>
        </form>
      </div>

      {loading ? <p>Loading...</p> : transactions.length === 0 ? (
        <p>{walletId ? 'No transactions found for this wallet.' : 'Select a wallet to view transactions.'}</p>
      ) : (
        <table className="transactions-table">
          <thead>
            <tr>
              <th>Tx Hash</th>
              <th>From</th>
              <th>To</th>
              <th>Amount</th>
              <th>Status</th>
              <th>Date</th>
            </tr>
          </thead>
          <tbody>
            {transactions.map((tx) => (
              <tr key={tx.id || tx.tx_hash}>
                <td className="mono">{tx.tx_hash ? `${tx.tx_hash.slice(0, 14)}…` : '—'}</td>
                <td className="mono">{tx.from ? `${tx.from.slice(0, 10)}…` : '—'}</td>
                <td className="mono">{tx.to ? `${tx.to.slice(0, 10)}…` : '—'}</td>
                <td>{tx.amount}{tx.token ? ` ${tx.token}` : ''}</td>
                <td>
                  <span className={`status ${tx.status === 'broadcast' || tx.status === 'confirmed' ? 'confirmed' : 'failed'}`}>
                    {tx.status || 'unknown'}
                  </span>
                </td>
                <td>{tx.created_at ? new Date(tx.created_at).toLocaleString() : '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

export default Transactions;
