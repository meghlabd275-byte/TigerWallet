import React, { useCallback, useEffect, useState } from 'react';
import api from '../services/api';

// Wallet & finance plane: multi-chain accounts, deterministic deposit
// addresses (QR + copy), signed withdrawals, instant convert, KYC-gated
// internal transfers, escrowed P2P marketplace, full ledger history.
// Every value comes from the canonical backend — no fabricated data.

interface Account { currency: string; balance: string; locked: string; available: string; usd_value?: number }
interface DepositAddr { asset: string; network: string; address: string; uri: string; assets: string[]; deposit_enabled: boolean }
interface PaymentMethod { code: string; name: string; kind: string; countries?: string[] }
interface EscrowOrder {
  id: string; seller_id: string; buyer_id?: string; currency: string; amount: string;
  fiat_currency: string; fiat_amount: string; payment_method_code: string;
  payment_method_name: string; payment_kind: string; country_code: string; status: string;
}

const ASSETS = ['BTC', 'ETH', 'USDT', 'USDC', 'BNB', 'SOL', 'TRX', 'MATIC', 'LTC', 'DOGE'];

// QrImage fetches the server-rendered QR PNG with the auth header (an <img>
// tag cannot send Authorization) and renders it as an object URL.
function QrImage({ asset }: { asset: string }) {
  const [url, setUrl] = useState('');
  useEffect(() => {
    let objectUrl = '';
    let mounted = true;
    (async () => {
      try {
        const token = api.getToken();
        const res = await fetch(api.depositQrUrl(asset, 200), {
          headers: token ? { Authorization: `Bearer ${token}` } : {},
        });
        if (!res.ok) return;
        objectUrl = URL.createObjectURL(await res.blob());
        if (mounted) setUrl(objectUrl);
      } catch { /* QR unavailable — address text still shown */ }
    })();
    return () => { mounted = false; if (objectUrl) URL.revokeObjectURL(objectUrl); };
  }, [asset]);
  if (!url) return null;
  return <img src={url} alt={`${asset} deposit QR`} width={128} height={128} />;
}

export default function Finance() {
  const [tab, setTab] = useState<'accounts' | 'deposit' | 'withdraw' | 'convert' | 'transfer' | 'p2p' | 'history'>('accounts');
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [addresses, setAddresses] = useState<DepositAddr[]>([]);
  const [history, setHistory] = useState<any[]>([]);
  const [withdrawals, setWithdrawals] = useState<any[]>([]);
  const [rates, setRates] = useState<any[]>([]);
  const [orders, setOrders] = useState<EscrowOrder[]>([]);
  const [methods, setMethods] = useState<PaymentMethod[]>([]);
  const [catalogInfo, setCatalogInfo] = useState({ total_methods: 0, total_countries: 0 });
  const [msg, setMsg] = useState('');
  const [err, setErr] = useState('');

  const load = useCallback(async () => {
    setErr('');
    try {
      const [a, d, h, w, r, o, pm] = await Promise.all([
        api.getFinanceAccounts().catch(() => ({ accounts: [] })),
        api.getDepositAddresses().catch(() => ({ addresses: [] })),
        api.getFinanceHistory().catch(() => ({ history: [] })),
        api.getWithdrawals().catch(() => ({ withdrawals: [] })),
        api.getConvertRates().catch(() => ({ rates: [] })),
        api.getEscrowOrders().catch(() => ({ orders: [] })),
        api.getPaymentMethods().catch(() => ({ methods: [], total_methods: 0, total_countries: 0 })),
      ]);
      setAccounts(a.accounts || []);
      setAddresses(d.addresses || []);
      setHistory(h.history || []);
      setWithdrawals(w.withdrawals || []);
      setRates(r.rates || []);
      setOrders((o.orders || []) as EscrowOrder[]);
      setMethods(pm.methods || []);
      setCatalogInfo({ total_methods: pm.total_methods, total_countries: pm.total_countries });
    } catch (e: unknown) {
      setErr(e instanceof Error ? e.message : 'Failed to load finance data');
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const flash = (m: string) => { setMsg(m); setTimeout(() => setMsg(''), 6000); };
  const fail = (e: unknown) => setErr(e instanceof Error ? e.message : 'Request failed');

  const copy = async (text: string) => {
    try { await navigator.clipboard.writeText(text); flash('Address copied'); } catch { /* clipboard unavailable */ }
  };

  // ---- Withdraw ----
  const [wCur, setWCur] = useState('BTC');
  const [wAmt, setWAmt] = useState('');
  const [wTo, setWTo] = useState('');
  const submitWithdraw = async () => {
    setErr('');
    try {
      const res = await api.createWithdrawal({ currency: wCur, amount: wAmt, to_address: wTo });
      flash(res.status === 'auto_approved'
        ? `Withdrawal auto-approved in ${res.approved_in_ms ?? '<1000'}ms`
        : 'Withdrawal queued for superadmin sign-off');
      setWAmt(''); setWTo('');
      load();
    } catch (e) { fail(e); }
  };

  // ---- Convert ----
  const [cFrom, setCFrom] = useState('USDT');
  const [cTo, setCTo] = useState('BTC');
  const [cAmt, setCAmt] = useState('');
  const submitConvert = async () => {
    setErr('');
    try {
      const res = await api.convert({ from_currency: cFrom, to_currency: cTo, amount: cAmt });
      flash(`Converted ${res.from_amount} ${res.from_currency} → ${res.to_amount} ${res.to_currency} @ ${res.rate}`);
      setCAmt('');
      load();
    } catch (e) { fail(e); }
  };

  // ---- Internal transfer ----
  const [tTo, setTTo] = useState('');
  const [tCur, setTCur] = useState('USDT');
  const [tAmt, setTAmt] = useState('');
  const submitTransfer = async () => {
    setErr('');
    try {
      await api.financeTransfer({ to_email: tTo, currency: tCur, amount: tAmt });
      flash('Transfer completed (atomic, KYC-gated)');
      setTTo(''); setTAmt('');
      load();
    } catch (e) { fail(e); }
  };

  // ---- P2P escrow ----
  const [pCur, setPCur] = useState('USDT');
  const [pAmt, setPAmt] = useState('');
  const [pFiat, setPFiat] = useState('USD');
  const [pFiatAmt, setPFiatAmt] = useState('');
  const [pMethod, setPMethod] = useState('');
  const [pCountry, setPCountry] = useState('US');
  const [disputeReason, setDisputeReason] = useState('');
  const submitOpenEscrow = async () => {
    setErr('');
    try {
      await api.openEscrow({
        currency: pCur, amount: pAmt, fiat_currency: pFiat, fiat_amount: pFiatAmt,
        payment_method_code: pMethod, country_code: pCountry,
      });
      flash('Escrow order opened — funds locked');
      setPAmt(''); setPFiatAmt('');
      load();
    } catch (e) { fail(e); }
  };
  const act = async (id: string, action: 'accept' | 'paid' | 'release' | 'dispute' | 'cancel') => {
    setErr('');
    try {
      await api.escrowAction(id, action, action === 'dispute' ? { reason: disputeReason || 'disputed by party' } : undefined);
      flash(`Escrow ${action} successful`);
      load();
    } catch (e) { fail(e); }
  };

  const tabs = [
    ['accounts', 'Accounts'], ['deposit', 'Deposit'], ['withdraw', 'Withdraw'],
    ['convert', 'Convert'], ['transfer', 'Transfer'], ['p2p', 'P2P Market'], ['history', 'History'],
  ] as const;

  return (
    <div className="page-container">
      <h2>Wallet &amp; Finance</h2>
      <div className="action-row" style={{ flexWrap: 'wrap', gap: 8, marginBottom: 16 }}>
        {tabs.map(([key, label]) => (
          <button key={key} className={tab === key ? 'primary-btn' : 'secondary-btn'} onClick={() => setTab(key)}>{label}</button>
        ))}
      </div>
      {msg && <div className="success-banner">{msg}</div>}
      {err && <div className="error-text">{err}</div>}

      {tab === 'accounts' && (
        <div className="record-list">
          {accounts.length === 0 && <div className="empty-state">No accounts yet</div>}
          {accounts.map(a => (
            <div key={a.currency} className="record-item">
              <strong>{a.currency}</strong>
              <span>Balance: {a.balance} · Locked: {a.locked} · Available: {a.available}</span>
              {a.usd_value !== undefined && <span className="muted">≈ ${a.usd_value.toFixed(2)}</span>}
            </div>
          ))}
        </div>
      )}

      {tab === 'deposit' && (
        <div className="record-list">
          {addresses.length === 0 && <div className="empty-state">Deposit addresses unavailable on this node</div>}
          {addresses.map(d => (
            <div key={d.asset} className="record-item" style={{ display: 'flex', gap: 16, alignItems: 'center' }}>
              <QrImage asset={d.asset} />
              <div>
                <strong>{d.asset}</strong> <span className="muted">({d.network} — {d.assets.join(', ')})</span>
                <div style={{ wordBreak: 'break-all', fontFamily: 'monospace', margin: '6px 0' }}>{d.address}</div>
                <div className="action-row">
                  <button className="secondary-btn" onClick={() => copy(d.address)}>Copy</button>
                  {!d.deposit_enabled && <span className="error-text">Deposits disabled</span>}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {tab === 'withdraw' && (
        <div>
          <div className="form-group">
            <label>Currency</label>
            <select value={wCur} onChange={e => setWCur(e.target.value)}>{ASSETS.map(a => <option key={a}>{a}</option>)}</select>
          </div>
          <div className="form-group"><label>Amount</label><input value={wAmt} onChange={e => setWAmt(e.target.value)} placeholder="0.0" /></div>
          <div className="form-group"><label>Destination address</label><input value={wTo} onChange={e => setWTo(e.target.value)} placeholder="bc1q… / 0x… / T…" /></div>
          <button className="primary-btn" onClick={submitWithdraw} disabled={!wAmt || !wTo}>Request withdrawal</button>
          <p className="muted">Every request is risk-scored and HMAC-signed. Below the auto threshold it is approved in under a second; larger or risky requests queue for superadmin sign-off. Rejections auto-refund.</p>
          <div className="record-list" style={{ marginTop: 16 }}>
            {withdrawals.map(w => (
              <div key={w.id} className="record-item">
                <strong>{w.amount} {w.currency}</strong> → <span style={{ wordBreak: 'break-all' }}>{w.to_address}</span>
                <span className="muted">risk {w.risk_score} · {w.status}{w.decision_reason ? ` · ${w.decision_reason}` : ''}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {tab === 'convert' && (
        <div>
          <div className="form-group"><label>From</label>
            <select value={cFrom} onChange={e => setCFrom(e.target.value)}>{ASSETS.map(a => <option key={a}>{a}</option>)}</select></div>
          <div className="form-group"><label>To</label>
            <select value={cTo} onChange={e => setCTo(e.target.value)}>{ASSETS.map(a => <option key={a}>{a}</option>)}</select></div>
          <div className="form-group"><label>Amount</label><input value={cAmt} onChange={e => setCAmt(e.target.value)} placeholder="0.0" /></div>
          <button className="primary-btn" onClick={submitConvert} disabled={!cAmt}>Convert instantly</button>
          <h4 style={{ marginTop: 16 }}>Admin-managed rates</h4>
          <div className="record-list">
            {rates.length === 0 && <div className="empty-state">No rates configured yet</div>}
            {rates.map(r => <div key={r.from_currency + r.to_currency} className="record-item">
              <strong>{r.from_currency}/{r.to_currency}</strong><span>{r.rate}</span>
            </div>)}
          </div>
        </div>
      )}

      {tab === 'transfer' && (
        <div>
          <div className="form-group"><label>Recipient email</label><input value={tTo} onChange={e => setTTo(e.target.value)} placeholder="user@example.com" /></div>
          <div className="form-group"><label>Currency</label>
            <select value={tCur} onChange={e => setTCur(e.target.value)}>{ASSETS.map(a => <option key={a}>{a}</option>)}</select></div>
          <div className="form-group"><label>Amount</label><input value={tAmt} onChange={e => setTAmt(e.target.value)} placeholder="0.0" /></div>
          <button className="primary-btn" onClick={submitTransfer} disabled={!tTo || !tAmt}>Send (KYC-gated, atomic)</button>
          <p className="muted">Internal P2P transfers settle atomically on the double-entry ledger. Both parties must be KYC-verified.</p>
        </div>
      )}

      {tab === 'p2p' && (
        <div>
          <p className="muted">Escrowed marketplace — {catalogInfo.total_methods} local payment methods across {catalogInfo.total_countries} countries (bank + mobile).</p>
          <h4>Open a sell order (escrow locks your funds)</h4>
          <div className="form-group"><label>Sell currency</label>
            <select value={pCur} onChange={e => setPCur(e.target.value)}>{ASSETS.map(a => <option key={a}>{a}</option>)}</select></div>
          <div className="form-group"><label>Amount</label><input value={pAmt} onChange={e => setPAmt(e.target.value)} placeholder="0.0" /></div>
          <div className="form-group"><label>Fiat currency</label><input value={pFiat} onChange={e => setPFiat(e.target.value.toUpperCase())} placeholder="USD" /></div>
          <div className="form-group"><label>Fiat amount</label><input value={pFiatAmt} onChange={e => setPFiatAmt(e.target.value)} placeholder="100.00" /></div>
          <div className="form-group"><label>Country</label><input value={pCountry} onChange={e => setPCountry(e.target.value.toUpperCase())} placeholder="US" maxLength={2} /></div>
          <div className="form-group"><label>Payment method</label>
            <select value={pMethod} onChange={e => setPMethod(e.target.value)}>
              <option value="">Select…</option>
              {methods.filter(m => !m.countries || m.countries.includes(pCountry)).map(m => (
                <option key={m.code} value={m.code}>{m.name} ({m.kind})</option>
              ))}
            </select></div>
          <button className="primary-btn" onClick={submitOpenEscrow} disabled={!pAmt || !pFiatAmt || !pMethod}>Open escrow order</button>

          <h4 style={{ marginTop: 16 }}>Marketplace</h4>
          <div className="record-list">
            {orders.length === 0 && <div className="empty-state">No open orders</div>}
            {orders.map(o => (
              <div key={o.id} className="record-item">
                <strong>{o.amount} {o.currency}</strong> for {o.fiat_amount} {o.fiat_currency}
                <span className="muted">{o.payment_method_name} · {o.country_code} · {o.status}</span>
                <div className="action-row">
                  {o.status === 'open' && <button className="secondary-btn" onClick={() => act(o.id, 'accept')}>Buy (accept)</button>}
                  {o.status === 'escrowed' && <button className="secondary-btn" onClick={() => act(o.id, 'paid')}>Mark paid</button>}
                  {o.status === 'paid' && <button className="primary-btn" onClick={() => act(o.id, 'release')}>Release escrow</button>}
                  {(o.status === 'escrowed' || o.status === 'paid') && (
                    <>
                      <input placeholder="Dispute reason" value={disputeReason} onChange={e => setDisputeReason(e.target.value)} />
                      <button className="secondary-btn" onClick={() => act(o.id, 'dispute')}>Dispute</button>
                    </>
                  )}
                  {o.status === 'open' && <button className="secondary-btn" onClick={() => act(o.id, 'cancel')}>Cancel (mine)</button>}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {tab === 'history' && (
        <div className="record-list">
          {history.length === 0 && <div className="empty-state">No ledger history yet</div>}
          {history.map(h => (
            <div key={h.id} className="record-item">
              <strong>{h.kind}</strong> <span>{h.direction === 'debit' ? '−' : '+'}{h.amount} {h.currency}</span>
              <span className="muted">balance after: {h.balance_after} · {new Date(h.created_at).toLocaleString()}{h.memo ? ` · ${h.memo}` : ''}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
