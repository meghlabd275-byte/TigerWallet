// Swap Page — token swap via CoinGecko quote (/swap/quote) + on-chain AMM
// (/amm/quote, /amm/swap). Real price discovery, no fabricated rates.
import React, { useState, useEffect } from 'react';
import { api, WalletRecord } from '../services/api';

const CHAIN_OPTIONS = [
  { id: 1, label: 'Ethereum' },
  { id: 56, label: 'BNB Chain' },
  { id: 137, label: 'Polygon' },
  { id: 42161, label: 'Arbitrum' },
  { id: 10, label: 'Optimism' },
  { id: 8453, label: 'Base' },
];

export default function Swap() {
  const [wallets, setWallets] = useState<WalletRecord[]>([]);
  const [walletId, setWalletId] = useState('');
  const [password, setPassword] = useState('');
  const [fromToken, setFromToken] = useState('ETH');
  const [toToken, setToToken] = useState('USDC');
  const [fromAmount, setFromAmount] = useState('');
  const [chainId, setChainId] = useState(1);
  const [quote, setQuote] = useState<unknown>(null);
  const [ammQuote, setAmmQuote] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [result, setResult] = useState<string | null>(null);

  useEffect(() => {
    api.getWallets().then((data) => {
      setWallets(data.wallets || []);
      if (data.wallets && data.wallets.length > 0) setWalletId(data.wallets[0].id);
    }).catch(() => {});
  }, []);

  const fetchQuote = async () => {
    setError('');
    setQuote(null);
    setAmmQuote(null);
    if (!fromAmount || parseFloat(fromAmount) <= 0) return;
    setBusy(true);
    try {
      const q = await api.getSwapQuote({ fromToken, toToken, fromAmount, chainId });
      setQuote(q);
      try {
        const aq = await api.getAmmQuote({ fromToken, toToken, fromAmount, chainId });
        setAmmQuote(aq);
      } catch {
        // AMM quote may be unavailable for this chain/token pair — honest.
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Quote failed');
    } finally {
      setBusy(false);
    }
  };

  useEffect(() => {
    const t = setTimeout(fetchQuote, 400);
    return () => clearTimeout(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fromToken, toToken, fromAmount, chainId]);

  const handleSwap = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setResult(null);
    setBusy(true);
    try {
      const res = await api.ammSwap({ walletId, password, fromToken, toToken, fromAmount, chainId });
      setResult(typeof res === 'string' ? res : JSON.stringify(res));
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Swap failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="swap-page">
      <h1>Swap</h1>
      {error && <div className="error">{error}</div>}
      {result && (
        <div className="success-banner">
          <h3>✓ Swap transaction submitted to the blockchain network</h3>
          <p className="mono">{result}</p>
        </div>
      )}
      <form onSubmit={handleSwap} className="swap-form">
        <div className="form-group">
          <label>From Wallet</label>
          <select value={walletId} onChange={(e) => setWalletId(e.target.value)} required>
            {wallets.map((w) => <option key={w.id} value={w.id}>{w.label || w.address.slice(0, 10)}</option>)}
          </select>
        </div>
        <div className="form-group">
          <label>Network</label>
          <select value={chainId} onChange={(e) => setChainId(Number(e.target.value))}>
            {CHAIN_OPTIONS.map((c) => <option key={c.id} value={c.id}>{c.label}</option>)}
          </select>
        </div>
        <div className="swap-row">
          <div className="form-group">
            <label>From Token</label>
            <input value={fromToken} onChange={(e) => setFromToken(e.target.value)} required />
          </div>
          <div className="form-group">
            <label>Amount</label>
            <input type="number" step="any" value={fromAmount} onChange={(e) => setFromAmount(e.target.value)} required />
          </div>
        </div>
        <div className="swap-row">
          <div className="form-group">
            <label>To Token</label>
            <input value={toToken} onChange={(e) => setToToken(e.target.value)} required />
          </div>
        </div>
        {Boolean(quote) && <div className="quote-box"><pre>{JSON.stringify(quote, null, 2)}</pre></div>}
        {Boolean(ammQuote) && <div className="quote-box amm"><h4>On-chain AMM quote</h4><pre>{JSON.stringify(ammQuote, null, 2)}</pre></div>}
        <div className="form-group">
          <label>Wallet Password</label>
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required minLength={8} />
        </div>
        <button type="submit" className="primary-btn" disabled={busy}>{busy ? 'Swapping…' : 'Swap'}</button>
      </form>
    </div>
  );
}
