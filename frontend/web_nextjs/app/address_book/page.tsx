'use client';

import React, { useState } from 'react';

interface Contact {
  id: string;
  name: string;
  address: string;
  chain: string;
  symbol: string;
  notes: string;
  isFavorite: boolean;
  lastUsed: number;
}

const MOCK_CONTACTS: Contact[] = [
  { id: '1', name: 'My Main Wallet', address: '0x742d35Cc6634C0532925a3b844Bc9e7595f', chain: 'Ethereum', symbol: 'ETH', notes: 'Primary wallet', isFavorite: true, lastUsed: Date.now() - 3600000 },
  { id: '2', name: 'Cold Storage', address: '0x8Ba1f109551bD432803012645Ac136ddd64DBA72', chain: 'Ethereum', symbol: 'ETH', notes: 'Long-term holding', isFavorite: true, lastUsed: Date.now() - 86400000 * 7 },
  { id: '3', name: 'DeFi Pool', address: '0x1234567890AbCdEf1234567890AbCdEf12345678', chain: 'BNB Chain', symbol: 'BNB', notes: 'Staking pool', isFavorite: false, lastUsed: Date.now() - 86400000 * 2 },
  { id: '4', name: 'NFT Collector', address: '0xAaAaAaAaAaAaAaAaAaAaAaAaAaAaAaAaAaAaAaAaA', chain: 'Polygon', symbol: 'MATIC', notes: 'NFT purchases', isFavorite: false, lastUsed: Date.now() - 86400000 * 14 },
  { id: '5', name: 'Business Account', address: '0xBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB', chain: 'Solana', symbol: 'SOL', notes: 'Business transactions', isFavorite: true, lastUsed: Date.now() - 3600000 * 3 },
];

export default function AddressBookPage() {
  const [contacts, setContacts] = useState<Contact[]>(MOCK_CONTACTS);
  const [searchQuery, setSearchQuery] = useState('');
  const [showAddModal, setShowAddModal] = useState(false);
  const [editingContact, setEditingContact] = useState<Contact | null>(null);
  const [newContact, setNewContact] = useState<Partial<Contact>>({
    name: '',
    address: '',
    chain: 'Ethereum',
    symbol: 'ETH',
    notes: '',
    isFavorite: false,
  });

  const filteredContacts = contacts.filter(c => 
    c.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    c.address.toLowerCase().includes(searchQuery.toLowerCase()) ||
    c.chain.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const favorites = filteredContacts.filter(c => c.isFavorite);
  const regular = filteredContacts.filter(c => !c.isFavorite);

  const handleSave = () => {
    if (!newContact.name || !newContact.address) return;
    
    const contact: Contact = {
      id: Date.now().toString(),
      name: newContact.name,
      address: newContact.address,
      chain: newContact.chain || 'Ethereum',
      symbol: newContact.symbol || 'ETH',
      notes: newContact.notes || '',
      isFavorite: newContact.isFavorite || false,
      lastUsed: Date.now(),
    };
    
    setContacts(prev => [...prev, contact]);
    setNewContact({ name: '', address: '', chain: 'Ethereum', symbol: 'ETH', notes: '', isFavorite: false });
    setShowAddModal(false);
  };

  const handleDelete = (id: string) => {
    setContacts(prev => prev.filter(c => c.id !== id));
  };

  const toggleFavorite = (id: string) => {
    setContacts(prev => prev.map(c => 
      c.id === id ? { ...c, isFavorite: !c.isFavorite } : c
    ));
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

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900">
      <header className="bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700">
        <div className="max-w-4xl mx-auto px-4 py-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <a href="/" className="text-2xl">🐯</a>
              <div>
                <h1 className="text-xl font-bold">Address Book</h1>
                <p className="text-slate-500 text-sm">Manage your saved addresses</p>
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
            className="w-full px-4 py-3 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800"
          />
        </div>

        {/* Stats */}
        <div className="grid grid-cols-3 gap-4 mb-6">
          <div className="bg-white dark:bg-slate-800 rounded-lg p-4 border">
            <p className="text-sm text-slate-500">Total Contacts</p>
            <p className="text-2xl font-bold">{contacts.length}</p>
          </div>
          <div className="bg-white dark:bg-slate-800 rounded-lg p-4 border">
            <p className="text-sm text-slate-500">Favorites</p>
            <p className="text-2xl font-bold text-yellow-500">{favorites.length}</p>
          </div>
          <div className="bg-white dark:bg-slate-800 rounded-lg p-4 border">
            <p className="text-sm text-slate-500">Chains Used</p>
            <p className="text-2xl font-bold">{new Set(contacts.map(c => c.chain)).size}</p>
          </div>
        </div>

        {/* Favorites */}
        {favorites.length > 0 && (
          <div className="mb-8">
            <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
              <span className="text-yellow-500">⭐</span> Favorites
            </h3>
            <div className="space-y-3">
              {favorites.map(contact => (
                <div key={contact.id} className="bg-white dark:bg-slate-800 rounded-lg p-4 border border-yellow-200 dark:border-yellow-800">
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-1">
                        <h4 className="font-semibold">{contact.name}</h4>
                        <button onClick={() => toggleFavorite(contact.id)} className="text-yellow-500">⭐</button>
                      </div>
                      <p className="font-mono text-sm text-slate-500 mb-2">{contact.address.slice(0, 10)}...{contact.address.slice(-8)}</p>
                      <div className="flex items-center gap-3">
                        <span className={`px-2 py-1 rounded text-xs text-white ${getChainColor(contact.chain)}`}>
                          {contact.chain}
                        </span>
                        <span className="text-sm text-slate-500">{contact.symbol}</span>
                      </div>
                      {contact.notes && <p className="text-sm text-slate-400 mt-2">{contact.notes}</p>}
                    </div>
                    <div className="flex flex-col gap-2">
                      <button
                        onClick={() => copyAddress(contact.address)}
                        className="px-3 py-1 bg-slate-100 dark:bg-slate-700 rounded text-sm"
                      >
                        Copy
                      </button>
                      <button
                        onClick={() => setEditingContact(contact)}
                        className="px-3 py-1 bg-blue-100 dark:bg-blue-900/30 text-blue-600 rounded text-sm"
                      >
                        Edit
                      </button>
                      <button
                        onClick={() => handleDelete(contact.id)}
                        className="px-3 py-1 bg-red-100 dark:bg-red-900/30 text-red-600 rounded text-sm"
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
                <div key={contact.id} className="bg-white dark:bg-slate-800 rounded-lg p-4 border">
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-1">
                        <h4 className="font-semibold">{contact.name}</h4>
                        <button onClick={() => toggleFavorite(contact.id)} className="text-slate-400 hover:text-yellow-500">☆</button>
                      </div>
                      <p className="font-mono text-sm text-slate-500 mb-2">{contact.address.slice(0, 10)}...{contact.address.slice(-8)}</p>
                      <div className="flex items-center gap-3">
                        <span className={`px-2 py-1 rounded text-xs text-white ${getChainColor(contact.chain)}`}>
                          {contact.chain}
                        </span>
                        <span className="text-sm text-slate-500">{contact.symbol}</span>
                      </div>
                      {contact.notes && <p className="text-sm text-slate-400 mt-2">{contact.notes}</p>}
                    </div>
                    <div className="flex flex-col gap-2">
                      <button
                        onClick={() => copyAddress(contact.address)}
                        className="px-3 py-1 bg-slate-100 dark:bg-slate-700 rounded text-sm"
                      >
                        Copy
                      </button>
                      <button
                        onClick={() => handleDelete(contact.id)}
                        className="px-3 py-1 bg-red-100 dark:bg-red-900/30 text-red-600 rounded text-sm"
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

        {filteredContacts.length === 0 && (
          <div className="text-center py-12 text-slate-500">
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
          <div className="bg-white dark:bg-slate-800 rounded-xl p-6 max-w-md w-full mx-4">
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
