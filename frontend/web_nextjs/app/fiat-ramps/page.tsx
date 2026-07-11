'use client';

import React, { useState } from 'react';

interface FiatProvider { id: string; name: string; logo: string; fees: string; methods: string[]; }

const PROVIDERS: FiatProvider[] = [
  { id: 'banxa', name: 'Banxa', logo: '🅱️', fees: '2.5%', methods: ['Credit Card', 'Debit Card', 'Apple Pay', 'Google Pay', 'Bank Transfer'] },
  { id: 'moonpay', name: 'MoonPay', logo: '🌙', fees: '3.5%', methods: ['Credit Card', 'Debit Card', 'Apple Pay', 'Google Pay', 'Bank Transfer'] },
  { id: 'transak', name: 'Transak', logo: '🔄', fees: '2%', methods: ['Credit Card', 'Debit Card', 'Bank Transfer', 'SEPA'] },
  { id: 'simplex', name: 'Simplex', logo: '💳', fees: '3%', methods: ['Credit Card', 'Debit Card', 'Apple Pay'] },
];

export default function FiatRamps() {
  const [provider, setProvider] = useState('');
  const [fiatAmount, setFiatAmount] = useState('');
  const [cryptoAmount, setCryptoAmount] = useState('');
  const [crypto, setCrypto] = useState('ETH');
  const [method, setMethod] = useState('');
  const [loading, setLoading] = useState(false);

  const handleBuy = async () => {
    if (!provider || !fiatAmount || !method) return;
    setLoading(true);
    await new Promise(r => setTimeout(r, 2000));
    alert(`Redirecting to ${provider} to buy ${cryptoAmount} ${crypto} for $${fiatAmount}`);
    setLoading(false);
  };

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-white">
      <header className="bg-white dark:bg-slate-800 border-b p-4"><div className="flex items-center gap-4"><a href="/wallet" className="text-2xl">🐯</a><h1 className="text-xl font-bold">Buy Crypto</h1></div></header>
      <div className="max-w-2xl mx-auto p-8">
        <div className="bg-white dark:bg-slate-800 rounded-lg p-6 mb-6">
          <h2 className="font-semibold mb-4">Select Provider</h2>
          <div className="grid grid-cols-2 gap-4 mb-6">{PROVIDERS.map(p => <button key={p.id} onClick={() => setProvider(p.id)} className={`p-4 rounded-lg border-2 ${provider === p.id ? 'border-orange-500 bg-orange-50' : 'border-slate-200 dark:border-slate-700'}`}><div className="text-3xl mb-2">{p.logo}</div><div className="font-semibold">{p.name}</div><div className="text-xs text-slate-500">Fees: {p.fees}</div></button>)}</div>
          <div className="mb-4"><label className="block text-sm text-slate-500 mb-2">Amount ($)</label><input type="number" value={fiatAmount} onChange={(e) => setFiatAmount(e.target.value)} placeholder="100" className="w-full bg-slate-100 dark:bg-slate-700 rounded-lg px-4 py-3 text-xl" /></div>
          <div className="mb-4"><label className="block text-sm text-slate-500 mb-2">You Get</label><div className="flex gap-2"><select value={crypto} onChange={(e) => setCrypto(e.target.value)} className="bg-slate-100 dark:bg-slate-700 rounded-lg px-4 py-3"><option>ETH</option><option>BTC</option><option>USDT</option><option>BNB</option><option>SOL</option></select><input type="number" value={cryptoAmount} onChange={(e) => setCryptoAmount(e.target.value)} placeholder="0.0" className="flex-1 bg-slate-100 dark:bg-slate-700 rounded-lg px-4 py-3 text-xl" /></div></div>
          <div className="mb-6"><label className="block text-sm text-slate-500 mb-2">Payment Method</label><select value={method} onChange={(e) => setMethod(e.target.value)} className="w-full bg-slate-100 dark:bg-slate-700 rounded-lg px-4 py-3">{PROVIDERS.find(p => p.id === provider)?.methods.map(m => <option key={m}>{m}</option>)}</select></div>
          <button onClick={handleBuy} disabled={loading || !provider || !fiatAmount} className="w-full bg-orange-500 hover:bg-orange-600 disabled:bg-slate-400 text-white py-4 rounded-lg font-semibold">{loading ? 'Processing...' : 'Buy Now'}</button>
        </div>
      </div>
    </div>
  );
}
