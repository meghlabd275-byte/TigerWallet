// Non-EVM Chains Page — derive native addresses (bitcoin/solana/cosmos) from
// the stored seed and sign messages. Real key derivation + signing on the
// backend (mainnet only); fail-closed errors, no fabricated addresses.
import React from 'react';
import { api } from '../services/api';
export default function NonEvm() {
  const [wallets, setWallets] = React.useState<any[]>([]);
  const [chain, setChain] = React.useState('solana');
  const [password, setPassword] = React.useState('');
  const [message, setMessage] = React.useState('');
  const [derivedAddress, setDerivedAddress] = React.useState('');
  const [signature, setSignature] = React.useState('');
  const [error, setError] = React.useState('');
  React.useEffect(() => {
    api.getWallets().then((d) => setWallets(d.wallets || d || [])).catch(() => {});
  }, []);
  const derive = async () => {
    setError(''); setDerivedAddress('');
    const wallet = wallets[0];
    if (!wallet) { setError('No wallet available'); return; }
    if (!password) { setError('Enter the wallet password'); return; }
    try {
      const r = await api.nonEvmAddress({ walletId: wallet.id, password, chainType: chain });
      setDerivedAddress(r.address || '');
      if (!r.address) setError('No address returned');
    } catch (e: any) {
      setError(`Derive failed: ${e.message}`);
    }
  };
  const sign = async () => {
    setError(''); setSignature('');
    const wallet = wallets[0];
    if (!wallet) { setError('No wallet available'); return; }
    if (!password || !message) { setError('Enter password and message'); return; }
    try {
      const r = await api.nonEvmSign({ walletId: wallet.id, password, message, chainType: chain });
      setSignature(r.signature || '');
      if (!r.signature) setError('No signature returned');
    } catch (e: any) {
      setError(`Sign failed: ${e.message}`);
    }
  };
  return (
    <div>
      <header className="page-header"><h1>Non-EVM Chains</h1></header>
      {error && <div className="error">{error}</div>}
      <div style={{ marginBottom: 24 }}>
        <h2>Derive native address</h2>
        <div className="form-group">
          <label>Chain</label>
          <select value={chain} onChange={(e) => setChain(e.target.value)}>
            <option value="bitcoin">Bitcoin</option>
            <option value="solana">Solana</option>
            <option value="cosmos">Cosmos</option>
          </select>
        </div>
        <div className="form-group">
          <label>Wallet password</label>
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
        </div>
        <button className="primary-btn" onClick={derive}>Derive Address</button>
        {derivedAddress && (
          <div className="quote-box">
            <code>{derivedAddress}</code>
          </div>
        )}
      </div>
      <div style={{ marginBottom: 24 }}>
        <h2>Sign message</h2>
        <div className="form-group">
          <label>Message</label>
          <input value={message} onChange={(e) => setMessage(e.target.value)} />
        </div>
        <button className="primary-btn" onClick={sign}>Sign Message</button>
        {signature && (
          <div className="quote-box">
            <code>{signature}</code>
          </div>
        )}
      </div>
    </div>
  );
}
