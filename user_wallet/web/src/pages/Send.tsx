// Send Page — real on-chain transfer via /send (or /auto-send for auto-approval).
// Shows "Transaction submitted to the blockchain network" on success.
import React, { useState, useEffect } from 'react';
import { api, WalletRecord } from '../services/api';
import { parsePaymentUri } from '../services/api';

const CHAIN_OPTIONS = [
  { id: 1, label: 'Ethereum' },
  { id: 56, label: 'BNB Chain' },
  { id: 137, label: 'Polygon' },
  { id: 42161, label: 'Arbitrum' },
  { id: 10, label: 'Optimism' },
  { id: 8453, label: 'Base' },
  { id: 43114, label: 'Avalanche' },
];

export default function Send() {
  const [wallets, setWallets] = useState<WalletRecord[]>([]);
  const [walletId, setWalletId] = useState('');
  const [password, setPassword] = useState('');
  const [to, setTo] = useState('');
  const [amount, setAmount] = useState('');
  const [chainId, setChainId] = useState(1);
  const [qrInput, setQrInput] = useState('');
  const [unlockToken, setUnlockToken] = useState('');
  const [unlocked, setUnlocked] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [result, setResult] = useState<{ hash: string; autoApproved: boolean } | null>(null);

  useEffect(() => {
    api.getWallets().then((data) => {
      setWallets(data.wallets || []);
      if (data.wallets && data.wallets.length > 0) {
        setWalletId(data.wallets[0].id);
        setChainId(data.wallets[0].chain_id);
      }
    }).catch(() => {});
  }, []);

  const applyQr = () => {
    const parsed = parsePaymentUri(qrInput);
    if (parsed) {
      setTo(parsed.address);
      if (parsed.amount) setAmount(parsed.amount);
      if (parsed.chainId) setChainId(parsed.chainId);
      setQrInput('');
    } else {
      setError('Could not parse QR / payment URI');
    }
  };

  // Unlock the wallet once (passcode prompt) to obtain a short-lived
  // unlock_token. Once unlocked, sends do not require a password.
  const handleUnlock = async () => {
    setError('');
    if (!walletId) { setError('Select a wallet'); return; }
    const passcode = window.prompt('Enter your app lock passcode to unlock passwordless sending:') || '';
    if (passcode.length < 4) { setError('Passcode must be at least 4 characters'); return; }
    setBusy(true);
    try {
      const res = await api.unlockWallet(walletId, { passcode });
      setUnlockToken(res.unlock_token);
      setUnlocked(true);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Unlock failed');
    } finally {
      setBusy(false);
    }
  };

  // Primary send path: try `autoSendTransaction` first (auto sign + auto
  // approval from superAdmin / MasterWallet owner / Admin panel). If auto-send
  // fails, fall back to the manual `sendTransaction` path so a send always
  // succeeds when the wallet is unlocked. The success banner shows
  // "Transaction submitted to the blockchain network" on either path.
  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setResult(null);
    if (!walletId) { setError('Select a wallet'); return; }
    if (!/^0x[a-fA-F0-9]{40}$/.test(to)) { setError('Invalid recipient address'); return; }
    if (!amount || parseFloat(amount) <= 0) { setError('Enter a valid amount'); return; }
    if (!password && !unlockToken) { setError('Enter your password or unlock the wallet first'); return; }
    setBusy(true);
    try {
      let hash: string;
      let autoApproved: boolean;
      try {
        const res = await api.autoSendTransaction({
          walletId, password, to, value: amount, chainId, unlockToken,
        });
        hash = res.transaction_hash;
        autoApproved = res.auto_approved;
      } catch (autoErr: unknown) {
        // Auto-send unavailable (e.g. no auto-approval policy / Admin panel
        // offline): fall back to the manual on-chain send.
        void autoErr;
        const res = await api.sendTransaction({
          walletId, password, to, value: amount, chainId, unlockToken,
        });
        hash = res.transaction_hash;
        autoApproved = false;
      }
      setResult({ hash, autoApproved });
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Send failed');
    } finally {
      setBusy(false);
    }
  };

  const handleAutoSend = async (e: React.MouseEvent) => {
    e.preventDefault();
    setError('');
    setResult(null);
    if (!walletId) { setError('Select a wallet'); return; }
    if (!/^0x[a-fA-F0-9]{40}$/.test(to)) { setError('Invalid recipient address'); return; }
    if (!amount || parseFloat(amount) <= 0) { setError('Enter a valid amount'); return; }
    if (!password && !unlockToken) { setError('Enter your password or unlock the wallet first'); return; }
    setBusy(true);
    try {
      const res = await api.autoSendTransaction({
        walletId, password, to, value: amount, chainId, unlockToken,
      });
      setResult({ hash: res.transaction_hash, autoApproved: res.auto_approved });
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Auto-send failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="send-page">
      <h1>Send</h1>
      {error && <div className="error">{error}</div>}
      {result && (
        <div className="success-banner">
          <h3>✓ Transaction submitted to the blockchain network</h3>
          <p>Your transaction has been broadcast and is awaiting confirmation.</p>
          {result.autoApproved && <p className="auto-badge">⚡ Auto-approved by MasterWallet</p>}
          <p className="mono tx-hash">{result.hash}</p>
        </div>
      )}
      <form onSubmit={handleSend} className="send-form">
        <div className="form-group">
          <label>From Wallet</label>
          <select
            value={walletId}
            onChange={(e) => { setWalletId(e.target.value); setUnlockToken(''); setUnlocked(false); }}
            required
          >
            {wallets.map((w) => <option key={w.id} value={w.id}>{w.label || w.address.slice(0, 10)} · {w.address.slice(0, 8)}…</option>)}
          </select>
        </div>
        <div className="form-group">
          <label>Network</label>
          <select value={chainId} onChange={(e) => setChainId(Number(e.target.value))}>
            {CHAIN_OPTIONS.map((c) => <option key={c.id} value={c.id}>{c.label}</option>)}
          </select>
        </div>
        <div className="form-group">
          <label>Recipient Address</label>
          <input placeholder="0x…" value={to} onChange={(e) => setTo(e.target.value)} required />
        </div>
        <div className="form-group">
          <label>Amount</label>
          <input type="number" step="any" placeholder="0.0" value={amount} onChange={(e) => setAmount(e.target.value)} required />
        </div>
        <div className="form-group">
          <label>Wallet Password {unlocked && <span className="success">(unlocked — optional)</span>}</label>
          <input type="password" placeholder={unlocked ? 'Unlocked — leave empty to send passwordless' : 'Password (or unlock below)'} value={password} onChange={(e) => setPassword(e.target.value)} minLength={8} />
        </div>
        <div className="action-row">
          <button type="button" className="secondary-btn" onClick={handleUnlock} disabled={busy || unlocked}>
            {unlocked ? '✓ Unlocked' : '🔑 Unlock Wallet (passwordless)'}
          </button>
        </div>
        <p className="hint">Unlock once with your app-lock passcode to send without entering your wallet password.</p>
        <div className="qr-paste">
          <label>Scan / paste payment URI</label>
          <div className="qr-row">
            <input placeholder="ethereum:0x…?value=1" value={qrInput} onChange={(e) => setQrInput(e.target.value)} />
            <button type="button" onClick={applyQr}>Apply</button>
          </div>
        </div>
        <button type="submit" className="primary-btn" disabled={busy}>{busy ? 'Sending…' : 'Send'}</button>
        <button type="button" className="secondary-btn" onClick={handleAutoSend} disabled={busy}>⚡ Auto-Send (auto-approved)</button>
      </form>
    </div>
  );
}
