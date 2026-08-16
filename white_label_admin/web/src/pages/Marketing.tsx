/**
 * Marketing (marketing_admin) — campaigns, promotions, notifications.
 * Backend: GET /api/v1/admin/notifications (campaigns proxy),
 * POST /api/v1/admin/notifications/send, POST /api/v1/admin/notifications/broadcast.
 */

import React, { useEffect, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

export default function Marketing() {
  const { isDark } = useTheme();
  const [notifications, setNotifications] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [msg, setMsg] = useState('');
  const [form, setForm] = useState({ title: '', message: '', notification_type: 'campaign', scope: 'broadcast' });
  const [adminId, setAdminId] = useState('');

  useEffect(() => { load(); }, []);

  const load = async () => {
    setLoading(true); setError('');
    try {
      const data = await whiteLabelAdminApi.getNotifications();
      setNotifications(data.notifications || []);
    } catch (e: any) { setError(e.message || 'Failed to load campaigns'); }
    finally { setLoading(false); }
  };

  const send = async () => {
    setMsg(''); setError('');
    if (!form.title || !form.message) { setError('Title and message required.'); return; }
    try {
      if (form.scope === 'broadcast') {
        const r = await whiteLabelAdminApi.broadcastNotification({ title: form.title, message: form.message, notification_type: form.notification_type });
        setMsg(`Broadcast sent to ${r.recipients ?? 'all'} recipients.`);
      } else {
        if (!adminId) { setError('Admin ID required for targeted send.'); return; }
        await whiteLabelAdminApi.sendNotification({ title: form.title, message: form.message, notification_type: form.notification_type, admin_id: adminId });
        setMsg('Notification sent to admin.');
      }
      setForm({ title: '', message: '', notification_type: 'campaign', scope: 'broadcast' }); setAdminId(''); load();
    } catch (e: any) { setError(e.message || 'Failed to send'); }
  };

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';
  const thBg = isDark ? 'bg-gray-700' : 'bg-gray-50';
  const inputCls = `w-full px-3 py-2 rounded border ${border} ${cardBg} ${cardText}`;

  return (
    <div className="p-6">
      <h1 className={`text-2xl font-bold mb-2 ${cardText}`}>Marketing</h1>
      <p className={`mb-4 ${muted}`}>Campaigns, promotions, notifications and broadcasts.</p>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className={`${cardBg} rounded-lg shadow border ${border} p-5`}>
          <h2 className={`text-lg font-semibold mb-4 ${cardText}`}>New Campaign</h2>
          {msg && <div className={`mb-3 p-2 rounded text-sm ${isDark ? 'bg-green-900/50 text-green-200' : 'bg-green-50 text-green-700'}`}>{msg}</div>}
          {error && <div className={`mb-3 p-2 rounded text-sm ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
          <div className="space-y-3">
            <input placeholder="Title" value={form.title} onChange={e => setForm({ ...form, title: e.target.value })} className={inputCls} />
            <textarea placeholder="Message" value={form.message} onChange={e => setForm({ ...form, message: e.target.value })} rows={3} className={inputCls} />
            <select value={form.notification_type} onChange={e => setForm({ ...form, notification_type: e.target.value })} className={inputCls}>
              <option value="campaign">Campaign</option>
              <option value="promotion">Promotion</option>
              <option value="announcement">Announcement</option>
              <option value="reward">Reward</option>
            </select>
            <select value={form.scope} onChange={e => setForm({ ...form, scope: e.target.value })} className={inputCls}>
              <option value="broadcast">Broadcast (all admins)</option>
              <option value="targeted">Targeted (single admin)</option>
            </select>
            {form.scope === 'targeted' && <input placeholder="Admin ID" value={adminId} onChange={e => setAdminId(e.target.value)} className={inputCls} />}
            <button onClick={send} className="w-full px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">Send</button>
          </div>
        </div>

        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border} lg:col-span-2`}>
          <h2 className={`px-6 py-3 text-sm font-semibold ${cardText} ${thBg}`}>Sent Notifications</h2>
          {loading && <div className={`p-6 ${muted}`}>Loading…</div>}
          {!loading && (
            <table className="w-full">
              <thead className={thBg}><tr>
                <th className={`px-6 py-2 text-left text-xs font-medium ${muted} uppercase`}>Title</th>
                <th className={`px-6 py-2 text-left text-xs font-medium ${muted} uppercase`}>Type</th>
                <th className={`px-6 py-2 text-left text-xs font-medium ${muted} uppercase`}>Status</th>
                <th className={`px-6 py-2 text-left text-xs font-medium ${muted} uppercase`}>Sent</th>
              </tr></thead>
              <tbody className={`divide-y ${border}`}>
                {notifications.length === 0 && <tr><td colSpan={4} className={`px-6 py-8 text-center ${muted}`}>No notifications yet.</td></tr>}
                {notifications.map((n) => (
                  <tr key={n.id}>
                    <td className={`px-6 py-3 ${cardText}`}>{n.title}<div className={`text-xs ${muted}`}>{n.message}</div></td>
                    <td className={`px-6 py-3 ${muted}`}>{n.notification_type}</td>
                    <td className="px-6 py-3"><span className={`px-2 py-0.5 text-xs rounded ${n.is_read ? (isDark ? 'bg-gray-700 text-gray-300' : 'bg-gray-200 text-gray-700') : (isDark ? 'bg-blue-900 text-blue-200' : 'bg-blue-100 text-blue-800')}`}>{n.is_read ? 'Read' : 'Unread'}</span></td>
                    <td className={`px-6 py-3 text-xs ${muted}`}>{n.created_at ? new Date(n.created_at).toLocaleString() : '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </div>
  );
}
