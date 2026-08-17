// Address Book Page — manage saved recipient contacts (real PG CRUD).
import React, { useState, useEffect } from 'react';
import { api } from '../services/api';

interface Contact { id: string; name: string; address: string; chain_id?: number; }

export default function AddressBook() {
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [loading, setLoading] = useState(true);
  const [name, setName] = useState('');
  const [address, setAddress] = useState('');
  const [chainId, setChainId] = useState(1);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const load = () => {
    setLoading(true);
    api.getAddressBookContacts().then((data) => {
      setContacts((data.contacts as Contact[]) || []);
    }).catch(() => setContacts([])).finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, []);

  const save = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    if (!/^0x[a-fA-F0-9]{40}$/.test(address)) { setError('Invalid address'); return; }
    setBusy(true);
    try {
      if (editingId) {
        await api.updateContact(editingId, { name, address, chainId });
      } else {
        await api.addContact({ name, address, chainId });
      }
      setName(''); setAddress(''); setEditingId(null);
      load();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Save failed');
    } finally {
      setBusy(false);
    }
  };

  const remove = async (id: string) => {
    if (!window.confirm('Delete this contact?')) return;
    setBusy(true);
    try { await api.deleteContact(id); load(); } catch (err: unknown) { setError(err instanceof Error ? err.message : 'Delete failed'); } finally { setBusy(false); }
  };

  const edit = (c: Contact) => {
    setEditingId(c.id); setName(c.name); setAddress(c.address); setChainId(c.chain_id || 1);
  };

  return (
    <div className="addressbook-page">
      <h1>Address Book</h1>
      {error && <div className="error">{error}</div>}
      <form onSubmit={save} className="contact-form">
        <input placeholder="Name" value={name} onChange={(e) => setName(e.target.value)} required />
        <input placeholder="0x…" value={address} onChange={(e) => setAddress(e.target.value)} required />
        <select value={chainId} onChange={(e) => setChainId(Number(e.target.value))}>
          <option value={1}>Ethereum</option>
          <option value={56}>BNB</option>
          <option value={137}>Polygon</option>
          <option value={42161}>Arbitrum</option>
          <option value={10}>Optimism</option>
          <option value={8453}>Base</option>
        </select>
        <button type="submit" disabled={busy}>{editingId ? 'Update' : 'Add'}</button>
        {editingId && <button type="button" onClick={() => { setEditingId(null); setName(''); setAddress(''); }}>Cancel</button>}
      </form>
      {loading ? <p>Loading…</p> : contacts.length === 0 ? <p>No saved contacts.</p> : (
        <ul className="contact-list">
          {contacts.map((c) => (
            <li key={c.id}>
              <div><strong>{c.name}</strong></div>
              <div className="mono small">{c.address}</div>
              <div className="small">Chain {c.chain_id}</div>
              <button onClick={() => edit(c)}>Edit</button>
              <button onClick={() => remove(c.id)}>Delete</button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
