/**
 * Address Book Page - Saved contacts.
 *
 * Fetches contacts from the canonical backend (GET /address-book/contacts),
 * supports adding (POST /address-book/contacts) and deleting
 * (DELETE /address-book/contacts/:id) contacts. All calls go through
 * WalletService; no mock data.
 */

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { WalletService } from '../services/WalletService';
import LoadingSpinner from '../components/LoadingSpinner';

interface Contact {
  id: string;
  name: string;
  address: string;
  chain_id?: number;
  label?: string;
}

function AddressBookPage() {
  const { theme } = useTheme();
  const [walletService] = useState(() => new WalletService());

  const [contacts, setContacts] = useState<Contact[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Add-contact form state
  const [name, setName] = useState('');
  const [address, setAddress] = useState('');
  const [chainId, setChainId] = useState('');
  const [label, setLabel] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [submitMessage, setSubmitMessage] = useState<string | null>(null);

  const loadContacts = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = (await walletService.getAddressBookContacts()) as Contact[] | { contacts?: Contact[] };
      const list = Array.isArray(data) ? data : (data?.contacts ?? []);
      setContacts(
        (list ?? []).map((c) => ({
          id: String(c.id ?? ''),
          name: String(c.name ?? ''),
          address: String(c.address ?? ''),
          chain_id: typeof c.chain_id === 'number' ? c.chain_id : undefined,
          label: c.label ? String(c.label) : undefined,
        }))
      );
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Failed to load contacts');
      setContacts([]);
    } finally {
      setLoading(false);
    }
  }, [walletService]);

  useEffect(() => {
    loadContacts();
  }, [loadContacts]);

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSubmitMessage(null);

    if (!name.trim() || !address.trim()) {
      setError('Name and address are required');
      return;
    }

    setSubmitting(true);
    try {
      await walletService.addContact({
        name: name.trim(),
        address: address.trim(),
        chainId: chainId ? Number(chainId) : undefined,
      });
      setSubmitMessage('Contact added');
      setName('');
      setAddress('');
      setChainId('');
      setLabel('');
      await loadContacts();
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Failed to add contact');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id: string) => {
    setError(null);
    try {
      await walletService.deleteContact(id);
      await loadContacts();
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Failed to delete contact');
    }
  };

  const cardClass = `card mb-6 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`;
  const inputClass = `input w-full ${theme === 'dark' ? 'bg-slate-900 border-slate-700' : 'bg-white'}`;

  return (
    <div className="p-6 max-w-3xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">Address Book</h1>

      {error && (
        <div className={`card mb-6 ${theme === 'dark' ? 'bg-red-900/30' : 'bg-red-50'}`}>
          <p className="text-sm text-red-500">{error}</p>
        </div>
      )}

      {submitMessage && (
        <div className={`card mb-6 ${theme === 'dark' ? 'bg-green-900/30' : 'bg-green-50'}`}>
          <p className="text-sm text-green-500">{submitMessage}</p>
        </div>
      )}

      {/* Add contact form */}
      <form onSubmit={handleAdd} className={cardClass}>
        <h3 className="font-semibold mb-4">Add Contact</h3>

        <div className="mb-4">
          <label className="label">Name</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Jane Doe"
            className={inputClass}
            required
          />
        </div>

        <div className="mb-4">
          <label className="label">Address</label>
          <input
            type="text"
            value={address}
            onChange={(e) => setAddress(e.target.value)}
            placeholder="0x..."
            className={`${inputClass} font-mono text-sm`}
            required
          />
        </div>

        <div className="grid grid-cols-2 gap-4 mb-6">
          <div>
            <label className="label">Chain ID (optional)</label>
            <input
              type="number"
              value={chainId}
              onChange={(e) => setChainId(e.target.value)}
              placeholder="1"
              className={inputClass}
            />
          </div>
          <div>
            <label className="label">Label (optional)</label>
            <input
              type="text"
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              placeholder="Friend"
              className={inputClass}
            />
          </div>
        </div>

        <button type="submit" disabled={submitting} className="btn btn-primary w-full">
          {submitting ? 'Adding...' : 'Add Contact'}
        </button>
      </form>

      {/* Contacts list */}
      <h3 className="font-semibold mb-3">Saved Contacts</h3>
      {loading ? (
        <LoadingSpinner label="Loading contacts..." />
      ) : contacts.length === 0 ? (
        <div className={`card text-center py-12 opacity-60 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
          No contacts saved yet.
        </div>
      ) : (
        <div className="space-y-3">
          {contacts.map((c) => (
            <div key={c.id} className={`card ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
              <div className="flex justify-between items-start gap-3">
                <div className="min-w-0">
                  <div className="font-semibold flex items-center gap-2">
                    {c.name}
                    {c.label && (
                      <span className={`text-xs px-2 py-0.5 rounded ${theme === 'dark' ? 'bg-slate-700' : 'bg-gray-200'}`}>
                        {c.label}
                      </span>
                    )}
                  </div>
                  <p className="text-xs font-mono opacity-60 mt-1 truncate">{c.address}</p>
                  {c.chain_id !== undefined && (
                    <p className="text-xs opacity-40 mt-1">Chain ID: {c.chain_id}</p>
                  )}
                </div>
                <button
                  onClick={() => handleDelete(c.id)}
                  className="btn btn-secondary text-sm whitespace-nowrap"
                >
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default AddressBookPage;
