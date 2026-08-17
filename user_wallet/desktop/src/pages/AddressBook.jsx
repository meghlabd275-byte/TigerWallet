import React, { useState, useEffect } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { api } from '../services/api';

const CHAIN_OPTIONS = [
  { value: 'ethereum', label: 'Ethereum', id: 1 },
  { value: 'bsc', label: 'BNB Chain', id: 56 },
  { value: 'polygon', label: 'Polygon', id: 137 },
];
const CHAIN_IDS = { ethereum: 1, bsc: 56, polygon: 137 };

const EMPTY = { name: '', address: '', chainId: 'ethereum' };

function AddressBook() {
  const { theme } = useTheme();
  const isDark = theme === 'dark';

  const [contacts, setContacts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [form, setForm] = useState(EMPTY);
  const [editingId, setEditingId] = useState(null);
  const [busy, setBusy] = useState(false);
  const [info, setInfo] = useState('');

  useEffect(() => {
    let alive = true;
    setLoading(true);
    api.getAddressBookContacts()
      .then((data) => { if (alive) { setContacts(data.contacts || []); setLoading(false); } })
      .catch((err) => { if (alive) { setError(err.message || 'Failed to load contacts'); setLoading(false); } });
    return () => { alive = false; };
  }, []);

  const resetForm = () => { setForm(EMPTY); setEditingId(null); };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setInfo('');
    if (!form.name.trim() || !form.address.trim()) { setError('Name and address are required'); return; }
    setBusy(true);
    try {
      const body = { name: form.name.trim(), address: form.address.trim(), chainId: CHAIN_IDS[form.chainId] || 1 };
      if (editingId) {
        await api.updateContact(editingId, body);
        setInfo('Contact updated.');
      } else {
        await api.addContact(body);
        setInfo('Contact added.');
      }
      resetForm();
      const data = await api.getAddressBookContacts();
      setContacts(data.contacts || []);
    } catch (err) {
      setError(err.message || 'Failed to save contact');
    } finally {
      setBusy(false);
    }
  };

  const startEdit = (c) => {
    setEditingId(c.id || c.contact_id || c.contactId);
    setForm({
      name: c.name || '',
      address: c.address || '',
      chainId: c.chain_id === 56 ? 'bsc' : c.chain_id === 137 ? 'polygon' : 'ethereum',
    });
    setInfo('');
    setError('');
  };

  const handleDelete = async (c) => {
    setError('');
    setInfo('');
    if (!window.confirm(`Delete contact "${c.name}"?`)) return;
    try {
      await api.deleteContact(c.id || c.contact_id || c.contactId);
      const data = await api.getAddressBookContacts();
      setContacts(data.contacts || []);
      setInfo('Contact deleted.');
    } catch (err) {
      setError(err.message || 'Failed to delete contact');
    }
  };

  return (
    <div className="wallets-page">
      <header className="page-header">
        <h1>Address Book</h1>
      </header>

      {error && <div className="error">{error}</div>}
      {info && <div className="success-banner" style={{ marginBottom: '16px' }}><h3 style={{ color: isDark ? '#4CAF50' : 'var(--accent)' }}>✓ {info}</h3></div>}

      <form className="import-form" style={{ maxWidth: '600px' }} onSubmit={handleSubmit}>
        <label style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>{editingId ? 'Edit contact' : 'Add contact'}</label>
        <input placeholder="Name" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required />
        <input placeholder="Address (0x...)" value={form.address} onChange={(e) => setForm({ ...form, address: e.target.value })} required />
        <select value={form.chainId} onChange={(e) => setForm({ ...form, chainId: e.target.value })}>
          {CHAIN_OPTIONS.map((c) => <option key={c.value} value={c.value}>{c.label}</option>)}
        </select>
        <div className="mnemonic-actions">
          <button type="submit" disabled={busy}>{busy ? 'Saving…' : editingId ? 'Update' : 'Add'}</button>
          {editingId && <button type="button" className="link-btn" onClick={resetForm}>Cancel</button>}
        </div>
      </form>

      {loading ? (
        <p>Loading...</p>
      ) : contacts.length === 0 ? (
        <p>No contacts yet. Add one above.</p>
      ) : (
        <div className="wallets-grid" style={{ marginTop: '20px' }}>
          {contacts.map((c, idx) => (
            <div key={c.id || c.contact_id || idx} className="wallet-card">
              <h3>{c.name}</h3>
              <p className="network">Chain #{c.chain_id}</p>
              <p className="address">{c.address}</p>
              <div className="mnemonic-actions" style={{ marginTop: '12px' }}>
                <button onClick={() => startEdit(c)}>✏️ Edit</button>
                <button onClick={() => handleDelete(c)}>🗑️ Delete</button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default AddressBook;
