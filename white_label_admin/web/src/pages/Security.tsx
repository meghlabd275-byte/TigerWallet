/**
 * Security (security_admin, WL-client-only) — IP whitelist, banned users.
 * Backend: GET/POST /api/v1/admin/ip-whitelist, DELETE /:id,
 * GET /api/v1/admin/users, POST /users/:id/ban, /users/:id/unban,
 * /users/:id/suspend.
 */

import React, { useEffect, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

export default function Security() {
  const { isDark } = useTheme();
  const [ips, setIps] = useState<any[]>([]);
  const [users, setUsers] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [tab, setTab] = useState<'whitelist' | 'bans'>('whitelist');
  const [ipAddress, setIpAddress] = useState('');
  const [ipLabel, setIpLabel] = useState('');

  useEffect(() => { load(); }, []);

  const load = async () => {
    setLoading(true); setError('');
    try {
      const [i, u] = await Promise.all([
        whiteLabelAdminApi.getIPWhitelist(),
        whiteLabelAdminApi.getUsers(),
      ]);
      setIps(i.ip_whitelist || i.ips || i || []);
      setUsers(u.users || []);
    } catch (e: any) { setError(e.message || 'Failed to load security data'); }
    finally { setLoading(false); }
  };

  const addIP = async () => {
    if (!ipAddress) { setError('IP address required'); return; }
    try { await whiteLabelAdminApi.addIPWhitelist({ ip_address: ipAddress, label: ipLabel || undefined }); setIpAddress(''); setIpLabel(''); load(); }
    catch (e: any) { setError(e.message || 'Failed to add IP'); }
  };

  const removeIP = async (id: string) => {
    if (!confirm('Remove this IP from the whitelist?')) return;
    try { await whiteLabelAdminApi.removeIPWhitelist(id); load(); }
    catch (e: any) { setError(e.message || 'Failed to remove IP'); }
  };

  const ban = async (id: string) => {
    if (!confirm('Ban this user?')) return;
    try { await whiteLabelAdminApi.banUser(id); load(); }
    catch (e: any) { setError(e.message || 'Failed to ban user'); }
  };
  const unban = async (id: string) => {
    try { await whiteLabelAdminApi.unbanUser(id); load(); }
    catch (e: any) { setError(e.message || 'Failed to unban user'); }
  };
  const suspend = async (id: string) => {
    if (!confirm('Suspend this user?')) return;
    try { await whiteLabelAdminApi.suspendUser(id); load(); }
    catch (e: any) { setError(e.message || 'Failed to suspend user'); }
  };

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';
  const thBg = isDark ? 'bg-gray-700' : 'bg-gray-50';
  const inputCls = isDark ? 'bg-gray-700 text-white border-gray-600' : 'bg-white text-gray-900 border-gray-300';

  const stat = (l: string, v: string) => (
    <div className={`${cardBg} p-4 rounded-lg shadow border ${border}`}><p className={`text-xs ${muted}`}>{l}</p><p className={`text-xl font-bold mt-1 ${cardText}`}>{v}</p></div>
  );

  return (
    <div className="p-6">
      <h1 className={`text-2xl font-bold mb-2 ${cardText}`}>Security</h1>
      <p className={`mb-4 ${muted}`}>IP whitelist, banned and suspended users.</p>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
        {stat('Whitelisted IPs', String(ips.length))}
        {stat('Total Users', String(users.length))}
        {stat('Banned Users', String(users.filter((u) => u.status === 'banned').length))}
        {stat('Suspended Users', String(users.filter((u) => u.status === 'suspended').length))}
      </div>

      <div className="flex gap-2 mb-4">
        {(['whitelist', 'bans'] as const).map((t) => (
          <button key={t} onClick={() => setTab(t)} className={`px-3 py-1 rounded text-sm ${tab === t ? 'bg-blue-600 text-white' : (isDark ? 'bg-gray-700 text-gray-200' : 'bg-gray-200 text-gray-700')}`}>{t === 'whitelist' ? 'IP Whitelist' : 'Users'}</button>
        ))}
      </div>

      {error && <div className={`mb-3 p-2 rounded text-sm ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading…</div>}

      {!loading && tab === 'whitelist' && (
        <>
          <div className={`${cardBg} p-4 rounded-lg shadow border ${border} mb-4`}>
            <h2 className={`text-sm font-semibold mb-3 ${cardText}`}>Add IP to Whitelist</h2>
            <div className="flex flex-wrap gap-2">
              <input value={ipAddress} onChange={(e) => setIpAddress(e.target.value)} placeholder="IP address (e.g. 203.0.113.10)" className={`px-3 py-2 rounded border ${inputCls}`} />
              <input value={ipLabel} onChange={(e) => setIpLabel(e.target.value)} placeholder="Label (optional)" className={`px-3 py-2 rounded border ${inputCls}`} />
              <button onClick={addIP} className="px-4 py-2 rounded bg-blue-600 text-white">Add</button>
            </div>
          </div>

          <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
            <table className="w-full">
              <thead className={thBg}><tr>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>IP Address</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Label</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Added</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Action</th>
              </tr></thead>
              <tbody className={`divide-y ${border}`}>
                {ips.length === 0 && <tr><td colSpan={4} className={`px-6 py-8 text-center ${muted}`}>No whitelisted IPs.</td></tr>}
                {ips.map((ip) => (
                  <tr key={ip.id}>
                    <td className={`px-6 py-4 font-mono text-xs ${cardText}`}>{ip.ip_address}</td>
                    <td className={`px-6 py-4 ${muted}`}>{ip.label || '—'}</td>
                    <td className={`px-6 py-4 text-xs ${muted}`}>{ip.created_at ? new Date(ip.created_at).toLocaleString() : '—'}</td>
                    <td className="px-6 py-4"><button onClick={() => removeIP(ip.id)} className="text-red-600 hover:underline text-sm">Remove</button></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {!loading && tab === 'bans' && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}><tr>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>User</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Email</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Status</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Actions</th>
            </tr></thead>
            <tbody className={`divide-y ${border}`}>
              {users.length === 0 && <tr><td colSpan={4} className={`px-6 py-8 text-center ${muted}`}>No users.</td></tr>}
              {users.map((u) => (
                <tr key={u.id}>
                  <td className={`px-6 py-4 ${cardText}`}>{u.username}</td>
                  <td className={`px-6 py-4 ${muted}`}>{u.email}</td>
                  <td className="px-6 py-4"><span className={`px-2 py-0.5 text-xs rounded ${u.status === 'banned' ? (isDark ? 'bg-red-900 text-red-200' : 'bg-red-100 text-red-800') : u.status === 'suspended' ? (isDark ? 'bg-yellow-900 text-yellow-200' : 'bg-yellow-100 text-yellow-800') : (isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800')}`}>{u.status}</span></td>
                  <td className="px-6 py-4 flex gap-2">
                    {u.status === 'banned'
                      ? <button onClick={() => unban(u.id)} className="text-green-600 hover:underline text-sm">Unban</button>
                      : <button onClick={() => ban(u.id)} className="text-red-600 hover:underline text-sm">Ban</button>}
                    {u.status !== 'suspended' && u.status !== 'banned' && (
                      <button onClick={() => suspend(u.id)} className="text-yellow-600 hover:underline text-sm">Suspend</button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
