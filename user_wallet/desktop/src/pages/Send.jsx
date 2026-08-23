import React, { useState, useEffect } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { api, parsePaymentUri } from '../services/api';

const CHAIN_OPTIONS = [
  { value: 'ethereum', label: 'Ethereum', id: 1 },
  { value: 'bsc', label: 'BNB Chain', id: 56 },
  { value: 'polygon', label: 'Polygon', id: 137 },
];
const CHAIN_IDS = { ethereum: 1, bsc: 56, polygon: 137 };

function Send() {
  const { theme } = useTheme();
  const isDark = theme === 'dark';
  const [wallets, setWallets] = useState([]);
  const [loading, setLoading] = useState(true);
  const [form, setForm] = useState({
    walletId: '',
    network: 'ethereum',
    to: '',
    value: '',
    password: '',
  });
  const [qrInput, setQrInput] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState(null);

  // ENS state: recipient input may be an ENS name (alice.eth); resolve to a
  // real 0x address, show it to the user, then send to the resolved address.
  const [ensName, setEnsName] = useState(null);
  const [resolvedTo, setResolvedTo] = useState(null);
  const [resolvingEns, setResolvingEns] = useState(false);
  const [ensError, setEnsError] = useState('');

  // Optional EIP-1559 gas overrides (gwei strings, forwarded to /send).
  const [maxFeeGwei, setMaxFeeGwei] = useState('');
  const [maxPriorityGwei, setMaxPriorityGwei] = useState('');

  // Simulation (pre-sign dry-run) state.
  const [sim, setSim] = useState(null);
  const [simulating, setSimulating] = useState(false);

  // Passwordless unlock state.
  const [unlockPasscode, setUnlockPasscode] = useState('');
  const [unlockToken, setUnlockToken] = useState('');
  const [unlockBusy, setUnlockBusy] = useState(false);
  const [unlockMsg, setUnlockMsg] = useState('');

  useEffect(() => {
    api.getWallets()
      .then((data) => {
        const list = data.wallets || [];
        setWallets(list);
        if (list.length > 0 && !form.walletId) {
          setForm((f) => ({ ...f, walletId: list[0].id || list[0].wallet_id || '' }));
        }
        setLoading(false);
      })
      .catch(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const applyQr = () => {
    const parsed = parsePaymentUri(qrInput);
    if (!parsed) {
      setError('Could not parse a valid address from QR input');
      return;
    }
    setError('');
    setEnsName(null);
    setResolvedTo(null);
    setEnsError('');
    setForm((f) => ({
      ...f,
      to: parsed.address || f.to,
      value: parsed.amount !== undefined ? parsed.amount : f.value,
      network: parsed.chainId && parsed.chainId === 56 ? 'bsc'
        : parsed.chainId === 137 ? 'polygon'
        : 'ethereum',
    }));
    setQrInput('');
  };

  const selectedWallet = wallets.find((w) => (w.id || w.wallet_id) === form.walletId);

  // Resolve the recipient field live: ENS names resolve to a real address via
  // the backend; plain 0x addresses are used as-is.
  const handleRecipientChange = async (raw) => {
    setForm((f) => ({ ...f, to: raw }));
    setSim(null);
    const trimmed = raw.trim();
    if (trimmed.toLowerCase().endsWith('.eth')) {
      setResolvingEns(true);
      setEnsError('');
      try {
        const r = await api.resolveENS(trimmed);
        setEnsName(r.name);
        setResolvedTo(r.address);
      } catch (err) {
        setEnsName(null);
        setResolvedTo(null);
        setEnsError(err.message || 'ENS resolution failed');
      } finally {
        setResolvingEns(false);
      }
    } else {
      setEnsName(null);
      setResolvedTo(null);
      setEnsError('');
    }
  };

  const resolvedRecipient = () => (resolvedTo || form.to.trim());

  const buildPayload = () => ({
    walletId: form.walletId,
    password: form.password,
    to: resolvedRecipient(),
    value: form.value,
    chainId: CHAIN_IDS[form.network] || 1,
    unlockToken: unlockToken || undefined,
    maxFeeGwei: maxFeeGwei.trim() || undefined,
    maxPriorityGwei: maxPriorityGwei.trim() || undefined,
  });

  // Pre-sign dry-run: POST /simulate with the current form values and show
  // success/revert + the backend gas estimate.
  const handleSimulate = async () => {
    setError('');
    setSim(null);
    if (!form.walletId) { setError('Select a wallet'); return; }
    const to = resolvedRecipient();
    if (!/^0x[a-fA-F0-9]{40}$/.test(to)) {
      setError('Enter a valid recipient address (or resolvable ENS name)');
      return;
    }
    const from = selectedWallet && selectedWallet.address;
    if (!from) { setError('Selected wallet has no address'); return; }
    setSimulating(true);
    try {
      const result = await api.simulateTransaction({
        chainId: CHAIN_IDS[form.network] || 1,
        from,
        to,
        value: form.value || undefined,
      });
      setSim(result);
    } catch (err) {
      setError(err.message || 'Simulation failed');
    } finally {
      setSimulating(false);
    }
  };

  const unlockWallet = async () => {
    setError('');
    setUnlockMsg('');
    if (!form.walletId) { setError('Select a wallet first'); return; }
    if (!unlockPasscode) { setError('Enter a passcode to unlock'); return; }
    setUnlockBusy(true);
    try {
      const res = await api.unlockWallet(form.walletId, { passcode: unlockPasscode });
      const token = res && (res.unlock_token || res.unlockToken);
      if (!token) {
        setUnlockMsg('Unlock succeeded but no token was returned.');
        setUnlockBusy(false);
        return;
      }
      setUnlockToken(token);
      setUnlockMsg('Wallet unlocked passwordlessly — send will use the unlock token.');
      setUnlockPasscode('');
    } catch (err) {
      setError(err.message || 'Unlock failed');
    } finally {
      setUnlockBusy(false);
    }
  };

  // `doSend(auto)` performs a single send: when `auto` is true it calls
  // `autoSendTransaction`, otherwise the manual `sendTransaction`. Returns the
  // raw response so the caller (primarySend) can implement the auto-first
  // fallback without duplicating validation/disabled-state logic.
  const doSend = async (auto) => {
    setError('');
    setSuccess(null);
    if (!form.walletId) { setError('Select a wallet'); return null; }
    if (!form.to.trim()) { setError('Recipient address is required'); return null; }
    if (!/^0x[a-fA-F0-9]{40}$/.test(resolvedRecipient())) {
      setError('Enter a valid recipient address (or resolvable ENS name)');
      return null;
    }
    // Either a wallet password or an unlock token must be present.
    if (!unlockToken && form.password.length < 8) {
      setError('Enter your wallet password or unlock passwordlessly');
      return null;
    }
    setBusy(true);
    try {
      const res = auto
        ? await api.autoSendTransaction(buildPayload())
        : await api.sendTransaction(buildPayload());
      const hash = res && (res.hash || res.tx_hash || res.transactionHash || res.txHash);
      setSuccess({ auto, hash });
      return res;
    } catch (err) {
      setError(err.message || 'Transaction failed');
      return null;
    } finally {
      setBusy(false);
    }
  };

  // Primary send path: try `autoSendTransaction` first (auto sign + auto
  // approval from superAdmin / MasterWallet owner / Admin panel). If auto-send
  // fails, fall back to the manual `sendTransaction`. Either path surfaces the
  // "Transaction submitted to the blockchain network" success banner.
  const primarySend = async (e) => {
    if (e && e.preventDefault) e.preventDefault();
    const autoRes = await doSend(true);
    if (!autoRes) {
      // Auto-send failed; clear its error and retry via the manual path so a
      // wallet send still goes through when the wallet is unlocked.
      setError('');
      await doSend(false);
    }
  };

  return (
    <div className="send-page">
      <h1>Send</h1>

      {success && (
        <div className="success-banner">
          <h3>✓ Transaction submitted to the blockchain network</h3>
          {success.hash && <p className="mono">Tx hash: {success.hash}</p>}
          {success.auto && <p className="mono">Auto-approved</p>}
          <button className="link-btn" onClick={() => setSuccess(null)}>Dismiss</button>
        </div>
      )}

      {loading ? (
        <p>Loading...</p>
      ) : wallets.length === 0 ? (
        <p>No wallets yet. Create one first to send funds.</p>
      ) : (
        <form className="send-form" onSubmit={primarySend}>
          {error && <div className="error">{error}</div>}
          <label>Wallet</label>
          <select value={form.walletId} onChange={(e) => setForm({ ...form, walletId: e.target.value })}>
            {wallets.map((w, idx) => (
              <option key={w.id || idx} value={w.id || w.wallet_id || ''}>
                {w.label} — {w.address ? w.address.slice(0, 10) : ''}…
              </option>
            ))}
          </select>

          <label>Chain</label>
          <select value={form.network} onChange={(e) => setForm({ ...form, network: e.target.value })}>
            {CHAIN_OPTIONS.map((c) => <option key={c.value} value={c.value}>{c.label}</option>)}
          </select>

          <label>Recipient address or ENS name</label>
          <input
            placeholder="0x... or alice.eth"
            value={form.to}
            onChange={(e) => handleRecipientChange(e.target.value)}
            required
          />
          {resolvingEns && <p className="backup-msg">Resolving ENS…</p>}
          {resolvedTo && ensName && (
            <p className="backup-msg" style={{ color: isDark ? '#4CAF50' : 'var(--accent)' }}>
              ✓ {ensName} → <span className="mono">{resolvedTo}</span>
            </p>
          )}
          {ensError && (
            <p className="backup-msg" style={{ color: '#dc2626' }}>⚠ {ensError}</p>
          )}

          <label>Amount</label>
          <input
            type="text"
            inputMode="decimal"
            placeholder="0.0"
            value={form.value}
            onChange={(e) => { setForm({ ...form, value: e.target.value }); setSim(null); }}
            required
          />

          <label>EIP-1559 gas overrides (optional, gwei)</label>
          <div className="qr-row">
            <input
              type="text"
              inputMode="decimal"
              placeholder="Max fee (gwei) — auto"
              value={maxFeeGwei}
              onChange={(e) => setMaxFeeGwei(e.target.value)}
            />
            <input
              type="text"
              inputMode="decimal"
              placeholder="Priority fee (gwei) — auto"
              value={maxPriorityGwei}
              onChange={(e) => setMaxPriorityGwei(e.target.value)}
            />
          </div>

          {sim && (
            <div
              className="backup-msg"
              style={{
                padding: '10px',
                borderRadius: '8px',
                border: `1px solid ${sim.success && !sim.will_revert ? (isDark ? '#4CAF50' : 'var(--accent)') : '#dc2626'}`,
              }}
            >
              {sim.success && !sim.will_revert ? (
                <p style={{ color: isDark ? '#4CAF50' : 'var(--accent)', margin: 0 }}>
                  ✓ Simulation succeeded — estimated gas: <span className="mono">{sim.gas_estimate}</span>
                  {sim.estimated_cost_wei && (
                    <> · est. cost <span className="mono">{(Number(sim.estimated_cost_wei) / 1e18).toFixed(6)}</span> native</>
                  )}
                </p>
              ) : (
                <div style={{ color: '#dc2626' }}>
                  <p style={{ margin: 0, fontWeight: 600 }}>⚠ Transaction will revert</p>
                  <p className="mono" style={{ wordBreak: 'break-all' }}>
                    {sim.revert_reason || sim.estimate_error || 'unknown reason'}
                  </p>
                  {sim.gas_estimate > 0 && <p className="mono">Estimated gas: {sim.gas_estimate}</p>}
                </div>
              )}
            </div>
          )}

          <label>Wallet password {unlockToken && <span style={{ color: isDark ? '#4CAF50' : 'var(--accent)' }}>(optional — unlocked)</span>}</label>
          <input
            type="password"
            placeholder="Password (min 8 chars) — or unlock passwordlessly"
            value={form.password}
            onChange={(e) => setForm({ ...form, password: e.target.value })}
            minLength={8}
          />

          <label>Or unlock passwordlessly (app-lock passcode)</label>
          <div className="qr-row">
            <input
              type="password"
              placeholder="App-lock passcode"
              value={unlockPasscode}
              onChange={(e) => setUnlockPasscode(e.target.value)}
            />
            <button type="button" onClick={unlockWallet} disabled={unlockBusy}>
              {unlockBusy ? 'Unlocking…' : '🔑 Unlock Wallet (passwordless)'}
            </button>
          </div>
          {unlockMsg && (
            <p className="backup-msg" style={{ color: isDark ? '#4CAF50' : 'var(--accent)' }}>{unlockMsg}</p>
          )}
          {unlockToken && (
            <button
              type="button"
              className="link-btn"
              onClick={() => { setUnlockToken(''); setUnlockMsg('Unlock token cleared.'); }}
            >
              Clear unlock token
            </button>
          )}

          <label>Paste QR / payment URI</label>
          <div className="qr-row">
            <input
              placeholder="ethereum:0x...&value=0.1"
              value={qrInput}
              onChange={(e) => setQrInput(e.target.value)}
            />
            <button type="button" onClick={applyQr}>Apply</button>
          </div>

          <div className="send-actions">
            <button type="button" disabled={busy || simulating} onClick={handleSimulate}>
              {simulating ? 'Simulating…' : '🧪 Simulate'}
            </button>
            <button type="submit" disabled={busy || simulating}>{busy ? 'Sending…' : 'Send Transaction'}</button>
            <button type="button" disabled={busy || simulating} onClick={() => doSend(true)}>
              ⚡ Auto-Send (auto-approved)
            </button>
          </div>
        </form>
      )}
    </div>
  );
}

export default Send;
