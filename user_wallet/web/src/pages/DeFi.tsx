// DeFi Page — the full DeFi surface: lending, copy-trading, governance/DAO,
// perpetual, margin, prediction markets, launchpool, token sales, dApps.
// All real backend fetches; no fabricated data.
import React, { useState, useEffect } from 'react';
import { api, WalletRecord } from '../services/api';

type Tab = 'lending' | 'copytrading' | 'governance' | 'perpetual' | 'margin' | 'prediction' | 'launchpool' | 'tokensales' | 'dapps';

export default function DeFi() {
  const [tab, setTab] = useState<Tab>('lending');
  const [wallets, setWallets] = useState<WalletRecord[]>([]);
  const [walletId, setWalletId] = useState('');
  const [password, setPassword] = useState('');
  const [asset, setAsset] = useState('ETH');
  const [amount, setAmount] = useState('');
  const [pair, setPair] = useState('ETH/USDC');
  const [side, setSide] = useState('long');
  const [leverage, setLeverage] = useState(2);
  const [data, setData] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [result, setResult] = useState('');

  useEffect(() => {
    api.getWallets().then((d) => { setWallets(d.wallets || []); if (d.wallets && d.wallets[0]) setWalletId(d.wallets[0].id); }).catch(() => {});
  }, []);

  const loadTab = (t: Tab) => {
    setTab(t);
    setError(''); setResult(''); setData(null);
    const fetcher: Record<Tab, () => Promise<unknown>> = {
      lending: () => Promise.all([api.getLendingMarkets(), api.getLendingPositions()]).then(([m, p]) => ({ markets: m, positions: p })),
      copytrading: () => api.getCopyTraders(),
      governance: () => api.getDaoProposals(),
      perpetual: () => api.getPerpetualPositions(),
      margin: () => api.getMarginPositions(),
      prediction: () => api.getPredictionMarkets(),
      launchpool: () => Promise.all([api.getLaunchpool(), api.getLaunchpoolStakes()]).then(([l, s]) => ({ pool: l, stakes: s })),
      tokensales: () => api.getTokenSales(),
      dapps: () => Promise.all([api.getDapps(), api.getDappCategories()]).then(([d, c]) => ({ dapps: d, categories: c })),
    };
    fetcher[t]().then(setData).catch((e: unknown) => setError(e instanceof Error ? e.message : 'Load failed'));
  };

  useEffect(() => { loadTab('lending'); /* eslint-disable-next-line */ }, []);

  const act = async (action: string) => {
    setError(''); setResult(''); setBusy(true);
    try {
      let res: unknown;
      switch (action) {
        case 'supply': res = await api.lendingSupply({ walletId, password, asset, amount, chainId: 1 }); break;
        case 'borrow': res = await api.lendingBorrow({ walletId, password, asset, amount, chainId: 1 }); break;
        case 'withdraw': res = await api.lendingWithdraw({ walletId, password, asset, amount, chainId: 1 }); break;
        case 'repay': res = await api.lendingRepay({ walletId, password, asset, amount, chainId: 1 }); break;
        case 'perpOpen': res = await api.createPerpetualPosition({ pair, side, size: amount, leverage, chainId: 1 }); break;
        case 'marginOpen': res = await api.createMarginPosition({ pair, side, size: amount, leverage, chainId: 1 }); break;
        case 'launchStake': res = await api.launchpoolStake({ walletId, password, amount }); break;
        default: throw new Error('Unknown action');
      }
      setResult(typeof res === 'string' ? res : JSON.stringify(res));
    } catch (err: unknown) { setError(err instanceof Error ? err.message : 'Action failed'); } finally { setBusy(false); }
  };

  const tabs: { id: Tab; label: string }[] = [
    { id: 'lending', label: 'Lending' },
    { id: 'copytrading', label: 'Copy Trading' },
    { id: 'governance', label: 'DAO' },
    { id: 'perpetual', label: 'Perpetual' },
    { id: 'margin', label: 'Margin' },
    { id: 'prediction', label: 'Prediction' },
    { id: 'launchpool', label: 'Launchpool' },
    { id: 'tokensales', label: 'Token Sales' },
    { id: 'dapps', label: 'dApps' },
  ];

  return (
    <div className="defi-page">
      <h1>DeFi</h1>
      <div className="tabs">
        {tabs.map((t) => <button key={t.id} className={tab === t.id ? 'tab active' : 'tab'} onClick={() => loadTab(t.id)}>{t.label}</button>)}
      </div>
      {error && <div className="error">{error}</div>}
      {result && <div className="success-banner"><h3>✓ Transaction submitted to the blockchain network</h3><pre className="mono">{result.slice(0, 500)}</pre></div>}
      <div className="defi-content">
        {(tab === 'lending' || tab === 'perpetual' || tab === 'margin' || tab === 'launchpool') && (
          <div className="action-form">
            <div className="form-group">
              <label>Wallet</label>
              <select value={walletId} onChange={(e) => setWalletId(e.target.value)}>
                {wallets.map((w) => <option key={w.id} value={w.id}>{w.label || w.address.slice(0, 10)}</option>)}
              </select>
            </div>
            <div className="form-group">
              <label>Password</label>
              <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} minLength={8} />
            </div>
            {tab === 'lending' && (<>
              <input value={asset} onChange={(e) => setAsset(e.target.value)} placeholder="Asset" />
              <input type="number" step="any" value={amount} onChange={(e) => setAmount(e.target.value)} placeholder="Amount" />
              <div className="action-row">
                <button onClick={() => act('supply')} disabled={busy}>Supply</button>
                <button onClick={() => act('borrow')} disabled={busy}>Borrow</button>
                <button onClick={() => act('withdraw')} disabled={busy}>Withdraw</button>
                <button onClick={() => act('repay')} disabled={busy}>Repay</button>
              </div>
            </>)}
            {(tab === 'perpetual' || tab === 'margin') && (<>
              <input value={pair} onChange={(e) => setPair(e.target.value)} placeholder="Pair (ETH/USDC)" />
              <select value={side} onChange={(e) => setSide(e.target.value)}><option value="long">Long</option><option value="short">Short</option></select>
              <input type="number" step="any" value={amount} onChange={(e) => setAmount(e.target.value)} placeholder="Size" />
              <input type="number" value={leverage} onChange={(e) => setLeverage(Number(e.target.value))} placeholder="Leverage" />
              <button onClick={() => act(tab === 'perpetual' ? 'perpOpen' : 'marginOpen')} disabled={busy}>Open Position</button>
            </>)}
            {tab === 'launchpool' && (<>
              <input type="number" step="any" value={amount} onChange={(e) => setAmount(e.target.value)} placeholder="Amount" />
              <button onClick={() => act('launchStake')} disabled={busy}>Stake</button>
            </>)}
          </div>
        )}
        {Boolean(data) && <div className="data-box"><pre>{JSON.stringify(data, null, 2).slice(0, 3000)}</pre></div>}
      </div>
    </div>
  );
}
