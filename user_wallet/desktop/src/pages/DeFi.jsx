import React, { useState, useEffect } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { api } from '../services/api';

const CHAIN_OPTIONS = [
  { value: 'ethereum', label: 'Ethereum', id: 1 },
  { value: 'bsc', label: 'BNB Chain', id: 56 },
  { value: 'polygon', label: 'Polygon', id: 137 },
];
const CHAIN_IDS = { ethereum: 1, bsc: 56, polygon: 137 };

const TABS = [
  { id: 'lending', label: 'Lending' },
  { id: 'copy', label: 'Copy Trading' },
  { id: 'dao', label: 'DAO / Governance' },
  { id: 'perpetual', label: 'Perpetual' },
  { id: 'margin', label: 'Margin' },
  { id: 'prediction', label: 'Prediction' },
  { id: 'launchpool', label: 'Launchpool' },
  { id: 'tokensales', label: 'Token Sales' },
];

function Section({ title, children }) {
  return (
    <section style={{ marginBottom: '24px' }}>
      <h3 style={{ marginBottom: '12px' }}>{title}</h3>
      {children}
    </section>
  );
}

function Msg({ error, info, isDark }) {
  return (
    <>
      {error && <div className="error">{error}</div>}
      {info && <div className="success-banner" style={{ marginBottom: '12px' }}><h3 style={{ color: isDark ? '#4CAF50' : 'var(--accent)' }}>✓ {info}</h3></div>}
    </>
  );
}

// ---------- Lending ----------
function LendingSection({ wallets, isDark }) {
  const [markets, setMarkets] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [info, setInfo] = useState('');
  const [action, setAction] = useState('supply');
  const [busy, setBusy] = useState(false);
  const [form, setForm] = useState({ walletId: '', network: 'ethereum', asset: '', amount: '', password: '' });

  useEffect(() => {
    let alive = true;
    setLoading(true);
    api.getLendingMarkets()
      .then((d) => { if (alive) { setMarkets(d.markets || []); setLoading(false); } })
      .catch((e) => { if (alive) { setError(e.message || 'Failed to load lending markets'); setLoading(false); } });
    return () => { alive = false; };
  }, []);

  useEffect(() => {
    if (wallets.length > 0 && !form.walletId) setForm((f) => ({ ...f, walletId: wallets[0].id || wallets[0].wallet_id || '' }));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [wallets]);

  const submit = async (e) => {
    e.preventDefault();
    setError(''); setInfo('');
    if (!form.walletId) { setError('Select a wallet'); return; }
    if (!form.asset) { setError('Asset is required'); return; }
    if (!form.amount) { setError('Amount is required'); return; }
    if (form.password.length < 8) { setError('Password is required (min 8 chars)'); return; }
    setBusy(true);
    try {
      const params = { walletId: form.walletId, password: form.password, asset: form.asset, amount: form.amount, chainId: CHAIN_IDS[form.network] || 1 };
      if (action === 'supply') await api.lendingSupply(params);
      else if (action === 'borrow') await api.lendingBorrow(params);
      else if (action === 'withdraw') await api.lendingWithdraw(params);
      else await api.lendingRepay(params);
      setInfo(`${action} submitted.`);
    } catch (err) {
      setError(err.message || `${action} failed`);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Section title="Lending markets">
      <Msg error={error} info={info} isDark={isDark} />
      {loading ? <p>Loading...</p> : markets.length === 0 ? <p>No lending markets available.</p> : (
        <div className="wallets-grid">
          {markets.map((m, idx) => (
            <div key={idx} className="wallet-card">
              <h3>{m.asset || m.symbol || 'Asset'}</h3>
              <p className="network">Supply APY: {m.supply_apy ?? '—'}</p>
              <p className="network">Borrow APY: {m.borrow_apy ?? '—'}</p>
            </div>
          ))}
        </div>
      )}
      {wallets.length === 0 ? <p style={{ marginTop: '12px' }}>No wallets available to act.</p> : (
        <form className="send-form" style={{ marginTop: '16px' }} onSubmit={submit}>
          <label>Action</label>
          <select value={action} onChange={(e) => setAction(e.target.value)}>
            <option value="supply">Supply</option>
            <option value="borrow">Borrow</option>
            <option value="withdraw">Withdraw</option>
            <option value="repay">Repay</option>
          </select>
          <label>Wallet</label>
          <select value={form.walletId} onChange={(e) => setForm({ ...form, walletId: e.target.value })}>
            {wallets.map((w, idx) => <option key={w.id || idx} value={w.id || w.wallet_id || ''}>{w.label}</option>)}
          </select>
          <label>Chain</label>
          <select value={form.network} onChange={(e) => setForm({ ...form, network: e.target.value })}>
            {CHAIN_OPTIONS.map((c) => <option key={c.value} value={c.value}>{c.label}</option>)}
          </select>
          <label>Asset</label>
          <input placeholder="e.g. USDC" value={form.asset} onChange={(e) => setForm({ ...form, asset: e.target.value })} required />
          <label>Amount</label>
          <input type="text" inputMode="decimal" placeholder="0.0" value={form.amount} onChange={(e) => setForm({ ...form, amount: e.target.value })} required />
          <label>Wallet password</label>
          <input type="password" placeholder="Password (min 8 chars)" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} minLength={8} />
          <div className="send-actions">
            <button type="submit" disabled={busy}>{busy ? 'Submitting…' : 'Submit'}</button>
          </div>
        </form>
      )}
    </Section>
  );
}

// ---------- Copy trading ----------
function CopySection({ isDark }) {
  const [traders, setTraders] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [info, setInfo] = useState('');
  const [followId, setFollowId] = useState('');
  const [allocation, setAllocation] = useState('');
  const [busy, setBusy] = useState(false);
  const [stopId, setStopId] = useState(null);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    api.getCopyTraders()
      .then((d) => { if (alive) { setTraders(d.traders || []); setLoading(false); } })
      .catch((e) => { if (alive) { setError(e.message || 'Failed to load copy traders'); setLoading(false); } });
    return () => { alive = false; };
  }, []);

  const follow = async (e) => {
    e.preventDefault();
    setError(''); setInfo('');
    if (!followId) { setError('Enter a trader id'); return; }
    setBusy(true);
    try {
      await api.followTrader({ traderId: followId, allocation: allocation ? Number(allocation) : undefined });
      setInfo('Following trader.');
      setFollowId(''); setAllocation('');
      const d = await api.getCopyTraders();
      setTraders(d.traders || []);
    } catch (err) {
      setError(err.message || 'Follow failed');
    } finally {
      setBusy(false);
    }
  };

  const stop = async (c) => {
    setError(''); setInfo('');
    const id = c.copier_id || c.copierId || c.id;
    setStopId(id);
    try {
      await api.stopCopyTrader(id);
      setInfo('Stopped copying trader.');
      const d = await api.getCopyTraders();
      setTraders(d.traders || []);
    } catch (err) {
      setError(err.message || 'Stop failed');
    } finally {
      setStopId(null);
    }
  };

  return (
    <Section title="Copy trading">
      <Msg error={error} info={info} isDark={isDark} />
      {loading ? <p>Loading...</p> : traders.length === 0 ? <p>No copy traders listed.</p> : (
        <div className="wallets-grid">
          {traders.map((t, idx) => (
            <div key={t.id || t.trader_id || idx} className="wallet-card">
              <h3>{t.username || t.name || 'Trader'}</h3>
              <p className="network">ROI: {t.roi ?? '—'}</p>
              <p className="network">Followers: {t.followers ?? '—'}</p>
              <div className="mnemonic-actions" style={{ marginTop: '12px' }}>
                <button onClick={() => stop(t)} disabled={stopId === (t.copier_id || t.copierId || t.id)}>Stop</button>
              </div>
            </div>
          ))}
        </div>
      )}
      <form className="send-form" style={{ marginTop: '16px' }} onSubmit={follow}>
        <label>Trader id</label>
        <input placeholder="trader id" value={followId} onChange={(e) => setFollowId(e.target.value)} required />
        <label>Allocation (optional)</label>
        <input type="text" inputMode="decimal" placeholder="amount" value={allocation} onChange={(e) => setAllocation(e.target.value)} />
        <div className="send-actions">
          <button type="submit" disabled={busy}>{busy ? 'Following…' : 'Follow'}</button>
        </div>
      </form>
    </Section>
  );
}

// ---------- DAO ----------
function DaoSection({ isDark }) {
  const [proposals, setProposals] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [info, setInfo] = useState('');
  const [newProposal, setNewProposal] = useState({ title: '', description: '' });
  const [busy, setBusy] = useState(false);
  const [voteId, setVoteId] = useState(null);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    api.getDaoProposals()
      .then((d) => { if (alive) { setProposals(d.proposals || []); setLoading(false); } })
      .catch((e) => { if (alive) { setError(e.message || 'Failed to load proposals'); setLoading(false); } });
    return () => { alive = false; };
  }, []);

  const create = async (e) => {
    e.preventDefault();
    setError(''); setInfo('');
    if (!newProposal.title.trim() || !newProposal.description.trim()) { setError('Title and description are required'); return; }
    setBusy(true);
    try {
      await api.createDaoProposal({ title: newProposal.title.trim(), description: newProposal.description.trim() });
      setInfo('Proposal created.');
      setNewProposal({ title: '', description: '' });
      const d = await api.getDaoProposals();
      setProposals(d.proposals || []);
    } catch (err) {
      setError(err.message || 'Create failed');
    } finally {
      setBusy(false);
    }
  };

  const vote = async (p, support) => {
    setError(''); setInfo('');
    const id = p.id || p.proposal_id || p.proposalId;
    setVoteId(id);
    try {
      await api.voteDaoProposal({ proposalId: id, support });
      setInfo('Vote submitted.');
      const d = await api.getDaoProposals();
      setProposals(d.proposals || []);
    } catch (err) {
      setError(err.message || 'Vote failed');
    } finally {
      setVoteId(null);
    }
  };

  return (
    <Section title="DAO / Governance">
      <Msg error={error} info={info} isDark={isDark} />
      {loading ? <p>Loading...</p> : proposals.length === 0 ? <p>No proposals yet.</p> : (
        <div className="wallets-grid">
          {proposals.map((p, idx) => {
            const id = p.id || p.proposal_id || p.proposalId || idx;
            return (
              <div key={id} className="wallet-card">
                <h3>{p.title || 'Proposal'}</h3>
                <p className="address">{p.description || ''}</p>
                <p className="network">For: {p.for_votes ?? '0'} · Against: {p.against_votes ?? '0'}</p>
                <div className="mnemonic-actions" style={{ marginTop: '12px' }}>
                  <button onClick={() => vote(p, true)} disabled={voteId === id}>👍 For</button>
                  <button onClick={() => vote(p, false)} disabled={voteId === id}>👎 Against</button>
                </div>
              </div>
            );
          })}
        </div>
      )}
      <form className="send-form" style={{ marginTop: '16px' }} onSubmit={create}>
        <label>Title</label>
        <input placeholder="Proposal title" value={newProposal.title} onChange={(e) => setNewProposal({ ...newProposal, title: e.target.value })} required />
        <label>Description</label>
        <textarea placeholder="Proposal description" rows={3} value={newProposal.description} onChange={(e) => setNewProposal({ ...newProposal, description: e.target.value })} required />
        <div className="send-actions">
          <button type="submit" disabled={busy}>{busy ? 'Creating…' : 'Create Proposal'}</button>
        </div>
      </form>
    </Section>
  );
}

// ---------- Perpetual ----------
function PerpetualSection({ isDark }) {
  const [positions, setPositions] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [info, setInfo] = useState('');
  const [form, setForm] = useState({ pair: 'ETH/USD', side: 'long', size: '', leverage: '1', network: 'ethereum' });
  const [busy, setBusy] = useState(false);
  const [closeId, setCloseId] = useState(null);

  const load = () => {
    setLoading(true);
    api.getPerpetualPositions()
      .then((d) => { setPositions(d.positions || []); setLoading(false); })
      .catch((e) => { setError(e.message || 'Failed to load positions'); setLoading(false); });
  };

  useEffect(() => { load(); }, []);

  const create = async (e) => {
    e.preventDefault();
    setError(''); setInfo('');
    if (!form.pair || !form.size) { setError('Pair and size are required'); return; }
    setBusy(true);
    try {
      await api.createPerpetualPosition({ pair: form.pair, side: form.side, size: form.size, leverage: Number(form.leverage) || 1, chainId: CHAIN_IDS[form.network] || 1 });
      setInfo('Perpetual position created.');
      setForm((f) => ({ ...f, size: '' }));
      load();
    } catch (err) {
      setError(err.message || 'Create failed');
    } finally {
      setBusy(false);
    }
  };

  const close = async (p) => {
    setError(''); setInfo('');
    const id = p.id || p.position_id || p.positionId;
    setCloseId(id);
    try {
      await api.closePerpetualPosition(id);
      setInfo('Position closed.');
      load();
    } catch (err) {
      setError(err.message || 'Close failed');
    } finally {
      setCloseId(null);
    }
  };

  return (
    <Section title="Perpetual positions">
      <Msg error={error} info={info} isDark={isDark} />
      {loading ? <p>Loading...</p> : positions.length === 0 ? <p>No open perpetual positions.</p> : (
        <div className="wallets-grid">
          {positions.map((p, idx) => {
            const id = p.id || p.position_id || p.positionId || idx;
            return (
              <div key={id} className="wallet-card">
                <h3>{p.pair || 'Pair'}</h3>
                <p className="network">Side: {p.side} · Size: {p.size}</p>
                <p className="network">Leverage: {p.leverage}x · PnL: {p.pnl ?? '—'}</p>
                <div className="mnemonic-actions" style={{ marginTop: '12px' }}>
                  <button onClick={() => close(p)} disabled={closeId === id}>Close</button>
                </div>
              </div>
            );
          })}
        </div>
      )}
      <form className="send-form" style={{ marginTop: '16px' }} onSubmit={create}>
        <label>Pair</label>
        <input placeholder="ETH/USD" value={form.pair} onChange={(e) => setForm({ ...form, pair: e.target.value })} required />
        <label>Side</label>
        <select value={form.side} onChange={(e) => setForm({ ...form, side: e.target.value })}>
          <option value="long">Long</option>
          <option value="short">Short</option>
        </select>
        <label>Size</label>
        <input type="text" inputMode="decimal" placeholder="0.0" value={form.size} onChange={(e) => setForm({ ...form, size: e.target.value })} required />
        <label>Leverage</label>
        <input type="number" min="1" placeholder="1" value={form.leverage} onChange={(e) => setForm({ ...form, leverage: e.target.value })} />
        <label>Chain</label>
        <select value={form.network} onChange={(e) => setForm({ ...form, network: e.target.value })}>
          {CHAIN_OPTIONS.map((c) => <option key={c.value} value={c.value}>{c.label}</option>)}
        </select>
        <div className="send-actions">
          <button type="submit" disabled={busy}>{busy ? 'Opening…' : 'Open Position'}</button>
        </div>
      </form>
    </Section>
  );
}

// ---------- Margin ----------
function MarginSection({ isDark }) {
  const [positions, setPositions] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [info, setInfo] = useState('');
  const [form, setForm] = useState({ pair: 'ETH/USD', side: 'long', size: '', leverage: '1', network: 'ethereum' });
  const [busy, setBusy] = useState(false);
  const [closeId, setCloseId] = useState(null);

  const load = () => {
    setLoading(true);
    api.getMarginPositions()
      .then((d) => { setPositions(d.positions || []); setLoading(false); })
      .catch((e) => { setError(e.message || 'Failed to load positions'); setLoading(false); });
  };

  useEffect(() => { load(); }, []);

  const create = async (e) => {
    e.preventDefault();
    setError(''); setInfo('');
    if (!form.pair || !form.size) { setError('Pair and size are required'); return; }
    setBusy(true);
    try {
      await api.createMarginPosition({ pair: form.pair, side: form.side, size: form.size, leverage: Number(form.leverage) || 1, chainId: CHAIN_IDS[form.network] || 1 });
      setInfo('Margin position created.');
      setForm((f) => ({ ...f, size: '' }));
      load();
    } catch (err) {
      setError(err.message || 'Create failed');
    } finally {
      setBusy(false);
    }
  };

  const close = async (p) => {
    setError(''); setInfo('');
    const id = p.id || p.position_id || p.positionId;
    setCloseId(id);
    try {
      await api.closeMarginPosition(id);
      setInfo('Position closed.');
      load();
    } catch (err) {
      setError(err.message || 'Close failed');
    } finally {
      setCloseId(null);
    }
  };

  return (
    <Section title="Margin positions">
      <Msg error={error} info={info} isDark={isDark} />
      {loading ? <p>Loading...</p> : positions.length === 0 ? <p>No open margin positions.</p> : (
        <div className="wallets-grid">
          {positions.map((p, idx) => {
            const id = p.id || p.position_id || p.positionId || idx;
            return (
              <div key={id} className="wallet-card">
                <h3>{p.pair || 'Pair'}</h3>
                <p className="network">Side: {p.side} · Size: {p.size}</p>
                <p className="network">Leverage: {p.leverage}x</p>
                <div className="mnemonic-actions" style={{ marginTop: '12px' }}>
                  <button onClick={() => close(p)} disabled={closeId === id}>Close</button>
                </div>
              </div>
            );
          })}
        </div>
      )}
      <form className="send-form" style={{ marginTop: '16px' }} onSubmit={create}>
        <label>Pair</label>
        <input placeholder="ETH/USD" value={form.pair} onChange={(e) => setForm({ ...form, pair: e.target.value })} required />
        <label>Side</label>
        <select value={form.side} onChange={(e) => setForm({ ...form, side: e.target.value })}>
          <option value="long">Long</option>
          <option value="short">Short</option>
        </select>
        <label>Size</label>
        <input type="text" inputMode="decimal" placeholder="0.0" value={form.size} onChange={(e) => setForm({ ...form, size: e.target.value })} required />
        <label>Leverage</label>
        <input type="number" min="1" placeholder="1" value={form.leverage} onChange={(e) => setForm({ ...form, leverage: e.target.value })} />
        <label>Chain</label>
        <select value={form.network} onChange={(e) => setForm({ ...form, network: e.target.value })}>
          {CHAIN_OPTIONS.map((c) => <option key={c.value} value={c.value}>{c.label}</option>)}
        </select>
        <div className="send-actions">
          <button type="submit" disabled={busy}>{busy ? 'Opening…' : 'Open Position'}</button>
        </div>
      </form>
    </Section>
  );
}

// ---------- Prediction ----------
function PredictionSection({ isDark }) {
  const [markets, setMarkets] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [info, setInfo] = useState('');
  const [betId, setBetId] = useState(null);
  const [amounts, setAmounts] = useState({});

  useEffect(() => {
    let alive = true;
    setLoading(true);
    api.getPredictionMarkets()
      .then((d) => { if (alive) { setMarkets(d.markets || []); setLoading(false); } })
      .catch((e) => { if (alive) { setError(e.message || 'Failed to load markets'); setLoading(false); } });
    return () => { alive = false; };
  }, []);

  const bet = async (m, side) => {
    setError(''); setInfo('');
    const id = m.id || m.market_id || m.marketId;
    const amount = amounts[id] || '';
    if (!amount) { setError('Enter an amount'); return; }
    setBetId(id + side);
    try {
      await api.placePredictionBet({ marketId: id, side, amount });
      setInfo(`Bet placed (${side}).`);
    } catch (err) {
      setError(err.message || 'Bet failed');
    } finally {
      setBetId(null);
    }
  };

  return (
    <Section title="Prediction markets">
      <Msg error={error} info={info} isDark={isDark} />
      {loading ? <p>Loading...</p> : markets.length === 0 ? <p>No prediction markets available.</p> : (
        <div className="wallets-grid">
          {markets.map((m, idx) => {
            const id = m.id || m.market_id || m.marketId || idx;
            return (
              <div key={id} className="wallet-card">
                <h3>{m.question || m.title || 'Market'}</h3>
                <p className="network">Ends: {m.end_date || m.ends_at || '—'}</p>
                <input
                  type="text"
                  inputMode="decimal"
                  placeholder="Amount"
                  value={amounts[id] || ''}
                  onChange={(e) => setAmounts((a) => ({ ...a, [id]: e.target.value }))}
                />
                <div className="mnemonic-actions" style={{ marginTop: '12px' }}>
                  <button onClick={() => bet(m, true)} disabled={betId === id + true}>👍 Yes</button>
                  <button onClick={() => bet(m, false)} disabled={betId === id + false}>👎 No</button>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </Section>
  );
}

// ---------- Launchpool ----------
function LaunchpoolSection({ wallets, isDark }) {
  const [pool, setPool] = useState(null);
  const [stakes, setStakes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [info, setInfo] = useState('');
  const [form, setForm] = useState({ walletId: '', network: 'ethereum', amount: '', password: '', action: 'stake' });
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    Promise.all([api.getLaunchpool(), api.getLaunchpoolStakes()])
      .then(([p, s]) => { if (!alive) return; setPool(p); setStakes(s.stakes || []); setLoading(false); })
      .catch((e) => { if (alive) { setError(e.message || 'Failed to load launchpool'); setLoading(false); } });
    return () => { alive = false; };
  }, []);

  useEffect(() => {
    if (wallets.length > 0 && !form.walletId) setForm((f) => ({ ...f, walletId: wallets[0].id || wallets[0].wallet_id || '' }));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [wallets]);

  const submit = async (e) => {
    e.preventDefault();
    setError(''); setInfo('');
    if (!form.walletId) { setError('Select a wallet'); return; }
    if (!form.amount) { setError('Amount is required'); return; }
    if (form.password.length < 8) { setError('Password is required (min 8 chars)'); return; }
    setBusy(true);
    try {
      const params = { walletId: form.walletId, password: form.password, amount: form.amount };
      if (form.action === 'stake') await api.launchpoolStake(params);
      else await api.launchpoolUnstake(params);
      setInfo(`${form.action} submitted.`);
      setForm((f) => ({ ...f, amount: '' }));
      const s = await api.getLaunchpoolStakes();
      setStakes(s.stakes || []);
    } catch (err) {
      setError(err.message || `${form.action} failed`);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Section title="Launchpool">
      <Msg error={error} info={info} isDark={isDark} />
      {loading ? <p>Loading...</p> : (
        <>
          {pool && (
            <div className="wallet-card" style={{ maxWidth: '500px', marginBottom: '16px' }}>
              <h3>{pool.name || 'Launchpool'}</h3>
              <p className="network">APY: {pool.apy ?? '—'}</p>
              <p className="network">Ends: {pool.end_date || pool.ends_at || '—'}</p>
            </div>
          )}
          {stakes.length === 0 ? <p>No launchpool stakes yet.</p> : (
            <div className="wallets-grid">
              {stakes.map((s, idx) => (
                <div key={idx} className="wallet-card">
                  <h3>Stake</h3>
                  <p className="network">Amount: {s.amount}</p>
                  <p className="network">Rewards: {s.rewards ?? '—'}</p>
                </div>
              ))}
            </div>
          )}
          {wallets.length === 0 ? <p style={{ marginTop: '12px' }}>No wallets available to stake.</p> : (
            <form className="send-form" style={{ marginTop: '16px' }} onSubmit={submit}>
              <label>Action</label>
              <select value={form.action} onChange={(e) => setForm({ ...form, action: e.target.value })}>
                <option value="stake">Stake</option>
                <option value="unstake">Unstake</option>
              </select>
              <label>Wallet</label>
              <select value={form.walletId} onChange={(e) => setForm({ ...form, walletId: e.target.value })}>
                {wallets.map((w, idx) => <option key={w.id || idx} value={w.id || w.wallet_id || ''}>{w.label}</option>)}
              </select>
              <label>Amount</label>
              <input type="text" inputMode="decimal" placeholder="0.0" value={form.amount} onChange={(e) => setForm({ ...form, amount: e.target.value })} required />
              <label>Wallet password</label>
              <input type="password" placeholder="Password (min 8 chars)" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} minLength={8} />
              <div className="send-actions">
                <button type="submit" disabled={busy}>{busy ? 'Submitting…' : 'Submit'}</button>
              </div>
            </form>
          )}
        </>
      )}
    </Section>
  );
}

// ---------- Token sales ----------
function TokenSalesSection({ isDark }) {
  const [sales, setSales] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [info, setInfo] = useState('');
  const [amounts, setAmounts] = useState({});
  const [partId, setPartId] = useState(null);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    api.getTokenSales()
      .then((d) => { if (alive) { setSales(d.sales || []); setLoading(false); } })
      .catch((e) => { if (alive) { setError(e.message || 'Failed to load token sales'); setLoading(false); } });
    return () => { alive = false; };
  }, []);

  const participate = async (s) => {
    setError(''); setInfo('');
    const id = s.id || s.sale_id || s.saleId;
    const amount = amounts[id] || '';
    if (!amount) { setError('Enter an amount'); return; }
    setPartId(id);
    try {
      await api.participateTokenSale({ saleId: id, amount });
      setInfo('Participation submitted.');
    } catch (err) {
      setError(err.message || 'Participation failed');
    } finally {
      setPartId(null);
    }
  };

  return (
    <Section title="Token sales">
      <Msg error={error} info={info} isDark={isDark} />
      {loading ? <p>Loading...</p> : sales.length === 0 ? <p>No token sales available.</p> : (
        <div className="wallets-grid">
          {sales.map((s, idx) => {
            const id = s.id || s.sale_id || s.saleId || idx;
            return (
              <div key={id} className="wallet-card">
                <h3>{s.name || s.token || 'Token sale'}</h3>
                <p className="network">Price: {s.price ?? '—'}</p>
                <p className="network">Ends: {s.end_date || s.ends_at || '—'}</p>
                <input
                  type="text"
                  inputMode="decimal"
                  placeholder="Amount"
                  value={amounts[id] || ''}
                  onChange={(e) => setAmounts((a) => ({ ...a, [id]: e.target.value }))}
                />
                <div className="mnemonic-actions" style={{ marginTop: '12px' }}>
                  <button onClick={() => participate(s)} disabled={partId === id}>
                    {partId === id ? 'Submitting…' : 'Participate'}
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </Section>
  );
}

function DeFi() {
  const { theme } = useTheme();
  const isDark = theme === 'dark';
  const [tab, setTab] = useState('lending');
  const [wallets, setWallets] = useState([]);
  const [walletsLoading, setWalletsLoading] = useState(true);

  useEffect(() => {
    let alive = true;
    api.getWallets()
      .then((d) => { if (!alive) return; setWallets(d.wallets || []); setWalletsLoading(false); })
      .catch(() => { if (alive) setWalletsLoading(false); });
    return () => { alive = false; };
  }, []);

  return (
    <div className="wallets-page">
      <header className="page-header">
        <h1>DeFi</h1>
      </header>

      <div className="mnemonic-actions" style={{ marginBottom: '20px', borderBottom: `1px solid var(--border)`, paddingBottom: '12px' }}>
        {TABS.map((t) => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            style={tab === t.id ? { background: 'var(--accent)', color: 'white' } : { background: 'var(--bg-secondary)', color: 'var(--text-primary)' }}
          >
            {t.label}
          </button>
        ))}
      </div>

      {walletsLoading && <p>Loading wallets…</p>}

      {tab === 'lending' && <LendingSection wallets={wallets} isDark={isDark} />}
      {tab === 'copy' && <CopySection isDark={isDark} />}
      {tab === 'dao' && <DaoSection isDark={isDark} />}
      {tab === 'perpetual' && <PerpetualSection isDark={isDark} />}
      {tab === 'margin' && <MarginSection isDark={isDark} />}
      {tab === 'prediction' && <PredictionSection isDark={isDark} />}
      {tab === 'launchpool' && <LaunchpoolSection wallets={wallets} isDark={isDark} />}
      {tab === 'tokensales' && <TokenSalesSection isDark={isDark} />}
    </div>
  );
}

export default DeFi;
