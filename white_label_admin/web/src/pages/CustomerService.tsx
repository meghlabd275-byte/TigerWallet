/**
 * Customer Service (customer_service_admin) — support tickets, response times,
 * satisfaction. Backend: GET /api/v1/admin/tickets, PUT status, POST messages.
 */

import React, { useEffect, useMemo, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

export default function CustomerService() {
  const { isDark } = useTheme();
  const [tickets, setTickets] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selected, setSelected] = useState<any | null>(null);
  const [messages, setMessages] = useState<any[]>([]);
  const [reply, setReply] = useState('');

  useEffect(() => { load(); }, []);

  const load = async () => {
    setLoading(true); setError('');
    try {
      const data = await whiteLabelAdminApi.getTickets();
      setTickets(data.tickets || []);
    } catch (e: any) { setError(e.message || 'Failed to load tickets'); }
    finally { setLoading(false); }
  };

  const openTicket = async (t: any) => {
    setSelected(t); setMessages([]); setReply('');
    try {
      const d = await whiteLabelAdminApi.getTicket(t.id);
      setMessages(d.messages || []);
    } catch (e: any) { setError(e.message || 'Failed to load ticket thread'); }
  };

  const setStatus = async (id: string, status: string) => {
    try { await whiteLabelAdminApi.updateTicketStatus(id, status); load(); if (selected?.id === id) setSelected({ ...selected, status }); }
    catch (e: any) { setError(e.message || 'Failed to update ticket'); }
  };

  const sendReply = async () => {
    if (!selected || !reply.trim()) return;
    try { await whiteLabelAdminApi.addTicketMessage(selected.id, reply); setReply(''); openTicket(selected); }
    catch (e: any) { setError(e.message || 'Failed to send reply'); }
  };

  const stats = useMemo(() => {
    const open = tickets.filter((t) => t.status !== 'resolved' && t.status !== 'closed').length;
    const resolved = tickets.filter((t) => t.status === 'resolved' || t.status === 'closed').length;
    const rate = tickets.length ? Math.round((resolved / tickets.length) * 100) : 0;
    return { open, resolved, rate, total: tickets.length };
  }, [tickets]);

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';
  const thBg = isDark ? 'bg-gray-700' : 'bg-gray-50';
  const inputCls = `w-full px-3 py-2 rounded border ${border} ${cardBg} ${cardText}`;

  const badge = (s: string) => {
    const b = 'px-2 py-0.5 text-xs rounded';
    if (s === 'resolved' || s === 'closed') return `${b} ${isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800'}`;
    if (s === 'open') return `${b} ${isDark ? 'bg-blue-900 text-blue-200' : 'bg-blue-100 text-blue-800'}`;
    return `${b} ${isDark ? 'bg-yellow-900 text-yellow-200' : 'bg-yellow-100 text-yellow-800'}`;
  };

  return (
    <div className="p-6">
      <h1 className={`text-2xl font-bold mb-2 ${cardText}`}>Customer Service</h1>
      <p className={`mb-4 ${muted}`}>Support ticket queue, response times and resolution rate.</p>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
        <div className={`${cardBg} p-4 rounded-lg shadow border ${border}`}><p className={`text-xs ${muted}`}>Open</p><p className={`text-xl font-bold mt-1 ${cardText}`}>{stats.open}</p></div>
        <div className={`${cardBg} p-4 rounded-lg shadow border ${border}`}><p className={`text-xs ${muted}`}>Resolved</p><p className={`text-xl font-bold mt-1 ${cardText}`}>{stats.resolved}</p></div>
        <div className={`${cardBg} p-4 rounded-lg shadow border ${border}`}><p className={`text-xs ${muted}`}>Resolution Rate</p><p className={`text-xl font-bold mt-1 ${cardText}`}>{stats.rate}%</p></div>
        <div className={`${cardBg} p-4 rounded-lg shadow border ${border}`}><p className={`text-xs ${muted}`}>Total</p><p className={`text-xl font-bold mt-1 ${cardText}`}>{stats.total}</p></div>
      </div>

      {error && <div className={`mb-3 p-2 rounded text-sm ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading…</div>}

      {!loading && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border} lg:col-span-2`}>
            <table className="w-full">
              <thead className={thBg}><tr>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Title</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Type</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Priority</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Status</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Created</th>
              </tr></thead>
              <tbody className={`divide-y ${border}`}>
                {tickets.length === 0 && <tr><td colSpan={5} className={`px-6 py-8 text-center ${muted}`}>No tickets.</td></tr>}
                {tickets.map((t) => (
                  <tr key={t.id} className="cursor-pointer hover:opacity-80" onClick={() => openTicket(t)}>
                    <td className={`px-6 py-3 ${cardText}`}>{t.title}</td>
                    <td className={`px-6 py-3 ${muted}`}>{t.ticket_type}</td>
                    <td className="px-6 py-3"><span className={`px-2 py-0.5 text-xs rounded ${t.priority === 'high' ? (isDark ? 'bg-red-900 text-red-200' : 'bg-red-100 text-red-800') : (isDark ? 'bg-gray-700 text-gray-300' : 'bg-gray-200 text-gray-700')}`}>{t.priority}</span></td>
                    <td className="px-6 py-3"><span className={badge(t.status)}>{t.status}</span></td>
                    <td className={`px-6 py-3 text-xs ${muted}`}>{t.created_at ? new Date(t.created_at).toLocaleString() : '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className={`${cardBg} rounded-lg shadow border ${border} p-4`}>
            {!selected && <p className={muted}>Select a ticket to view details.</p>}
            {selected && (
              <>
                <h2 className={`font-semibold ${cardText}`}>{selected.title}</h2>
                <p className={`text-sm ${muted} mb-2`}>{selected.description || '—'}</p>
                <div className="flex gap-2 mb-3">
                  <button onClick={() => setStatus(selected.id, 'resolved')} className="text-green-600 hover:underline text-sm">Resolve</button>
                  <button onClick={() => setStatus(selected.id, 'closed')} className="text-gray-600 hover:underline text-sm">Close</button>
                </div>
                <div className={`space-y-2 mb-3 max-h-60 overflow-y-auto ${border} rounded p-2`}>
                  {messages.length === 0 && <p className={`text-xs ${muted}`}>No messages yet.</p>}
                  {messages.map((m) => (
                    <div key={m.id} className={`text-sm ${m.is_internal ? muted : cardText}`}><span className="font-mono text-xs">{String(m.created_by).substring(0, 6)}…</span>: {m.message}</div>
                  ))}
                </div>
                <textarea value={reply} onChange={e => setReply(e.target.value)} placeholder="Reply…" className={inputCls} rows={3} />
                <button onClick={sendReply} className="mt-2 w-full px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">Send</button>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
