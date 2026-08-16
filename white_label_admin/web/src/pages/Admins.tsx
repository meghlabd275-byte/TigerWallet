/**
 * Admins page — WL client manages scoped sub-admins.
 * The WL client can add/edit/remove admins and assign any of the 13 scoped
 * roles (trading_admin, p2p_admin, bot_admin, listing_admin, liquidity_admin,
 * wallet_admin, customer_service_admin, marketing_admin, kyc_admin, card_admin,
 * reward_admin, security_admin, compliance_admin). wl_client = the WL owner
 * (implicit; full tenancy control).
 */

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../context/ThemeContext';
import { whiteLabelAdminApi as api } from '../services/api';

interface Admin {
  id: string;
  username: string;
  email: string;
  role: string;
  scopes: string[];
  is_active: boolean;
  created_at: string;
  last_login: string;
}

interface ScopeInfo {
  scope: string;
  label: string;
}

const SCOPE_LABELS: Record<string, string> = {
  wl_client: 'WL Client Owner (full tenancy control)',
  trading_admin: 'Trading (futures, margin, options, copy, convert)',
  p2p_admin: 'P2P & Fiat (p2p, on-ramp, off-ramp, merchant)',
  bot_admin: 'Bots (all WL bots)',
  listing_admin: 'Listing (coin/token, trading pairs, listingManager)',
  liquidity_admin: 'Liquidity (all liquidity sources)',
  wallet_admin: 'Wallets (MasterWallet + UserWallet)',
  customer_service_admin: 'Customer Service (tickets, support)',
  marketing_admin: 'Marketing & Promotion',
  kyc_admin: 'KYC (review)',
  card_admin: 'WL-Branded CryptoCard',
  reward_admin: 'Reward System',
  security_admin: 'Security (WL client only)',
  compliance_admin: 'Compliance (audit, reports, SLA)',
};

export default function Admins() {
  const { theme } = useTheme();
  const isDark = theme === 'dark';
  const [admins, setAdmins] = useState<Admin[]>([]);
  const [scopes, setScopes] = useState<ScopeInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showAdd, setShowAdd] = useState(false);
  const [editing, setEditing] = useState<Admin | null>(null);
  const [addForm, setAddForm] = useState({ username: '', email: '', password: '' });
  const [editScopes, setEditScopes] = useState<string[]>([]);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [adminsResp, scopesResp] = await Promise.all([
        api.getAdmins(),
        api.getScopes(),
      ]);
      setAdmins(adminsResp.admins || []);
      setScopes(scopesResp.scopes || []);
    } catch (e: any) {
      setError(e.message || 'Failed to load admins');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleAdd = async () => {
    try {
      await api.createAdmin(addForm);
      setShowAdd(false);
      setAddForm({ username: '', email: '', password: '' });
      load();
    } catch (e: any) {
      setError(e.message || 'Failed to create admin');
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this admin? This cannot be undone.')) return;
    try {
      await api.deleteAdmin(id);
      load();
    } catch (e: any) {
      setError(e.message || 'Failed to delete admin');
    }
  };

  const handleSuspend = async (id: string) => {
    try { await api.suspendAdmin(id); load(); }
    catch (e: any) { setError(e.message || 'Failed to suspend admin'); }
  };

  const handleActivate = async (id: string) => {
    try { await api.activateAdmin(id); load(); }
    catch (e: any) { setError(e.message || 'Failed to activate admin'); }
  };

  const openEdit = (admin: Admin) => {
    setEditing(admin);
    setEditScopes(admin.scopes || []);
  };

  const toggleScope = (scope: string) => {
    setEditScopes(prev => prev.includes(scope) ? prev.filter(s => s !== scope) : [...prev, scope]);
  };

  const handleSaveScopes = async () => {
    if (!editing) return;
    try {
      await api.updateAdmin(editing.id, { scopes: editScopes });
      setEditing(null);
      load();
    } catch (e: any) {
      setError(e.message || 'Failed to update admin scopes');
    }
  };

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className={`text-2xl font-bold ${cardText}`}>Admins & Roles</h1>
          <p className={muted}>Manage scoped sub-admins for your white-label products.</p>
        </div>
        <button
          onClick={() => setShowAdd(true)}
          className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
        >+ Add Admin</button>
      </div>

      {error && <div className={`mb-4 p-3 rounded ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading...</div>}

      {!loading && !error && (
        <div className={`rounded-lg border ${border} ${cardBg} overflow-hidden`}>
          <table className="w-full">
            <thead className={isDark ? 'bg-gray-700' : 'bg-gray-50'}>
              <tr>
                <th className={`px-4 py-3 text-left text-sm font-semibold ${cardText}`}>Username</th>
                <th className={`px-4 py-3 text-left text-sm font-semibold ${cardText}`}>Email</th>
                <th className={`px-4 py-3 text-left text-sm font-semibold ${cardText}`}>Role / Scopes</th>
                <th className={`px-4 py-3 text-left text-sm font-semibold ${cardText}`}>Status</th>
                <th className={`px-4 py-3 text-left text-sm font-semibold ${cardText}`}>Last Login</th>
                <th className={`px-4 py-3 text-right text-sm font-semibold ${cardText}`}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {admins.length === 0 && (
                <tr><td colSpan={6} className={`px-4 py-8 text-center ${muted}`}>No admins yet. Add your first sub-admin.</td></tr>
              )}
              {admins.map(a => (
                <tr key={a.id} className={`border-t ${border}`}>
                  <td className={`px-4 py-3 ${cardText}`}>{a.username}</td>
                  <td className={`px-4 py-3 ${cardText}`}>{a.email}</td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-1">
                      {a.role === 'wl_client' && (
                        <span className="px-2 py-0.5 text-xs rounded bg-purple-600 text-white">WL Client</span>
                      )}
                      {(a.scopes || []).map(s => (
                        <span key={s} className={`px-2 py-0.5 text-xs rounded ${isDark ? 'bg-blue-900 text-blue-200' : 'bg-blue-100 text-blue-800'}`}>
                          {s}
                        </span>
                      ))}
                      {(a.scopes || []).length === 0 && a.role !== 'wl_client' && (
                        <span className={muted}>no scopes</span>
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-0.5 text-xs rounded ${a.is_active ? (isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800') : (isDark ? 'bg-gray-700 text-gray-300' : 'bg-gray-200 text-gray-600')}`}>
                      {a.is_active ? 'Active' : 'Suspended'}
                    </span>
                  </td>
                  <td className={`px-4 py-3 text-sm ${muted}`}>{a.last_login ? new Date(a.last_login).toLocaleString() : '—'}</td>
                  <td className="px-4 py-3 text-right space-x-2">
                    <button onClick={() => openEdit(a)} className="text-blue-500 hover:underline text-sm">Edit Scopes</button>
                    {a.role !== 'wl_client' && a.is_active && (
                      <button onClick={() => handleSuspend(a.id)} className="text-yellow-500 hover:underline text-sm">Suspend</button>
                    )}
                    {a.role !== 'wl_client' && !a.is_active && (
                      <button onClick={() => handleActivate(a.id)} className="text-green-500 hover:underline text-sm">Activate</button>
                    )}
                    {a.role !== 'wl_client' && (
                      <button onClick={() => handleDelete(a.id)} className="text-red-500 hover:underline text-sm">Delete</button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Add admin modal */}
      {showAdd && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className={`${cardBg} rounded-lg p-6 w-full max-w-md`}>
            <h2 className={`text-lg font-bold mb-4 ${cardText}`}>Add Sub-Admin</h2>
            <div className="space-y-3">
              <input
                placeholder="Username"
                value={addForm.username}
                onChange={e => setAddForm({ ...addForm, username: e.target.value })}
                className={`w-full px-3 py-2 rounded border ${border} ${cardBg} ${cardText}`}
              />
              <input
                placeholder="Email"
                type="email"
                value={addForm.email}
                onChange={e => setAddForm({ ...addForm, email: e.target.value })}
                className={`w-full px-3 py-2 rounded border ${border} ${cardBg} ${cardText}`}
              />
              <input
                placeholder="Password (min 8 chars)"
                type="password"
                value={addForm.password}
                onChange={e => setAddForm({ ...addForm, password: e.target.value })}
                className={`w-full px-3 py-2 rounded border ${border} ${cardBg} ${cardText}`}
              />
            </div>
            <div className="flex justify-end gap-2 mt-4">
              <button onClick={() => setShowAdd(false)} className={`px-4 py-2 rounded ${isDark ? 'bg-gray-700 text-white' : 'bg-gray-200 text-gray-800'}`}>Cancel</button>
              <button onClick={handleAdd} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">Create</button>
            </div>
          </div>
        </div>
      )}

      {/* Edit scopes modal */}
      {editing && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className={`${cardBg} rounded-lg p-6 w-full max-w-lg max-h-[80vh] overflow-y-auto`}>
            <h2 className={`text-lg font-bold mb-1 ${cardText}`}>Edit Admin Scopes</h2>
            <p className={`text-sm mb-4 ${muted}`}>{editing.username} ({editing.email})</p>
            <div className="space-y-2">
              {scopes.filter(s => s.scope !== 'wl_client').map(s => (
                <label key={s.scope} className={`flex items-start gap-3 p-2 rounded border ${border} cursor-pointer`}>
                  <input
                    type="checkbox"
                    checked={editScopes.includes(s.scope)}
                    onChange={() => toggleScope(s.scope)}
                    className="mt-1"
                  />
                  <div>
                    <div className={`text-sm font-medium ${cardText}`}>{s.scope}</div>
                    <div className={`text-xs ${muted}`}>{s.label}</div>
                  </div>
                </label>
              ))}
            </div>
            <div className="flex justify-end gap-2 mt-4">
              <button onClick={() => setEditing(null)} className={`px-4 py-2 rounded ${isDark ? 'bg-gray-700 text-white' : 'bg-gray-200 text-gray-800'}`}>Cancel</button>
              <button onClick={handleSaveScopes} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">Save Scopes</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
