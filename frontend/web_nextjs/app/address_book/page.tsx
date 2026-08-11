'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../components/ThemeProvider';
import { api, AddressBookEntry } from '@/lib/api/client';

export default function AddressBookPage() {
  const [contacts, setContacts] = useState<AddressBookEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [showAddModal, setShowAddModal] = useState(false);
  const [editingContact, setEditingContact] = useState<AddressBookEntry | null>(null);
  const [newContact, setNewContact] = useState<Partial<AddressBookEntry>>({
    name: '',
    address: '',
    chain: 'Ethereum',
    symbol: 'ETH',
    notes: '',
    isFavorite: false,
  });
  const { isDark } = useTheme();

  const fetchContacts = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const res = await api.getAddressBook();
      if (res.success && res.data) {
        setContacts(res.data);
      } else {
        setContacts([]);
        if (res.error) setError(res.error);
      }
    } catch (err: any) {
      setError(err?.message || 'Failed to load contacts');
      setContacts([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchContacts();
  }, [fetchContacts]);

  const filteredContacts = contacts.filter(c =>
    (c.name || '').toLowerCase().includes(searchQuery.toLowerCase()) ||
    (c.address || '').toLowerCase().includes(searchQuery.toLowerCase()) ||
    (c.chain || '').toLowerCase().includes(searchQuery.toLowerCase())
  );

  const favorites = filteredContacts.filter(c => c.isFavorite);
  const regular = filteredContacts.filter(c => !c.isFavorite);

  const handleSave = async () => {
    if (!newContact.name || !newContact.address) return;
    try {
      setError(null);
      const entry = {
        name: newContact.name,
        address: newContact.address,
        chain: newContact.chain || 'Ethereum',
        symbol: newContact.symbol || 'ETH',
        notes: newContact.notes || '',
        isFavorite: newContact.isFavorite || false,
      };
      const res = await api.addAddress(entry);
      if (res.success && res.data) {
        setContacts(prev => [...prev, res.data!]);
      } else {
        setError(res.error || 'Failed to add contact');
        return;
      }
      setNewContact({ name: '', address: '', chain: 'Ethereum', symbol: 'ETH', notes: '', isFavorite: false });
      setShowAddModal(false);
    } catch (err: any) {
      setError(err?.message || 'Failed to add contact');
    }
  };

  const handleDelete = async (id: string) => {
    try {
      const res = await api.deleteAddress(id);
      if (res.success) {
        setContacts(prev => prev.filter(c => c.id !== id));
      } else {
        setError(res.error || 'Failed to delete contact');
      }
    } catch (err: any) {
      setError(err?.message || 'Failed to delete contact');
    }
  };

  const toggleFavorite = async (contact: AddressBookEntry) => {
    try {
      const res = await api.updateAddress(contact.id, { ...contact, isFavorite: !contact.isFavorite });
      if (res.success && res.data) {
        setContacts(prev => prev.map(c => c.id === contact.id ? res.data! : c));
      } else {
        setError(res.error || 'Failed to update contact');
      }
    } catch (err: any) {
      setError(err?.message || 'Failed to update contact');
    }
  };

  const copyAddress = (address: string) => {
    navigator.clipboard.writeText(address);
    alert('Address copied to clipboard!');
  };

  const getChainColor = (chain: string) => {
    const colors: { [key: string]: string } = {
      'Ethereum': 'bg-blue-500',
      'BNB Chain': 'bg-yellow-500',
      'Polygon': 'bg-purple-500',
      'Solana': 'bg-gradient-to-r from-purple-500 to-orange-500',
      'Arbitrum': 'bg-blue-600',
      'Optimism': 'bg-red-500',
      'Avalanche': 'bg-red-600',
    };
    return colors[chain] || 'bg-gray-500';
  };

  const formatAddr = (address: string) =>
    address && address.length > 18 ? `${address.slice(0, 10)}...${address.slice(-8)}` : address;

  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900 text-white' : 'bg-gray-50 text-gray-900'}`}>
      <header className={`${isDark ? 'bg-gray-800' : 'bg-white border border-gray-200'} border-b ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
        <div className="max-w-4xl mx-auto px-4 py-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <a href="/" className="text-2xl">🐯</a>
              <div>
                <h1 className="text-xl font-bold">Address Book</h1>
                <p className={`${isDark ? 'text-gray-400' : 'text-gray-500'} text-sm`}>Manage your saved addresses</p>
              </div>
            </div>
            <button
              onClick={() => setShowAddModal(true)}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
            >
              + Add Contact
            </button>
          </div>
        </div>
      </header>

      <div className="max-w-4xl mx-auto px-4 py-6">
        {/* Search */}
        <div className="mb-6">
          <input
            type="text"
            placeholder="Search by name, address, or chain..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className={`w-full px-4 py-3 border ${isDark ? 'border-gray-600 bg-gray-800' : 'border-gray-300 bg-white'} rounded-lg`}
          />
        </div>

        {/* Stats */}
        <div className="grid grid-cols-3 gap-4 mb-6">
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg p-4 border ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Total Contacts</p>
            <p className="text-2xl font-bold">{contacts.length}</p>
          </div>
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg p-4 border ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Favorites</p>
            <p className="text-2xl font-bold text-yellow-500">{favorites.length}</p>
          </div>
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg p-4 border ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Chains Used</p>
            <p className="text-2xl font-bold">{new Set(contacts.map(c => c.chain)).size}</p>
          </div>
        </div>

        {loading && (
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white border border-gray-200'} rounded-lg p-12 text-center mb-6`}>
            <p className="text-xl font-semibold">Loading contacts…</p>
          </div>
        )}
        {error && !loading && (
          <div className={`${isDark ? 'bg-red-900 text-red-200' : 'bg-red-100 text-red-700'} rounded-lg p-6 text-center mb-6`}>
            <p className="font-semibold">{error}</p>
            <button onClick={fetchContacts} className="mt-3 px-4 py-2 bg-red-600 text-white rounded-lg">Retry</button>
          </div>
        )}

        {/* Favorites */}
        {favorites.length > 0 && (
          <div className="mb-8">
            <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
              <span className="text-yellow-500">⭐</span> Favorites
            </h3>
            <div className="space-y-3">
              {favorites.map(contact => (
                <div key={contact.id} className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg p-4 border ${isDark ? 'border-yellow-800' : 'border-yellow-200'}`}>
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-1">
                        <h4 className="font-semibold">{contact.name}</h4>
                        <button onClick={() => toggleFavorite(contact)} className="text-yellow-500">⭐</button>
                      </div>
                      <p className={`font-mono text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'} mb-2`}>{formatAddr(contact.address)}</p>
                      <div className="flex items-center gap-3">
                        <span className={`px-2 py-1 rounded text-xs text-white ${getChainColor(contact.chain)}`}>
                          {contact.chain}
                        </span>
                        <span className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{contact.symbol}</span>
                      </div>
                      {contact.notes && <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'} mt-2`}>{contact.notes}</p>}
                    </div>
                    <div className="flex flex-col gap-2">
                      <button
                        onClick={() => copyAddress(contact.address)}
                        className={`px-3 py-1 ${isDark ? 'bg-gray-700' : 'bg-gray-100'} rounded text-sm`}
                      >
                        Copy
                      </button>
                      <button
                        onClick={() => setEditingContact(contact)}
                        className={`px-3 py-1 ${isDark ? 'bg-blue-900/30 text-blue-400' : 'bg-blue-100 text-blue-600'} rounded text-sm`}
                      >
                        Edit
                      </button>
                      <button
                        onClick={() => handleDelete(contact.id)}
                        className={`px-3 py-1 ${isDark ? 'bg-red-900/30 text-red-400' : 'bg-red-100 text-red-600'} rounded text-sm`}
                      >
                        Delete
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Regular Contacts */}
        {regular.length > 0 && (
          <div>
            <h3 className="text-lg font-semibold mb-4">All Contacts</h3>
            <div className="space-y-3">
              {regular.map(contact => (
                <div key={contact.id} className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg p-4 border ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-1">
                        <h4 className="font-semibold">{contact.name}</h4>
                        <button onClick={() => toggleFavorite(contact)} className={`${isDark ? 'text-gray-400' : 'text-gray-400'} hover:text-yellow-500`}>☆</button>
                      </div>
                      <p className={`font-mono text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'} mb-2`}>{formatAddr(contact.address)}</p>
                      <div className="flex items-center gap-3">
                        <span className={`px-2 py-1 rounded text-xs text-white ${getChainColor(contact.chain)}`}>
                          {contact.chain}
                        </span>
                        <span className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{contact.symbol}</span>
                      </div>
                      {contact.notes && <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'} mt-2`}>{contact.notes}</p>}
                    </div>
                    <div className="flex flex-col gap-2">
                      <button
                        onClick={() => copyAddress(contact.address)}
                        className={`px-3 py-1 ${isDark ? 'bg-gray-700' : 'bg-gray-100'} rounded text-sm`}
                      >
                        Copy
                      </button>
                      <button
                        onClick={() => handleDelete(contact.id)}
                        className={`px-3 py-1 ${isDark ? 'bg-red-900/30 text-red-400' : 'bg-red-100 text-red-600'} rounded text-sm`}
                      >
                        Delete
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {!loading && !error && filteredContacts.length === 0 && (
          <div className={`text-center py-12 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
            <p className="text-lg">No contacts found</p>
            <button onClick={() => setShowAddModal(true)} className="mt-4 text-blue-600 hover:underline">
              Add your first contact
            </button>
          </div>
        )}
      </div>

      {/* Add Modal */}
      {showAddModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white border border-gray-200'} rounded-xl p-6 max-w-md w-full mx-4`}>
            <h3 className="text-xl font-bold mb-4">Add New Contact</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-2">Name *</label>
                <input
                  type="text"
                  value={newContact.name}
                  onChange={(e) => setNewContact({ ...newContact, name: e.target.value })}
                  placeholder="e.g., My Main Wallet"
                  className="w-full p-3 border rounded-lg"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">Address *</label>
                <input
                  type="text"
                  value={newContact.address}
                  onChange={(e) => setNewContact({ ...newContact, address: e.target.value })}
                  placeholder="0x..."
                  className="w-full p-3 border rounded-lg font-mono"
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-2">Chain</label>
                  <select
                    value={newContact.chain}
                    onChange={(e) => setNewContact({ ...newContact, chain: e.target.value })}
                    className="w-full p-3 border rounded-lg"
                  >
                    <option value="Ethereum">Ethereum</option>
                    <option value="BNB Chain">BNB Chain</option>
                    <option value="Polygon">Polygon</option>
                    <option value="Solana">Solana</option>
                    <option value="Arbitrum">Arbitrum</option>
                    <option value="Optimism">Optimism</option>
                    <option value="Avalanche">Avalanche</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-2">Symbol</label>
                  <input
                    type="text"
                    value={newContact.symbol}
                    onChange={(e) => setNewContact({ ...newContact, symbol: e.target.value })}
                    placeholder="ETH"
                    className="w-full p-3 border rounded-lg"
                  />
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">Notes</label>
                <textarea
                  value={newContact.notes}
                  onChange={(e) => setNewContact({ ...newContact, notes: e.target.value })}
                  placeholder="Optional notes..."
                  className="w-full p-3 border rounded-lg h-20"
                />
              </div>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={newContact.isFavorite}
                  onChange={(e) => setNewContact({ ...newContact, isFavorite: e.target.checked })}
                />
                <span>Add to favorites</span>
              </label>
              <div className="flex gap-4">
                <button onClick={() => setShowAddModal(false)} className="flex-1 py-3 bg-slate-200 rounded-lg">Cancel</button>
                <button onClick={handleSave} className="flex-1 py-3 bg-blue-600 text-white rounded-lg">Save Contact</button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
