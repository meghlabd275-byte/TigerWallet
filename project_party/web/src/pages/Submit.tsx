// Submit Token Page - ProjectParty
import React, { useState } from 'react';
import { api } from '../services/api';

export default function Submit() {
  const [form, setForm] = useState({
    name: '', symbol: '', contract_address: '', network: 'ethereum',
    decimals: 18, total_supply: '', logo_url: '', website_url: '', description: ''
  });
  const [submitting, setSubmitting] = useState(false);
  const [message, setMessage] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      await api.submitToken(form);
      setMessage('Token submitted successfully! Pending approval.');
    } catch (err) {
      setMessage('Failed to submit token.');
    }
    setSubmitting(false);
  };

  return (
    <div className="submit-page">
      <h1>Submit Token</h1>
      {message && <p className="message">{message}</p>}
      <form onSubmit={handleSubmit}>
        <input placeholder="Token Name" value={form.name} onChange={e => setForm({...form, name: e.target.value})} required />
        <input placeholder="Symbol" value={form.symbol} onChange={e => setForm({...form, symbol: e.target.value})} required />
        <input placeholder="Contract Address" value={form.contract_address} onChange={e => setForm({...form, contract_address: e.target.value})} />
        <select value={form.network} onChange={e => setForm({...form, network: e.target.value})}>
          <option value="ethereum">Ethereum</option>
          <option value="bsc">BNB Chain</option>
          <option value="polygon">Polygon</option>
        </select>
        <input placeholder="Decimals" type="number" value={form.decimals} onChange={e => setForm({...form, decimals: parseInt(e.target.value)})} />
        <input placeholder="Total Supply" value={form.total_supply} onChange={e => setForm({...form, total_supply: e.target.value})} />
        <input placeholder="Logo URL" value={form.logo_url} onChange={e => setForm({...form, logo_url: e.target.value})} />
        <input placeholder="Website URL" value={form.website_url} onChange={e => setForm({...form, website_url: e.target.value})} />
        <textarea placeholder="Description" value={form.description} onChange={e => setForm({...form, description: e.target.value})} />
        <button type="submit" disabled={submitting}>{submitting ? 'Submitting...' : 'Submit'}</button>
      </form>
    </div>
  );
}
